package github

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// RealClient implements GitHubAPI using the actual GitHub API
type RealClient struct {
	client *github.Client
}

// NewRealClient creates a new GitHub API client with authentication
func NewRealClient() (*RealClient, error) {
	token, err := getGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	
	return &RealClient{
		client: github.NewClient(tc),
	}, nil
}

// NewRealClientWithToken creates a client with a provided token
func NewRealClientWithToken(token string) *RealClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	
	return &RealClient{
		client: github.NewClient(tc),
	}
}

// SearchCode implements GitHubAPI.SearchCode
func (c *RealClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
	// Convert our options to GitHub's format
	searchOpts := &github.SearchOptions{
		Sort:  opts.Sort,
		Order: opts.Order,
		ListOptions: github.ListOptions{
			Page:    opts.ListOptions.Page,
			PerPage: opts.ListOptions.PerPage,
		},
	}

	// Execute the search
	result, resp, err := c.client.Search.Code(ctx, query, searchOpts)
	if err != nil {
		return nil, formatGitHubError(err, resp)
	}

	// Convert the result to our format
	return convertSearchResults(result), nil
}

// GetFileContent implements GitHubAPI.GetFileContent
func (c *RealClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, formatGitHubError(err, resp)
	}

	if fileContent == nil {
		return nil, fmt.Errorf("file content is nil")
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return []byte(content), nil
}

// GetRateLimit implements GitHubAPI.GetRateLimit
func (c *RealClient) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	rateLimits, resp, err := c.client.RateLimits(ctx)
	if err != nil {
		return nil, formatGitHubError(err, resp)
	}

	// Return search-specific rate limits
	searchLimit := rateLimits.Search
	return &RateLimit{
		Limit:     searchLimit.Limit,
		Remaining: searchLimit.Remaining,
		Reset:     searchLimit.Reset.Time,
	}, nil
}

// Helper functions

// getGitHubToken retrieves GitHub token using gh CLI (same as ghx)
func getGitHubToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token from gh CLI: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("empty token from gh CLI")
	}

	return token, nil
}

// formatGitHubError converts GitHub API errors to our format with helpful messages
func formatGitHubError(err error, resp *github.Response) error {
	if err == nil {
		return nil
	}

	// Handle rate limiting
	if rateLimitErr, ok := err.(*github.RateLimitError); ok {
		resetTime := time.Until(rateLimitErr.Rate.Reset.Time)
		return &RateLimitError{
			Message:   "GitHub API rate limit exceeded",
			ResetTime: resetTime,
			Limit:     rateLimitErr.Rate.Limit,
			Remaining: rateLimitErr.Rate.Remaining,
		}
	}

	// Handle abuse rate limiting
	if abuseErr, ok := err.(*github.AbuseRateLimitError); ok {
		return &AbuseRateLimitError{
			Message:    "GitHub API abuse detection triggered",
			RetryAfter: abuseErr.RetryAfter,
		}
	}

	// Handle validation errors
	if errorResp, ok := err.(*github.ErrorResponse); ok {
		switch errorResp.Response.StatusCode {
		case http.StatusUnauthorized:
			return &AuthenticationError{
				Message: "GitHub authentication required or invalid",
			}
		case http.StatusForbidden:
			return &AuthorizationError{
				Message: "Access forbidden - check repository permissions",
			}
		case http.StatusNotFound:
			return &NotFoundError{
				Message: "Resource not found - repository may be private or deleted",
			}
		case http.StatusUnprocessableEntity:
			return &ValidationError{
				Message: "Invalid query syntax",
				Errors:  extractValidationErrors(errorResp),
			}
		}
	}

	// Generic error
	return &APIError{
		Message: err.Error(),
	}
}

// convertSearchResults converts GitHub's search results to our format
func convertSearchResults(result *github.CodeSearchResult) *SearchResults {
	items := make([]SearchItem, len(result.CodeResults))
	
	for i, item := range result.CodeResults {
		items[i] = convertSearchItem(item)
	}

	return &SearchResults{
		Total:             result.Total,
		IncompleteResults: result.IncompleteResults,
		Items:             items,
	}
}

// convertSearchItem converts a single search result item
func convertSearchItem(item *github.CodeResult) SearchItem {
	var textMatches []TextMatch
	for _, match := range item.TextMatches {
		textMatches = append(textMatches, convertTextMatch(match))
	}

	return SearchItem{
		Name:        item.Name,
		Path:        item.Path,
		SHA:         item.SHA,
		HTMLURL:     item.HTMLURL,
		Repository:  convertRepository(item.Repository),
		TextMatches: textMatches,
	}
}

// convertRepository converts GitHub repository to our format
func convertRepository(repo *github.Repository) Repository {
	var owner *User
	if repo.Owner != nil {
		owner = &User{
			Login:     repo.Owner.Login,
			ID:        repo.Owner.ID,
			NodeID:    repo.Owner.NodeID,
			AvatarURL: repo.Owner.AvatarURL,
			HTMLURL:   repo.Owner.HTMLURL,
			Type:      repo.Owner.Type,
		}
	}

	var createdAt, updatedAt, pushedAt *time.Time
	if repo.CreatedAt != nil {
		createdAt = &repo.CreatedAt.Time
	}
	if repo.UpdatedAt != nil {
		updatedAt = &repo.UpdatedAt.Time
	}
	if repo.PushedAt != nil {
		pushedAt = &repo.PushedAt.Time
	}

	return Repository{
		ID:              repo.ID,
		NodeID:          repo.NodeID,
		Name:            repo.Name,
		FullName:        repo.FullName,
		Owner:           owner,
		Private:         repo.Private,
		HTMLURL:         repo.HTMLURL,
		Description:     repo.Description,
		Fork:            repo.Fork,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		PushedAt:        pushedAt,
		StargazersCount: repo.StargazersCount,
		WatchersCount:   repo.WatchersCount,
		ForksCount:      repo.ForksCount,
		Language:        repo.Language,
		DefaultBranch:   repo.DefaultBranch,
	}
}

// convertTextMatch converts GitHub text match to our format
func convertTextMatch(match *github.TextMatch) TextMatch {
	var matches []Match
	for _, m := range match.Matches {
		matches = append(matches, Match{
			Text:    m.Text,
			Indices: m.Indices,
		})
	}

	return TextMatch{
		ObjectURL:  match.ObjectURL,
		ObjectType: match.ObjectType,
		Property:   match.Property,
		Fragment:   match.Fragment,
		Matches:    matches,
	}
}

// extractValidationErrors extracts validation error details
func extractValidationErrors(errorResp *github.ErrorResponse) []string {
	var errors []string
	for _, err := range errorResp.Errors {
		if err.Message != "" {
			errors = append(errors, err.Message)
		}
	}
	return errors
}

// Custom error types for better error handling

// RateLimitError represents a GitHub API rate limit error
type RateLimitError struct {
	Message   string
	ResetTime time.Duration
	Limit     int
	Remaining int
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// AbuseRateLimitError represents a GitHub API abuse detection error
type AbuseRateLimitError struct {
	Message    string
	RetryAfter *time.Duration
}

func (e *AbuseRateLimitError) Error() string {
	return e.Message
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// AuthorizationError represents an authorization failure
type AuthorizationError struct {
	Message string
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// ValidationError represents a query validation error
type ValidationError struct {
	Message string
	Errors  []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("%s: %s", e.Message, strings.Join(e.Errors, ", "))
	}
	return e.Message
}

// APIError represents a generic API error
type APIError struct {
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}