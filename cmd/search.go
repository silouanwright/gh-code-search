package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/silouanwright/gh-search/internal/github"
	"github.com/spf13/cobra"
)

var (
	// Global client for dependency injection (following gh-comment pattern)
	searchClient github.GitHubAPI

	// Search flags migrated from ghx
	searchLanguage  string
	searchRepo      []string
	searchFilename  string
	searchExtension string
	searchPath      string
	searchOwner     []string
	searchSize      string
	searchLimit     int
	searchPage      int    // New: page-based pagination
	contextLines    int
	outputFormat    string
	saveAs          string
	pipe            bool
	minStars        int
	sort            string
	order           string
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
  gh search "tsconfig.json" --language json --limit 10
  gh search "vite.config" --language javascript --context 30
  gh search "dockerfile" --filename dockerfile --repo "**/react"
  
  # Page-based search (API efficient for large datasets)
  gh search "config" --page 1 --limit 100        # Get first 100 results
  gh search "config" --page 2 --limit 100        # Get next 100 results
  gh search "config" --page 3 --limit 50         # Get results 201-250
  
  # Organization/owner-specific searches
  gh search "eslint.config.js" --owner microsoft --language javascript
  gh search "next.config.js" --owner vercel --page 1 --limit 50
  gh search "interface" --owner google --owner facebook --language typescript
  
  # Auto-pagination (less API efficient but convenient)
  gh search "hooks" --limit 200                  # Automatically fetches 2 pages

  # Pipe results for further processing
  gh search "react hooks" --language typescript --pipe`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Initialize client if not set (for testing)
	if searchClient == nil {
		client, err := createGitHubClient()
		if err != nil {
			return handleClientError(err)
		}
		searchClient = client
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

	// Execute search with error handling
	results, err := executeSearch(cmd.Context(), query)
	if err != nil {
		return handleSearchError(err, query)
	}

	if verbose {
		fmt.Printf("Found %d results\n", len(results.Items))
	}

	// Process and output results
	return outputResults(results)
}

// buildSearchQuery constructs GitHub search query from args and flags (migrated from ghx)
func buildSearchQuery(terms []string) string {
	var parts []string

	// Add search terms
	if len(terms) > 0 {
		parts = append(parts, strings.Join(terms, " "))
	}

	// Add language filter (from ghx --language)
	if searchLanguage != "" {
		parts = append(parts, fmt.Sprintf("language:%s", searchLanguage))
	}

	// Add filename filter (from ghx --filename)
	if searchFilename != "" {
		parts = append(parts, fmt.Sprintf("filename:%s", searchFilename))
	}

	// Add extension filter (from ghx --extension)
	if searchExtension != "" {
		parts = append(parts, fmt.Sprintf("extension:%s", searchExtension))
	}

	// Add repository filters (from ghx --repo)
	for _, repo := range searchRepo {
		parts = append(parts, fmt.Sprintf("repo:%s", repo))
	}

	// Add path filter (from ghx --path)
	if searchPath != "" {
		parts = append(parts, fmt.Sprintf("path:%s", searchPath))
	}

	// Add owner filters (from ghx --owner, mapped to user:)
	for _, owner := range searchOwner {
		parts = append(parts, fmt.Sprintf("user:%s", owner))
	}

	// Add size filter (from ghx --size)
	if searchSize != "" {
		parts = append(parts, fmt.Sprintf("size:%s", searchSize))
	}

	// Add stars filter (new enhancement)
	if minStars > 0 {
		parts = append(parts, fmt.Sprintf("stars:>=%d", minStars))
	}

	return strings.Join(parts, " ")
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
		}

		results, err := searchClient.SearchCode(ctx, query, opts)
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
	}

	return allResults, nil
}

// outputResults formats and outputs the search results
func outputResults(results *github.SearchResults) error {
	if results.Total != nil && *results.Total == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// For now, simple output - will be enhanced with formatters
	if pipe {
		return outputPipeFormat(results)
	}

	return outputDefaultFormat(results)
}

// outputPipeFormat outputs results in a pipe-friendly format
func outputPipeFormat(results *github.SearchResults) error {
	for _, item := range results.Items {
		if item.Repository.FullName != nil && item.Path != nil && item.HTMLURL != nil {
			fmt.Printf("%s:%s:%s\n", *item.Repository.FullName, *item.Path, *item.HTMLURL)
		}
	}
	return nil
}

// outputDefaultFormat outputs results in the default user-friendly format
func outputDefaultFormat(results *github.SearchResults) error {
	if results.Total != nil {
		fmt.Printf("🔍 Found %d results\n\n", *results.Total)
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

		fmt.Printf("📁 [%s](%s) ⭐ %d\n", repoName, repoURL, stars)

		// File path
		if item.Path != nil {
			fmt.Printf("📄 **%s**\n\n", *item.Path)
		}

		// Code content with basic formatting
		if len(item.TextMatches) > 0 {
			for _, match := range item.TextMatches {
				if match.Fragment != nil {
					lang := detectLanguage(*item.Path)
					fmt.Printf("```%s\n%s\n```\n", lang, *match.Fragment)
				}
			}
		}

		// Link to file
		if item.HTMLURL != nil {
			fmt.Printf("🔗 [View on GitHub](%s)\n\n", *item.HTMLURL)
		}

		fmt.Println("---")
	}

	return nil
}

// detectLanguage detects programming language from file path
func detectLanguage(path string) string {
	if strings.HasSuffix(path, ".go") {
		return "go"
	}
	if strings.HasSuffix(path, ".js") {
		return "javascript"
	}
	if strings.HasSuffix(path, ".ts") {
		return "typescript"
	}
	if strings.HasSuffix(path, ".tsx") {
		return "typescript"
	}
	if strings.HasSuffix(path, ".py") {
		return "python"
	}
	if strings.HasSuffix(path, ".json") {
		return "json"
	}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return "yaml"
	}
	if strings.HasSuffix(path, ".md") {
		return "markdown"
	}
	if strings.HasSuffix(path, ".dockerfile") || strings.Contains(strings.ToLower(path), "dockerfile") {
		return "dockerfile"
	}
	return ""
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
	searchCmd.Flags().StringSliceVarP(&searchRepo, "repo", "r", nil, "repository filter (supports wildcards)")
	searchCmd.Flags().StringVarP(&searchFilename, "filename", "f", "", "exact filename match")
	searchCmd.Flags().StringVarP(&searchExtension, "extension", "e", "", "file extension filter")
	searchCmd.Flags().StringVarP(&searchPath, "path", "p", "", "file path filter")
	searchCmd.Flags().StringSliceVarP(&searchOwner, "owner", "o", nil, "repository owner/organization filter")
	searchCmd.Flags().StringVar(&searchSize, "size", "", "file size filter (e.g., '>1000', '<500')")

	// Quality & ranking flags (enhanced from ghx)
	searchCmd.Flags().IntVar(&minStars, "min-stars", 0, "minimum repository stars")
	searchCmd.Flags().StringVar(&sort, "sort", "relevance", "sort by: relevance, stars, updated, created")
	searchCmd.Flags().StringVar(&order, "order", "desc", "sort order: asc, desc")

	// Output control flags (migrated from ghx)
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum results per page (default: 50, max: 100)")
	searchCmd.Flags().IntVar(&searchPage, "page", 0, "specific page number (more API efficient than auto-pagination)")
	searchCmd.Flags().IntVar(&contextLines, "context", 20, "context lines around matches")
	searchCmd.Flags().StringVar(&outputFormat, "format", "default", "output format: default, json, markdown, compact")
	searchCmd.Flags().BoolVarP(&pipe, "pipe", "", false, "output to stdout (for piping to other tools)")

	// Workflow integration flags (new)
	searchCmd.Flags().StringVar(&saveAs, "save", "", "save search with given name")

	// Set flag usage examples
	searchCmd.Flags().SetAnnotation("language", "examples", []string{"typescript", "go", "python", "javascript"})
	searchCmd.Flags().SetAnnotation("repo", "examples", []string{"facebook/react", "microsoft/vscode", "**/typescript"})
	searchCmd.Flags().SetAnnotation("filename", "examples", []string{"package.json", "tsconfig.json", "Dockerfile"})
	searchCmd.Flags().SetAnnotation("extension", "examples", []string{"ts", "go", "py", "js"})
	searchCmd.Flags().SetAnnotation("size", "examples", []string{">1000", "<500", "100..200"})
}