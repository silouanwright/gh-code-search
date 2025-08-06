package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/silouanwright/gh-search/internal/github"
	"github.com/silouanwright/gh-search/internal/search"
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
	outputFile      string // New: export to file
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

  # Export results to files
  gh search "config" --language json --output configs.md     # Markdown export
  gh search "hooks" --pipe --output data.txt                 # Pipe format export

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
	
	if results.Total != nil {
		output.WriteString(fmt.Sprintf("🔍 Found %d results\n\n", *results.Total))
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
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write file
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
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
	searchCmd.Flags().IntVar(&contextLines, "context", 20, "context lines around matches (GitHub API controls actual fragment size)")
	searchCmd.Flags().StringVar(&outputFormat, "format", "default", "output format: default, json, markdown, compact")
	searchCmd.Flags().StringVar(&outputFile, "output", "", "export results to file (e.g., results.md, data.json)")
	searchCmd.Flags().BoolVarP(&pipe, "pipe", "", false, "output to stdout (for piping to other tools)")

	// Workflow integration flags (new)
	// TODO: Implement saved searches functionality
	// searchCmd.Flags().StringVar(&saveAs, "save", "", "save search with given name")

	// Set flag usage examples
	searchCmd.Flags().SetAnnotation("language", "examples", []string{"typescript", "go", "python", "javascript"})
	searchCmd.Flags().SetAnnotation("repo", "examples", []string{"facebook/react", "microsoft/vscode", "**/typescript"})
	searchCmd.Flags().SetAnnotation("filename", "examples", []string{"package.json", "tsconfig.json", "Dockerfile"})
	searchCmd.Flags().SetAnnotation("extension", "examples", []string{"ts", "go", "py", "js"})
	searchCmd.Flags().SetAnnotation("size", "examples", []string{">1000", "<500", "100..200"})
}

// formatJSONResults formats search results as JSON
func formatJSONResults(results *github.SearchResults) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
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