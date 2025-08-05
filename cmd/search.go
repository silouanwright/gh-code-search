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
  
  # Pattern research with filtering  
  gh search "eslint.config.js" --language javascript --save eslint-research
  gh search "next.config.js" --repo vercel/next.js --context 50
  
  # Advanced filtering for quality results
  gh search "tailwind.config" --min-stars 1000 --language javascript
  gh search "package.json" --path "examples/" --limit 20

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

// executeSearch performs the GitHub search with options
func executeSearch(ctx context.Context, query string) (*github.SearchResults, error) {
	opts := &github.SearchOptions{
		Sort:  sort,
		Order: order,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: searchLimit,
		},
	}

	return searchClient.SearchCode(ctx, query, opts)
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

		fmt.Println("---\n")
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
	searchCmd.Flags().StringSliceVarP(&searchOwner, "owner", "o", nil, "repository owner filter")
	searchCmd.Flags().StringVar(&searchSize, "size", "", "file size filter (e.g., '>1000', '<500')")

	// Quality & ranking flags (enhanced from ghx)
	searchCmd.Flags().IntVar(&minStars, "min-stars", 0, "minimum repository stars")
	searchCmd.Flags().StringVar(&sort, "sort", "relevance", "sort by: relevance, stars, updated, created")
	searchCmd.Flags().StringVar(&order, "order", "desc", "sort order: asc, desc")

	// Output control flags (migrated from ghx)
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum results (default: 50, max: 1000)")
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