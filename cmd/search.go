package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/silouanwright/gh-scout/internal/github"
	"github.com/silouanwright/gh-scout/internal/search"
	"github.com/spf13/cobra"
)

var (
	// Global client for dependency injection (following gh-comment pattern)
	searchClient github.GitHubAPI
	// Mutex to protect concurrent client initialization
	searchClientMutex sync.Mutex
	// Rate limiter for search operations
	searchRateLimiter *github.RateLimiter

	// Search flags migrated from ghx
	searchLanguage  string
	searchRepo      []string
	searchFilename  string
	searchExtension string
	searchPath      string
	searchOwner     []string
	searchSize      string
	searchLimit     int
	searchPage      int // New: page-based pagination
	contextLines    int
	outputFormat    string
	outputFile      string // New: export to file
	pipe            bool
	minStars        int
	sort            string
	order           string
	liteMode        bool // --lite flag for lightweight results (saves API quota)

	// Batch search flags (Phase 2)
	batchRepos    []string // --repos flag for multiple repositories
	batchOrgs     []string // --orgs flag for multiple organizations
	aggregateMode bool     // --aggregate flag
	compareMode   bool     // --compare flag
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search <query> [flags]",
	Short: "Search GitHub code with intelligent filtering",
	Long: `Search GitHub's codebase to discover working examples and configurations.

Perfect for finding real-world usage patterns, configuration examples,
and best practices across millions of repositories.

Results include code context, repository information, and intelligent
ranking based on repository quality indicators.`,
	Example: `  # Configuration discovery workflows
  gh scout "tsconfig.json" --language json --limit 10
  gh scout "vite.config" --language javascript --context 30
  gh scout "dockerfile" --filename dockerfile --repo "**/react"

  # Page-based search (API efficient for large datasets)
  gh scout "config" --page 1 --limit 100        # Get first 100 results
  gh scout "config" --page 2 --limit 100        # Get next 100 results
  gh scout "config" --page 3 --limit 50         # Get results 201-250

  # Search by owner (works for both users and organizations)
  gh scout "eslint.config.js" --owner microsoft --language javascript    # Organization
  gh scout "func main" --owner torvalds --language c                     # Individual user
  gh scout "useState" --owner facebook --repo facebook/react             # Specific repo in org
  gh scout "const" --owner stitchfix --repo stitchfix/web-frontend       # Private org/repo
  gh scout "interface" --owner google --owner facebook --language typescript  # Multiple orgs

  # Topic-based workflow (find repos by topic, then search code)
  gh search repos --topic=react --stars=">1000" --json fullName | jq -r '.[].fullName' > react-repos.txt
  gh scout "useState" --repos $(cat react-repos.txt | tr '\n' ',')

  # Multi-repository batch searches (Phase 2)
  gh scout "docker-compose.yml" --repos microsoft/vscode,facebook/react,vercel/next.js --aggregate
  gh scout "tsconfig.json" --orgs microsoft,google,facebook --min-stars 500 --compare
  gh scout "webpack OR vite" --repos facebook/*,vercel/* --compare

  # Auto-pagination (less API efficient but convenient)
  gh scout "hooks" --limit 200                  # Automatically fetches 2 pages

  # Export results to files
  gh scout "config" --language json --output configs.md     # Markdown export
  gh scout "hooks" --pipe --output data.txt                 # Pipe format export

  # Pipe results for further processing
  gh scout "react hooks" --language typescript --pipe`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Validate input parameters to prevent panics
	if searchLimit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be greater than 0)", searchLimit)
	}
	// searchPage 0 means auto-pagination, negative values are invalid
	if searchPage < 0 {
		return fmt.Errorf("invalid page: %d (must be 0 or greater)", searchPage)
	}
	// Prevent integer overflow in pagination calculations
	const maxPage = 10000
	if searchPage > maxPage {
		return fmt.Errorf("page number too large (max: %d)", maxPage)
	}

	// Initialize client if not set (thread-safe)
	if searchClient == nil {
		searchClientMutex.Lock()
		// Double-check after acquiring lock
		if searchClient == nil {
			client, err := createGitHubClient()
			if err != nil {
				searchClientMutex.Unlock()
				return handleClientError(err)
			}
			searchClient = client
		}
		searchClientMutex.Unlock()
	}

	// Check if batch flags are used (Phase 2 functionality)
	if len(batchRepos) > 0 || len(batchOrgs) > 0 {
		return executeBatchRepoSearch(cmd.Context(), args)
	}

	// Build search query from args and flags (migrated from ghx)
	query := buildSearchQuery(args)

	if dryRun {
		fmt.Printf("Would search GitHub with query: %s\n", query)
		return nil
	}

	if verbose {
		fmt.Printf("Searching GitHub with query: %s\n", query)
	}

	// Execute search with error handling and timeout
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	// Add timeout for search operations (skip in tests to avoid conflicts with rate limiter)
	if !isTestEnvironment() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	results, err := executeSearch(ctx, query)
	if err != nil {
		return handleSearchError(err, query)
	}

	if verbose {
		fmt.Printf("Found %d results\n", len(results.Items))
	}

	// Process and output results
	return outputResults(results)
}

// buildSearchQuery constructs GitHub search query from args and flags using QueryBuilder
func buildSearchQuery(terms []string) string {
	// Use existing QueryBuilder to eliminate code duplication
	qb := search.NewQueryBuilder(terms)

	// Apply all CLI flags using the QueryBuilder methods
	if searchLanguage != "" {
		qb = qb.WithLanguage(searchLanguage)
	}

	if searchFilename != "" {
		qb = qb.WithFilename(searchFilename)
	}

	if searchExtension != "" {
		qb = qb.WithExtension(searchExtension)
	}

	if len(searchRepo) > 0 {
		qb = qb.WithRepositories(searchRepo)
	}

	if searchPath != "" {
		qb = qb.WithPath(searchPath)
	}

	if len(searchOwner) > 0 {
		qb = qb.WithOwners(searchOwner)
	}

	if searchSize != "" {
		qb = qb.WithSize(searchSize)
	}

	if minStars > 0 {
		qb = qb.WithMinStars(minStars)
	}

	return qb.Build()
}

// executeSearch performs the GitHub search with optional pagination
func executeSearch(ctx context.Context, query string) (*github.SearchResults, error) {
	// If user specified a page, use single-page mode (more API efficient)
	if searchPage > 0 {
		return executeSinglePageSearch(ctx, query)
	}

	// Otherwise, use automatic pagination for high limits (legacy behavior)
	return executeAutoPageSearch(ctx, query)
}

// executeSinglePageSearch fetches a specific page of results (API efficient)
func executeSinglePageSearch(ctx context.Context, query string) (*github.SearchResults, error) {
	// Cap limit to GitHub's max per page
	perPage := searchLimit
	if perPage > GitHubMaxResultsPerPage {
		perPage = GitHubMaxResultsPerPage
	}

	opts := &github.SearchOptions{
		Sort:  sort,
		Order: order,
		ListOptions: github.ListOptions{
			Page:    searchPage,
			PerPage: perPage,
		},
		SkipEnrichment: liteMode,
	}

	results, err := searchClient.SearchCode(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	if verbose {
		totalPages := 0
		if results.Total != nil && *results.Total > 0 {
			totalPages = (*results.Total + GitHubMaxResultsPerPage - 1) / GitHubMaxResultsPerPage
		}
		fmt.Printf("📄 Page %d of ~%d (%d results on this page)\n",
			searchPage, totalPages, len(results.Items))
	}

	return results, nil
}

// executeAutoPageSearch automatically paginates for high limits (legacy behavior)
func executeAutoPageSearch(ctx context.Context, query string) (*github.SearchResults, error) {
	// Initialize rate limiter if not set
	if searchRateLimiter == nil {
		searchRateLimiter = github.NewRateLimiter()
	}

	var allResults *github.SearchResults
	page := 1
	remaining := searchLimit

	for remaining > 0 {
		// GitHub API per_page is capped at 100
		perPage := remaining
		if perPage > GitHubMaxResultsPerPage {
			perPage = GitHubMaxResultsPerPage
		}

		opts := &github.SearchOptions{
			Sort:  sort,
			Order: order,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
			SkipEnrichment: liteMode,
		}

		// Execute search with rate limiting and retry logic
		var results *github.SearchResults
		err := searchRateLimiter.WithRetry(ctx, fmt.Sprintf("search page %d", page), func() error {
			var searchErr error
			results, searchErr = searchClient.SearchCode(ctx, query, opts)
			return searchErr
		})
		if err != nil {
			return nil, err
		}

		if allResults == nil {
			// First page - initialize with the results
			allResults = results
		} else {
			// Subsequent pages - append items
			allResults.Items = append(allResults.Items, results.Items...)
		}

		// Break if fewer results than requested were returned (no more data)
		if len(results.Items) < perPage {
			break
		}

		remaining -= len(results.Items)
		page++

		// Add intelligent delay between paginated calls (except after last page)
		if remaining > 0 && len(results.Items) == perPage {
			// Estimate complexity based on current pagination
			complexity := github.MediumComplexity
			if page > 10 { // High pagination suggests complex query
				complexity = github.HighComplexity
			} else if page <= 3 {
				complexity = github.LowComplexity
			}

			if verbose {
				fmt.Printf("  Adding delay between pages (complexity: %v)...\n", complexity)
			}

			err := searchRateLimiter.IntelligentDelay(ctx, complexity)
			if err != nil {
				return nil, fmt.Errorf("operation cancelled during pagination delay: %w", err)
			}
		}
	}

	return allResults, nil
}

// outputResults formats and outputs the search results
func outputResults(results *github.SearchResults) error {
	if results.Total != nil && *results.Total == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Determine output destination and format
	var output string
	var err error

	if pipe || outputFormat == "pipe" {
		output, err = formatPipeResults(results)
	} else {
		switch outputFormat {
		case "json":
			output, err = formatJSONResults(results)
		case "markdown":
			output, err = formatMarkdownResults(results)
		case "compact":
			output, err = formatCompactResults(results)
		case "default", "":
			output, err = formatDefaultResults(results)
		default:
			return fmt.Errorf("unsupported format: %s (supported: default, json, markdown, compact, pipe)", outputFormat)
		}
	}

	if err != nil {
		return err
	}

	// Output to file or stdout
	if outputFile != "" {
		return writeToFile(output, outputFile)
	}

	fmt.Print(output)
	return nil
}

// detectLanguage detects programming language from file path using constants map
func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	if lang, ok := LanguageExtensionMap[ext]; ok {
		return lang
	}

	// Special case for Dockerfile (no extension)
	if strings.Contains(strings.ToLower(filepath.Base(path)), "dockerfile") {
		return LanguageDockerfile
	}

	return ""
}

// formatPipeResults formats results for pipe output
func formatPipeResults(results *github.SearchResults) (string, error) {
	var output strings.Builder

	for _, item := range results.Items {
		if item.Repository.FullName != nil && item.Path != nil && item.HTMLURL != nil {
			output.WriteString(fmt.Sprintf("%s:%s:%s\n", *item.Repository.FullName, *item.Path, *item.HTMLURL))
		}
	}

	return output.String(), nil
}

// formatDefaultResults formats results for default output
func formatDefaultResults(results *github.SearchResults) (string, error) {
	var output strings.Builder

	// Show pagination-aware results summary
	if results.Total != nil {
		totalResults := *results.Total
		displayedCount := len(results.Items)

		// Calculate result range based on pagination
		var startResult, endResult int
		if searchPage > 0 {
			// Single page mode - calculate exact range
			startResult = ((searchPage - 1) * searchLimit) + 1
			endResult = startResult + displayedCount - 1
		} else {
			// Auto pagination mode (legacy)
			startResult = 1
			endResult = displayedCount
		}

		// Format the pagination-aware header
		if totalResults > displayedCount {
			output.WriteString(fmt.Sprintf("🔍 Found %d total results (showing %d-%d)\n\n",
				totalResults, startResult, endResult))
		} else {
			output.WriteString(fmt.Sprintf("🔍 Found %d results\n\n", totalResults))
		}

		// Add pagination guidance if there are more results
		if totalResults > endResult {
			remainingResults := totalResults - endResult
			nextPage := searchPage + 1
			if searchPage == 0 {
				nextPage = 2 // Auto pagination starts at page 1, next is page 2
			}

			output.WriteString(fmt.Sprintf("💡 **%d more results available** - Use `--page %d` to see results %d-%d\n\n",
				remainingResults, nextPage, endResult+1, min(totalResults, endResult+searchLimit)))
		}
	}

	for i, item := range results.Items {
		if i >= searchLimit {
			break
		}

		// Repository header with stars
		repoName := ""
		repoURL := ""
		stars := 0
		if item.Repository.FullName != nil {
			repoName = *item.Repository.FullName
		}
		if item.Repository.HTMLURL != nil {
			repoURL = *item.Repository.HTMLURL
		}
		if item.Repository.StargazersCount != nil {
			stars = *item.Repository.StargazersCount
		}

		output.WriteString(fmt.Sprintf("📁 [%s](%s) ⭐ %d\n", repoName, repoURL, stars))

		// File path
		if item.Path != nil {
			output.WriteString(fmt.Sprintf("📄 **%s**\n\n", *item.Path))
		}

		// Code content with basic formatting
		if len(item.TextMatches) > 0 {
			for _, match := range item.TextMatches {
				if match.Fragment != nil {
					lang := detectLanguage(*item.Path)
					output.WriteString(fmt.Sprintf("```%s\n%s\n```\n", lang, *match.Fragment))
				}
			}
		}

		// Link to file
		if item.HTMLURL != nil {
			output.WriteString(fmt.Sprintf("🔗 [View on GitHub](%s)\n\n", *item.HTMLURL))
		}

		output.WriteString("---\n")
	}

	return output.String(), nil
}

// writeToFile writes content to a file
func writeToFile(content, filename string) error {
	// Validate path to prevent directory traversal
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: directory traversal not allowed")
	}

	// Get absolute path for clarity
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Create directory if it doesn't exist (user-only permissions)
	dir := filepath.Dir(absPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write file with restrictive permissions (user-only)
	if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", absPath, err)
	}

	fmt.Printf("✅ Results exported to: %s\n", filename)
	return nil
}

// createGitHubClient creates a new GitHub API client
func createGitHubClient() (github.GitHubAPI, error) {
	return github.NewRealClient()
}

func init() {
	// Add search command to root
	rootCmd.AddCommand(searchCmd)

	// Core filtering flags (migrated from ghx)
	searchCmd.Flags().StringVarP(&searchLanguage, "language", "l", "", "programming language filter")
	searchCmd.Flags().StringSliceVarP(&searchRepo, "repo", "r", nil, "repository filter (supports wildcards). Tip: Use 'gh search repos --topic=TOPIC' to find repos by topic first")
	searchCmd.Flags().StringVarP(&searchFilename, "filename", "f", "", "exact filename match")
	searchCmd.Flags().StringVarP(&searchExtension, "extension", "e", "", "file extension filter")
	searchCmd.Flags().StringVarP(&searchPath, "path", "p", "", "file path filter")
	searchCmd.Flags().StringSliceVarP(&searchOwner, "owner", "o", nil, "filter by repository owner (user or organization)")
	searchCmd.Flags().StringVar(&searchSize, "size", "", "file size filter (e.g., '>1000', '<500')")

	// Quality & ranking flags (enhanced from ghx)
	searchCmd.Flags().IntVar(&minStars, "min-stars", 0, "minimum repository stars")
	searchCmd.Flags().StringVar(&sort, "sort", "relevance", "sort by: relevance, stars, updated, created")
	searchCmd.Flags().StringVar(&order, "order", "desc", "sort order: asc, desc")
	searchCmd.Flags().BoolVar(&liteMode, "lite", false, "lite mode: faster search, skips star counts (saves API quota)")

	// Output control flags (migrated from ghx)
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum results per page (default: 50, max: 100)")
	searchCmd.Flags().IntVar(&searchPage, "page", 0, "specific page number (more API efficient than auto-pagination)")
	searchCmd.Flags().IntVar(&contextLines, "context", 20, "context lines around matches (GitHub API controls actual fragment size)")
	searchCmd.Flags().StringVar(&outputFormat, "format", "default", "output format: default, json, markdown, compact")
	searchCmd.Flags().StringVar(&outputFile, "output", "", "export results to file (e.g., results.md, data.json)")
	searchCmd.Flags().BoolVarP(&pipe, "pipe", "", false, "output to stdout (for piping to other tools)")

	// Batch search flags (Phase 2)
	searchCmd.Flags().StringSliceVar(&batchRepos, "repos", nil, "search across multiple repositories (comma-separated)")
	searchCmd.Flags().StringSliceVar(&batchOrgs, "orgs", nil, "search across multiple organizations (comma-separated)")
	searchCmd.Flags().BoolVar(&aggregateMode, "aggregate", false, "aggregate results from multiple repositories")
	searchCmd.Flags().BoolVar(&compareMode, "compare", false, "enable comparison mode for multi-repository results")

	// Workflow integration flags (new)
	// TODO: Implement saved searches functionality
	// searchCmd.Flags().StringVar(&saveAs, "save", "", "save search with given name")

	// Set flag usage examples
	_ = searchCmd.Flags().SetAnnotation("language", "examples", []string{"typescript", "go", "python", "javascript"})
	_ = searchCmd.Flags().SetAnnotation("repo", "examples", []string{"facebook/react", "microsoft/vscode", "**/typescript"})
	_ = searchCmd.Flags().SetAnnotation("filename", "examples", []string{"package.json", "tsconfig.json", "Dockerfile"})
	_ = searchCmd.Flags().SetAnnotation("extension", "examples", []string{"ts", "go", "py", "js"})
	_ = searchCmd.Flags().SetAnnotation("size", "examples", []string{">1000", "<500", "100..200"})
	_ = searchCmd.Flags().SetAnnotation("repos", "examples", []string{"microsoft/vscode,facebook/react", "vercel/*,netlify/*"})
	_ = searchCmd.Flags().SetAnnotation("orgs", "examples", []string{"microsoft,google,facebook", "vercel,netlify"})
}

// formatJSONResults formats search results as JSON with pagination metadata
func formatJSONResults(results *github.SearchResults) (string, error) {
	// Create enhanced response with pagination info
	enhancedResults := struct {
		*github.SearchResults
		Pagination *PaginationInfo `json:"pagination,omitempty"`
	}{
		SearchResults: results,
	}

	// Add pagination metadata if available
	if results.Total != nil {
		totalResults := *results.Total
		displayedCount := len(results.Items)

		var startResult, endResult int
		if searchPage > 0 {
			startResult = ((searchPage - 1) * searchLimit) + 1
			endResult = startResult + displayedCount - 1
		} else {
			startResult = 1
			endResult = displayedCount
		}

		enhancedResults.Pagination = &PaginationInfo{
			TotalResults:     totalResults,
			DisplayedResults: displayedCount,
			StartResult:      startResult,
			EndResult:        endResult,
			CurrentPage:      max(1, searchPage),
			PerPage:          searchLimit,
			HasNextPage:      totalResults > endResult,
		}

		if enhancedResults.Pagination.HasNextPage {
			enhancedResults.Pagination.NextPage = enhancedResults.Pagination.CurrentPage + 1
		}
	}

	data, err := json.MarshalIndent(enhancedResults, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// formatMarkdownResults formats search results as Markdown
func formatMarkdownResults(results *github.SearchResults) (string, error) {
	var builder strings.Builder

	if results.Total != nil {
		builder.WriteString(fmt.Sprintf("# Search Results (%d total)\n\n", *results.Total))
	} else {
		builder.WriteString("# Search Results\n\n")
	}

	for i, item := range results.Items {
		path := ""
		if item.Path != nil {
			path = *item.Path
		}
		fullName := ""
		if item.Repository.FullName != nil {
			fullName = *item.Repository.FullName
		}
		htmlURL := ""
		if item.HTMLURL != nil {
			htmlURL = *item.HTMLURL
		}

		builder.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, path))
		builder.WriteString(fmt.Sprintf("**Repository:** %s\n", fullName))
		if item.Repository.Description != nil && *item.Repository.Description != "" {
			builder.WriteString(fmt.Sprintf("**Description:** %s\n", *item.Repository.Description))
		}
		builder.WriteString(fmt.Sprintf("**URL:** %s\n\n", htmlURL))

		if len(item.TextMatches) > 0 {
			builder.WriteString("**Code:**\n\n```")
			if item.Repository.Language != nil && *item.Repository.Language != "" {
				builder.WriteString(strings.ToLower(*item.Repository.Language))
			}
			builder.WriteString("\n")
			for _, match := range item.TextMatches {
				if match.Fragment != nil {
					builder.WriteString(*match.Fragment + "\n")
				}
			}
			builder.WriteString("```\n\n")
		}
		builder.WriteString("---\n\n")
	}

	return builder.String(), nil
}

// formatCompactResults formats search results in compact format
func formatCompactResults(results *github.SearchResults) (string, error) {
	var builder strings.Builder

	// Add pagination info for compact format too
	if results.Total != nil {
		totalResults := *results.Total
		displayedCount := len(results.Items)

		if searchPage > 0 {
			startResult := ((searchPage - 1) * searchLimit) + 1
			endResult := startResult + displayedCount - 1
			if totalResults > displayedCount {
				builder.WriteString(fmt.Sprintf("# Results %d-%d of %d total\n",
					startResult, endResult, totalResults))
			}
		} else if totalResults > displayedCount {
			builder.WriteString(fmt.Sprintf("# Results 1-%d of %d total\n",
				displayedCount, totalResults))
		}
	}

	for _, item := range results.Items {
		fullName := ""
		if item.Repository.FullName != nil {
			fullName = *item.Repository.FullName
		}
		path := ""
		if item.Path != nil {
			path = *item.Path
		}
		builder.WriteString(fmt.Sprintf("%s:%s\n", fullName, path))
	}

	return builder.String(), nil
}

// isTestEnvironment checks if we're running in test mode
func isTestEnvironment() bool {
	// Check if running under go test
	return os.Getenv("GO_TEST") == "1" || strings.HasSuffix(os.Args[0], ".test")
}

// executeBatchRepoSearch handles multi-repository search with aggregation (Phase 2)
func executeBatchRepoSearch(ctx context.Context, args []string) error {
	// Add timeout for batch operations (longer due to multiple searches)
	if ctx == nil {
		ctx = context.Background()
	}
	if !isTestEnvironment() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	query := strings.Join(args, " ")

	if dryRun {
		fmt.Printf("Would execute batch search: %s\n", query)
		if len(batchRepos) > 0 {
			fmt.Printf("Repositories: %s\n", strings.Join(batchRepos, ", "))
		}
		if len(batchOrgs) > 0 {
			fmt.Printf("Organizations: %s\n", strings.Join(batchOrgs, ", "))
		}
		return nil
	}

	// Collect all target repositories
	var targetRepos []string

	// Add explicit repos
	targetRepos = append(targetRepos, batchRepos...)

	// Add repos from organizations (simplified - in real implementation would use GitHub API)
	for _, org := range batchOrgs {
		if strings.Contains(org, "*") {
			// Handle wildcard patterns
			targetRepos = append(targetRepos, org+"/*")
		} else {
			// For now, treat as repo pattern
			targetRepos = append(targetRepos, org+"/*")
		}
	}

	if verbose {
		fmt.Printf("Executing batch search across %d repository patterns\n", len(targetRepos))
		fmt.Printf("Query: %s\n", query)
	}

	// Execute searches across all repositories
	var allResults *github.SearchResults
	totalResults := 0

	for i, repoPattern := range targetRepos {
		if verbose {
			fmt.Printf("Searching %d/%d: %s\n", i+1, len(targetRepos), repoPattern)
		}

		// Build query with repository filter (commented out - using QueryBuilder instead)

		// Apply other filters
		qb := search.NewQueryBuilder([]string{query})
		if searchLanguage != "" {
			qb = qb.WithLanguage(searchLanguage)
		}
		if searchFilename != "" {
			qb = qb.WithFilename(searchFilename)
		}
		if searchExtension != "" {
			qb = qb.WithExtension(searchExtension)
		}
		if searchPath != "" {
			qb = qb.WithPath(searchPath)
		}
		if searchSize != "" {
			qb = qb.WithSize(searchSize)
		}
		if minStars > 0 {
			qb = qb.WithMinStars(minStars)
		}

		// Add repository filter
		qb = qb.WithRepositories([]string{repoPattern})
		finalQuery := qb.Build()

		// Execute single repository search
		opts := &github.SearchOptions{
			Sort:  sort,
			Order: order,
			ListOptions: github.ListOptions{
				Page:    1,
				PerPage: searchLimit,
			},
			SkipEnrichment: liteMode,
		}

		results, err := searchClient.SearchCode(ctx, finalQuery, opts)
		if err != nil {
			if verbose {
				fmt.Printf("  Warning: Search failed for %s: %v\n", repoPattern, err)
			}
			continue
		}

		if verbose {
			fmt.Printf("  Found %d results\n", len(results.Items))
		}

		// Aggregate results
		if allResults == nil {
			allResults = results
		} else {
			allResults.Items = append(allResults.Items, results.Items...)
			if results.Total != nil {
				if allResults.Total == nil {
					allResults.Total = github.IntPtr(0)
				}
				*allResults.Total += *results.Total
			}
		}

		totalResults += len(results.Items)
	}

	if allResults == nil {
		fmt.Println("No results found across any repositories.")
		return nil
	}

	if verbose {
		fmt.Printf("Batch search completed: %d total results from %d repositories\n", totalResults, len(targetRepos))
	}

	// Handle comparison mode
	if compareMode {
		return outputBatchComparison(allResults, targetRepos)
	}

	// Handle aggregate mode or default output
	return outputResults(allResults)
}

// outputBatchComparison outputs results in comparison format
func outputBatchComparison(results *github.SearchResults, repos []string) error {
	fmt.Printf("# Multi-Repository Search Comparison\n\n")
	fmt.Printf("**Query executed across %d repository patterns**\n", len(repos))
	fmt.Printf("**Total results found: %d**\n\n", len(results.Items))

	// Group results by repository
	repoResults := make(map[string][]github.SearchItem)
	for _, item := range results.Items {
		if item.Repository.FullName != nil {
			repoName := *item.Repository.FullName
			repoResults[repoName] = append(repoResults[repoName], item)
		}
	}

	fmt.Printf("## Results by Repository\n\n")
	for repo, items := range repoResults {
		fmt.Printf("### %s (%d results)\n", repo, len(items))

		// Show top 3 results per repository
		maxShow := 3
		if len(items) < maxShow {
			maxShow = len(items)
		}

		for i := 0; i < maxShow; i++ {
			item := items[i]
			if item.Path != nil && item.HTMLURL != nil {
				fmt.Printf("- **%s** - [View](%s)\n", *item.Path, *item.HTMLURL)
			}
		}

		if len(items) > maxShow {
			fmt.Printf("- ... and %d more results\n", len(items)-maxShow)
		}
		fmt.Println()
	}

	// Simple pattern analysis
	fmt.Printf("## Analysis\n")
	fmt.Printf("- **Repositories with results**: %d\n", len(repoResults))
	fmt.Printf("- **Average results per repository**: %.1f\n", float64(len(results.Items))/float64(len(repoResults)))

	return nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// PaginationInfo contains metadata about pagination
type PaginationInfo struct {
	TotalResults     int  `json:"total_results"`
	DisplayedResults int  `json:"displayed_results"`
	StartResult      int  `json:"start_result"`
	EndResult        int  `json:"end_result"`
	CurrentPage      int  `json:"current_page"`
	PerPage          int  `json:"per_page"`
	HasNextPage      bool `json:"has_next_page"`
	NextPage         int  `json:"next_page,omitempty"`
}
