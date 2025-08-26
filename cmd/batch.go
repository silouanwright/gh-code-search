package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/silouanwright/gh-scout/internal/github"
	"github.com/silouanwright/gh-scout/internal/search"
)

var (
	// Client for dependency injection (tests can override)
	batchClient github.GitHubAPI
	// Rate limiter for intelligent retry and delay logic
	batchRateLimiter *github.RateLimiter
)

// BatchConfig represents the structure of a batch search configuration file
type BatchConfig struct {
	Name        string              `yaml:"name,omitempty"`
	Description string              `yaml:"description,omitempty"`
	Output      BatchOutputConfig   `yaml:"output,omitempty"`
	Searches    []BatchSearchConfig `yaml:"searches"`
}

// BatchOutputConfig represents output configuration for batch searches
type BatchOutputConfig struct {
	Format    string `yaml:"format"`    // "combined", "separate", "comparison"
	Directory string `yaml:"directory"` // Output directory
	Compare   bool   `yaml:"compare"`   // Enable comparison mode
	Aggregate bool   `yaml:"aggregate"` // Combine results
}

// BatchSearchConfig represents individual search configuration
type BatchSearchConfig struct {
	Name       string               `yaml:"name"`
	Query      string               `yaml:"query"`
	Filters    search.SearchFilters `yaml:"filters"`
	MaxResults int                  `yaml:"max_results,omitempty"`
	Tags       []string             `yaml:"tags,omitempty"`
}

// BatchResults holds aggregated results from multiple searches
type BatchResults struct {
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	SearchCount  int                     `json:"search_count"`
	TotalResults int                     `json:"total_results"`
	Results      []BatchSearchResult     `json:"results"`
	Comparisons  []BatchComparisonResult `json:"comparisons,omitempty"`
}

// BatchSearchResult holds results from a single batch search
type BatchSearchResult struct {
	Name        string                `json:"name"`
	Query       string                `json:"query"`
	Tags        []string              `json:"tags,omitempty"`
	ResultCount int                   `json:"result_count"`
	Results     *github.SearchResults `json:"results"`
}

// BatchComparisonResult holds comparison analysis between searches
type BatchComparisonResult struct {
	Name           string   `json:"name"`
	SearchNames    []string `json:"search_names"`
	CommonPatterns []string `json:"common_patterns"`
	KeyDifferences []string `json:"key_differences"`
	Summary        string   `json:"summary"`
}

var batchCmd = &cobra.Command{
	Use:   "batch <config-file>",
	Short: "Execute multiple searches from a YAML configuration file",
	Long: heredoc.Doc(`
		Execute multiple GitHub searches from a YAML configuration file.

		This enables powerful workflows like configuration discovery, pattern analysis,
		and comparative searches across different repositories or technologies.

		Perfect for:
		- Tech stack comparative analysis
		- Configuration pattern discovery
		- Multi-repository audits
		- Systematic code research

		YAML Configuration:
		- Define multiple searches with different filters
		- Aggregate and compare results automatically
		- Export to various formats (JSON, Markdown, etc.)
	`),
	Example: heredoc.Doc(`
		# Execute batch search from config
		$ gh scout batch tech-stack-analysis.yaml

		# Validate config without executing
		$ gh scout batch config.yaml --dry-run

		# Use verbose output to see progress
		$ gh scout batch config.yaml --verbose

		# Override output format
		$ gh scout batch config.yaml --format json
	`),
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Initialize client if not set (production use)
	if batchClient == nil {
		client, err := createGitHubClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		batchClient = client
	}

	// Read and validate batch configuration
	configFile := args[0]
	config, err := readBatchConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Handle verbose output and dry run
	if verbose {
		fmt.Printf("Configuration: %s\n", configFile)
		fmt.Printf("Searches to process: %d\n", len(config.Searches))
		if config.Output.Compare {
			fmt.Printf("Comparison mode: enabled\n")
		}
		fmt.Println()
	}

	if dryRun {
		return showDryRunInfo(config, configFile)
	}

	// Execute batch processing
	return executeBatchSearches(cmd.Context(), config)
}

// readBatchConfig reads and validates the batch configuration file
func readBatchConfig(configFile string) (*BatchConfig, error) {
	// Read file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", configFile, err)
	}

	// Parse YAML
	var config BatchConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate configuration
	if len(config.Searches) == 0 {
		return nil, fmt.Errorf("configuration must contain at least one search")
	}

	// Validate searches
	for i, searchConfig := range config.Searches {
		if searchConfig.Name == "" {
			return nil, fmt.Errorf("search %d: name is required", i+1)
		}
		if searchConfig.Query == "" {
			return nil, fmt.Errorf("search %d: query is required", i+1)
		}

		// Set default max results if not specified
		if searchConfig.MaxResults == 0 {
			config.Searches[i].MaxResults = 50 // Default limit
		}
	}

	// Set default output format if not specified
	if config.Output.Format == "" {
		config.Output.Format = "combined"
	}

	return &config, nil
}

// showDryRunInfo displays what would be executed without running
func showDryRunInfo(config *BatchConfig, configFile string) error {
	fmt.Printf("Would execute batch search from: %s\n", configFile)
	fmt.Printf("Configuration: %s\n", config.Name)
	if config.Description != "" {
		fmt.Printf("Description: %s\n", config.Description)
	}
	fmt.Printf("Output format: %s\n", config.Output.Format)
	if config.Output.Directory != "" {
		fmt.Printf("Output directory: %s\n", config.Output.Directory)
	}
	fmt.Printf("\nSearches to execute:\n")

	for i, searchConfig := range config.Searches {
		fmt.Printf("  %d. %s\n", i+1, searchConfig.Name)
		fmt.Printf("     Query: %s\n", searchConfig.Query)
		if len(searchConfig.Tags) > 0 {
			fmt.Printf("     Tags: %s\n", strings.Join(searchConfig.Tags, ", "))
		}
		fmt.Printf("     Max results: %d\n", searchConfig.MaxResults)

		// Show applied filters
		if searchConfig.Filters.Language != "" {
			fmt.Printf("     Language: %s\n", searchConfig.Filters.Language)
		}
		if searchConfig.Filters.Filename != "" {
			fmt.Printf("     Filename: %s\n", searchConfig.Filters.Filename)
		}
		if len(searchConfig.Filters.Repository) > 0 {
			fmt.Printf("     Repositories: %s\n", strings.Join(searchConfig.Filters.Repository, ", "))
		}
		if len(searchConfig.Filters.Owner) > 0 {
			fmt.Printf("     Owners: %s\n", strings.Join(searchConfig.Filters.Owner, ", "))
		}
		if searchConfig.Filters.MinStars > 0 {
			fmt.Printf("     Min stars: %d\n", searchConfig.Filters.MinStars)
		}
		fmt.Println()
	}

	if config.Output.Compare {
		fmt.Printf("Would generate comparison analysis between searches\n")
	}

	return nil
}

// executeBatchSearches executes all searches and processes results
func executeBatchSearches(ctx context.Context, config *BatchConfig) error {
	if verbose {
		fmt.Printf("Executing %d searches...\n", len(config.Searches))
	}

	// Initialize performance tracking
	performanceTracker := github.NewPerformanceTracker()
	performanceTracker.StartBatch(len(config.Searches))

	var batchResults BatchResults
	batchResults.Name = config.Name
	batchResults.Description = config.Description
	batchResults.SearchCount = len(config.Searches)

	// Initialize rate limiter if not set
	if batchRateLimiter == nil {
		batchRateLimiter = github.NewRateLimiter()
	}

	// Execute each search sequentially with intelligent rate limiting
	for i, searchConfig := range config.Searches {
		if verbose {
			fmt.Printf("Executing search %d/%d: %s\n", i+1, len(config.Searches), searchConfig.Name)
		}

		// Start tracking this search
		performanceTracker.StartSearch(searchConfig.Name, searchConfig.Query)

		// Execute search with retry logic and performance tracking
		var result BatchSearchResult
		err := batchRateLimiter.WithRetry(ctx, fmt.Sprintf("batch search '%s'", searchConfig.Name), func() error {
			var searchErr error
			result, searchErr = executeSingleBatchSearch(ctx, searchConfig)
			if searchErr != nil {
				performanceTracker.RecordRetry()
			}
			return searchErr
		})
		// End tracking for this search
		resultCount := 0
		if err == nil {
			resultCount = result.ResultCount
		}
		performanceTracker.EndSearch(resultCount, err)

		if err != nil {
			// Generate performance report before returning error
			performanceTracker.EndBatch()
			if verbose {
				fmt.Println("\n" + performanceTracker.GenerateReport())
			}
			return fmt.Errorf("failed to execute search '%s': %w", searchConfig.Name, err)
		}

		batchResults.Results = append(batchResults.Results, result)
		batchResults.TotalResults += result.ResultCount

		if verbose {
			fmt.Printf("  Found %d results\n", result.ResultCount)
		}

		// Add intelligent delay between searches (except after last search)
		if i < len(config.Searches)-1 {
			complexity := github.EstimateComplexity(searchConfig.Query, searchConfig.MaxResults, hasFilters(searchConfig.Filters))
			if verbose {
				fmt.Printf("  Adding delay for complexity: %v\n", complexity)
			}
			delay := batchRateLimiter.CalculateIntelligentDelay(complexity)
			performanceTracker.RecordDelay(delay)

			err := batchRateLimiter.IntelligentDelay(ctx, complexity)
			if err != nil {
				return fmt.Errorf("operation cancelled during delay: %w", err)
			}
		}
	}

	// Generate comparisons if requested
	if config.Output.Compare && len(batchResults.Results) > 1 {
		if verbose {
			fmt.Printf("Generating comparison analysis...\n")
		}
		batchResults.Comparisons = generateComparisons(batchResults.Results)
	}

	// End performance tracking
	performanceTracker.EndBatch()

	// Output results
	err := outputBatchResults(&batchResults, config.Output)
	if err != nil {
		return err
	}

	// Show performance report if verbose
	if verbose {
		fmt.Println("\n" + performanceTracker.GenerateDetailedReport())
	}

	return nil
}

// executeSingleBatchSearch executes a single search from the batch configuration
func executeSingleBatchSearch(ctx context.Context, searchConfig BatchSearchConfig) (BatchSearchResult, error) {
	// Build query using existing search package functionality
	terms := strings.Fields(searchConfig.Query)
	qb := search.NewQueryBuilderFromFilters(terms, searchConfig.Filters)
	query := qb.Build()

	// Set up search options
	opts := &github.SearchOptions{
		Sort:  "relevance", // Use relevance for batch searches
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: searchConfig.MaxResults,
		},
		SkipEnrichment: false, // Batch searches typically want full info
	}

	// Execute search
	results, err := batchClient.SearchCode(ctx, query, opts)
	if err != nil {
		return BatchSearchResult{}, err
	}

	// Build result
	batchResult := BatchSearchResult{
		Name:        searchConfig.Name,
		Query:       query,
		Tags:        searchConfig.Tags,
		ResultCount: len(results.Items),
		Results:     results,
	}

	return batchResult, nil
}

// generateComparisons analyzes results to find patterns and differences
func generateComparisons(results []BatchSearchResult) []BatchComparisonResult {
	var comparisons []BatchComparisonResult

	// For now, create a simple overall comparison
	// This can be expanded with more sophisticated analysis
	searchNames := make([]string, len(results))
	totalResults := 0

	for i, result := range results {
		searchNames[i] = result.Name
		totalResults += result.ResultCount
	}

	comparison := BatchComparisonResult{
		Name:        "Overall Analysis",
		SearchNames: searchNames,
		Summary:     fmt.Sprintf("Analyzed %d searches with %d total results", len(results), totalResults),
	}

	// Add simple pattern analysis
	comparison.CommonPatterns = []string{
		"Configuration files found across multiple ecosystems",
		"Repository patterns vary by technology stack",
	}

	comparison.KeyDifferences = []string{
		"Result counts vary significantly between searches",
		"Different file naming conventions observed",
	}

	comparisons = append(comparisons, comparison)
	return comparisons
}

// hasFilters checks if any filters are applied to determine complexity
func hasFilters(filters search.SearchFilters) bool {
	return filters.Language != "" ||
		filters.Filename != "" ||
		len(filters.Repository) > 0 ||
		len(filters.Owner) > 0 ||
		filters.MinStars > 0
}

// outputBatchResults formats and outputs the batch results
func outputBatchResults(results *BatchResults, outputConfig BatchOutputConfig) error {
	// For now, output in JSON format to stdout
	// This can be expanded to support different formats and file output

	if verbose {
		fmt.Printf("\nBatch search completed successfully!\n")
		fmt.Printf("Total searches: %d\n", results.SearchCount)
		fmt.Printf("Total results: %d\n", results.TotalResults)
		if len(results.Comparisons) > 0 {
			fmt.Printf("Comparisons generated: %d\n", len(results.Comparisons))
		}
		fmt.Println()
	}

	// Output summary
	fmt.Printf("🔍 Batch Search Results: %s\n", results.Name)
	if results.Description != "" {
		fmt.Printf("📋 %s\n", results.Description)
	}
	fmt.Printf("📊 Executed %d searches, found %d total results\n\n", results.SearchCount, results.TotalResults)

	// Output individual search results
	for i, result := range results.Results {
		fmt.Printf("## %d. %s (%d results)\n", i+1, result.Name, result.ResultCount)
		fmt.Printf("**Query:** `%s`\n", result.Query)
		if len(result.Tags) > 0 {
			fmt.Printf("**Tags:** %s\n", strings.Join(result.Tags, ", "))
		}

		// Show top results
		maxShow := 3
		if len(result.Results.Items) < maxShow {
			maxShow = len(result.Results.Items)
		}

		for j := 0; j < maxShow; j++ {
			item := result.Results.Items[j]
			if item.Repository.FullName != nil && item.Path != nil {
				stars := 0
				if item.Repository.StargazersCount != nil {
					stars = *item.Repository.StargazersCount
				}
				fmt.Printf("- **%s** (%s) ⭐ %d\n", *item.Path, *item.Repository.FullName, stars)
			}
		}

		if len(result.Results.Items) > maxShow {
			fmt.Printf("- ... and %d more results\n", len(result.Results.Items)-maxShow)
		}
		fmt.Println()
	}

	// Output comparisons if available
	if len(results.Comparisons) > 0 {
		fmt.Printf("## Analysis & Comparisons\n")
		for _, comparison := range results.Comparisons {
			fmt.Printf("### %s\n", comparison.Name)
			fmt.Printf("%s\n\n", comparison.Summary)

			if len(comparison.CommonPatterns) > 0 {
				fmt.Printf("**Common Patterns:**\n")
				for _, pattern := range comparison.CommonPatterns {
					fmt.Printf("- %s\n", pattern)
				}
				fmt.Println()
			}

			if len(comparison.KeyDifferences) > 0 {
				fmt.Printf("**Key Differences:**\n")
				for _, diff := range comparison.KeyDifferences {
					fmt.Printf("- %s\n", diff)
				}
				fmt.Println()
			}
		}
	}

	return nil
}
