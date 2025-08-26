package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// RealClient implements GitHubAPI using go-gh (GitHub CLI's library).
// It provides a production implementation that communicates with GitHub's API.
type RealClient struct {
	client *api.RESTClient
}

// NewRealClient creates a new GitHub API client using go-gh.
// It automatically handles authentication via gh auth token or GH_TOKEN env var
func NewRealClient() (*RealClient, error) {
	// go-gh automatically handles:
	// - Authentication (gh auth token, GH_TOKEN env var)
	// - Host configuration (GH_HOST, GH_ENTERPRISE_TOKEN)
	// - Request headers and API versioning

	// Create client with text-match header for code snippets
	opts := api.ClientOptions{
		Headers: map[string]string{
			"Accept": "application/vnd.github.text-match+json",
		},
	}

	client, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &RealClient{
		client: client,
	}, nil
}

// SearchCode implements GitHubAPI.SearchCode.
// It searches GitHub code with the provided query and returns matching results
func (c *RealClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
	// Build URL with query parameters
	params := url.Values{}
	params.Set("q", query)
	params.Set("per_page", strconv.Itoa(opts.ListOptions.PerPage))
	params.Set("page", strconv.Itoa(opts.ListOptions.Page))

	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}
	if opts.Order != "" {
		params.Set("order", opts.Order)
	}

	endpoint := fmt.Sprintf("search/code?%s", params.Encode())

	// Make the API call
	var result struct {
		TotalCount        int          `json:"total_count"`
		IncompleteResults bool         `json:"incomplete_results"`
		Items             []SearchItem `json:"items"`
	}

	// Use regular Get since go-gh doesn't support custom headers easily
	// The text-match header is nice to have but not required
	err := c.client.Get(endpoint, &result)
	if err != nil {
		return nil, formatGoGHError(err)
	}

	// Convert to our format
	searchResults := &SearchResults{
		Total:             &result.TotalCount,
		IncompleteResults: &result.IncompleteResults,
		Items:             result.Items,
	}

	// Enrich repository metadata unless explicitly skipped
	if !opts.SkipEnrichment {
		c.enrichRepositoryMetadata(ctx, searchResults)
	}

	return searchResults, nil
}

// GetFileContent implements GitHubAPI.GetFileContent.
// It retrieves the content of a file from a GitHub repository
func (c *RealClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, path)

	if ref != "" {
		endpoint = fmt.Sprintf("%s?ref=%s", endpoint, url.QueryEscape(ref))
	}

	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	err := c.client.Get(endpoint, &fileContent)
	if err != nil {
		return nil, formatGoGHError(err)
	}

	// Decode base64 content
	if fileContent.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file content: %w", err)
		}
		return decoded, nil
	}

	return []byte(fileContent.Content), nil
}

// GetRateLimit implements GitHubAPI.GetRateLimit.
// It returns the current API rate limit status for search operations
func (c *RealClient) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	var rateLimits struct {
		Resources struct {
			Core struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"core"`
			Search struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"search"`
		} `json:"resources"`
	}

	err := c.client.Get("rate_limit", &rateLimits)
	if err != nil {
		return nil, formatGoGHError(err)
	}

	// Return search-specific rate limits
	searchLimit := rateLimits.Resources.Search
	return &RateLimit{
		Limit:     searchLimit.Limit,
		Remaining: searchLimit.Remaining,
		Reset:     time.Unix(searchLimit.Reset, 0),
	}, nil
}

// enrichRepositoryMetadata fetches additional repository data if needed
func (c *RealClient) enrichRepositoryMetadata(ctx context.Context, results *SearchResults) {
	// Skip if results is nil or client is not properly initialized
	if results == nil || c == nil || c.client == nil {
		return
	}

	// Track which repos we've already enriched to avoid duplicates
	enriched := make(map[string]bool)

	for i := range results.Items {
		item := &results.Items[i]
		repoKey := fmt.Sprintf("%s/%s",
			item.Repository.GetOwnerLogin(),
			item.Repository.GetName())

		// Skip if already enriched or has star count
		if enriched[repoKey] || item.Repository.StargazersCount != nil {
			continue
		}

		// Fetch repository metadata
		var repoData Repository
		endpoint := fmt.Sprintf("repos/%s", repoKey)

		err := c.client.Get(endpoint, &repoData)
		if err != nil {
			// Don't fail the entire search if enrichment fails
			// Just skip this repository
			continue
		}

		// Update the repository data
		item.Repository = repoData
		enriched[repoKey] = true
	}
}

// formatGoGHError converts go-gh errors to our error types
func formatGoGHError(err error) error {
	if err == nil {
		return nil
	}

	// go-gh returns api.HTTPError for HTTP errors
	if httpErr, ok := err.(*api.HTTPError); ok {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized:
			return &AuthenticationError{Message: httpErr.Message}
		case http.StatusForbidden:
			// Check if it's a rate limit error
			if strings.Contains(strings.ToLower(httpErr.Message), "rate limit") {
				// Parse rate limit info from headers if available
				rateLimitErr := &RateLimitError{
					Message: httpErr.Message,
				}
				// Extract rate limit info from headers if available
				if httpErr.Headers != nil {
					if limitStr := httpErr.Headers.Get("X-RateLimit-Limit"); limitStr != "" {
						if limit, err := strconv.Atoi(limitStr); err == nil {
							rateLimitErr.Limit = limit
						}
					}
					if remainingStr := httpErr.Headers.Get("X-RateLimit-Remaining"); remainingStr != "" {
						if remaining, err := strconv.Atoi(remainingStr); err == nil {
							rateLimitErr.Remaining = remaining
						}
					}
					if resetStr := httpErr.Headers.Get("X-RateLimit-Reset"); resetStr != "" {
						if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
							resetTime := time.Unix(resetUnix, 0)
							rateLimitErr.ResetTime = time.Until(resetTime)
						}
					}
				}
				return rateLimitErr
			}
			return &AuthorizationError{Message: httpErr.Message}
		case http.StatusNotFound:
			return &NotFoundError{Message: httpErr.Message}
		case http.StatusUnprocessableEntity:
			return &ValidationError{
				Message: httpErr.Message,
				Errors:  []string{httpErr.Message},
			}
		case http.StatusTooManyRequests:
			return &AbuseRateLimitError{Message: httpErr.Message}
		default:
			return fmt.Errorf("GitHub API error (status %d): %s",
				httpErr.StatusCode, httpErr.Message)
		}
	}

	// Return the original error if it's not an HTTP error
	return err
}
