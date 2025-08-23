package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/silouanwright/gh-scout/internal/github"
	"github.com/silouanwright/gh-scout/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data and fixtures
const testBatchConfigYAML = `name: "Test Configuration"
description: "Test batch search configuration"
output:
  format: "combined"
  compare: true

searches:
  - name: "test-search-1"
    query: "config"
    filters:
      language: "json"
      min_stars: 100
    max_results: 10
    tags: ["config", "json"]

  - name: "test-search-2"
    query: "dockerfile"
    filters:
      language: "dockerfile"
      min_stars: 50
    max_results: 5
    tags: ["docker", "container"]`

const invalidBatchConfigYAML = `name: "Invalid Configuration"
searches:
  - name: ""
    query: ""
    max_results: 10`

const emptySearchesBatchConfigYAML = `name: "Empty Searches"
description: "Configuration with no searches"
searches: []`

func TestReadBatchConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *BatchConfig)
	}{
		{
			name:        "valid configuration",
			yamlContent: testBatchConfigYAML,
			wantErr:     false,
			validate: func(t *testing.T, config *BatchConfig) {
				assert.Equal(t, "Test Configuration", config.Name)
				assert.Equal(t, "Test batch search configuration", config.Description)
				assert.Equal(t, "combined", config.Output.Format)
				assert.True(t, config.Output.Compare)
				assert.Len(t, config.Searches, 2)

				// Validate first search
				search1 := config.Searches[0]
				assert.Equal(t, "test-search-1", search1.Name)
				assert.Equal(t, "config", search1.Query)
				assert.Equal(t, "json", search1.Filters.Language)
				assert.Equal(t, 100, search1.Filters.MinStars)
				assert.Equal(t, 10, search1.MaxResults)
				assert.Equal(t, []string{"config", "json"}, search1.Tags)

				// Validate second search
				search2 := config.Searches[1]
				assert.Equal(t, "test-search-2", search2.Name)
				assert.Equal(t, "dockerfile", search2.Query)
				assert.Equal(t, "dockerfile", search2.Filters.Language)
				assert.Equal(t, 50, search2.Filters.MinStars)
				assert.Equal(t, 5, search2.MaxResults)
				assert.Equal(t, []string{"docker", "container"}, search2.Tags)
			},
		},
		{
			name:        "invalid configuration - empty name and query",
			yamlContent: invalidBatchConfigYAML,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid configuration - no searches",
			yamlContent: emptySearchesBatchConfigYAML,
			wantErr:     true,
			errContains: "must contain at least one search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(t.TempDir(), "test-config.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			// Test reading configuration
			config, err := readBatchConfig(tmpFile)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestReadBatchConfig_FileErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		errContains string
	}{
		{
			name: "file does not exist",
			setupFile: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			errContains: "failed to read file",
		},
		{
			name: "invalid YAML syntax",
			setupFile: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "invalid.yaml")
				err := os.WriteFile(tmpFile, []byte("invalid: yaml: content: ["), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			errContains: "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := tt.setupFile(t)
			config, err := readBatchConfig(configFile)

			assert.Error(t, err)
			assert.Nil(t, config)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestShowDryRunInfo(t *testing.T) {
	config := &BatchConfig{
		Name:        "Test Dry Run",
		Description: "Testing dry run functionality",
		Output: BatchOutputConfig{
			Format:    "comparison",
			Directory: "test-output",
		},
		Searches: []BatchSearchConfig{
			{
				Name:       "search-1",
				Query:      "test query",
				MaxResults: 20,
				Tags:       []string{"test", "example"},
				Filters: search.SearchFilters{
					Language: "go",
					MinStars: 50,
				},
			},
		},
	}

	err := showDryRunInfo(config, "test-config.yaml")
	assert.NoError(t, err)
	// In a real implementation, you might capture stdout to verify output
}

func TestExecuteSingleBatchSearch(t *testing.T) {
	// Create mock client
	mockClient := github.NewMockClient()

	// Set up mock expectations
	expectedResults := &github.SearchResults{
		Total: github.IntPtr(2),
		Items: []github.SearchItem{
			{
				Repository: github.Repository{
					FullName:        github.StringPtr("owner/repo1"),
					StargazersCount: github.IntPtr(100),
				},
				Path:    github.StringPtr("config.json"),
				HTMLURL: github.StringPtr("https://github.com/owner/repo1/blob/main/config.json"),
			},
			{
				Repository: github.Repository{
					FullName:        github.StringPtr("owner/repo2"),
					StargazersCount: github.IntPtr(200),
				},
				Path:    github.StringPtr("app.json"),
				HTMLURL: github.StringPtr("https://github.com/owner/repo2/blob/main/app.json"),
			},
		},
	}

	mockClient.SetSearchResults("config language:json stars:>=100", expectedResults)

	// Set mock client
	originalClient := batchClient
	batchClient = mockClient
	defer func() { batchClient = originalClient }()

	// Test search configuration
	searchConfig := BatchSearchConfig{
		Name:       "test-search",
		Query:      "config",
		MaxResults: 10,
		Tags:       []string{"config", "json"},
		Filters: search.SearchFilters{
			Language: "json",
			MinStars: 100,
		},
	}

	// Execute search
	result, err := executeSingleBatchSearch(context.Background(), searchConfig)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, "test-search", result.Name)
	assert.Equal(t, "config language:json stars:>=100", result.Query)
	assert.Equal(t, []string{"config", "json"}, result.Tags)
	assert.Equal(t, 2, result.ResultCount)
	assert.NotNil(t, result.Results)
	assert.Len(t, result.Results.Items, 2)

	// Verify mock was called
	assert.True(t, mockClient.VerifyCall("SearchCode", "config language:json stars:>=100"))
}

func TestExecuteBatchSearches(t *testing.T) {
	// Create mock client
	mockClient := github.NewMockClient()

	// Set up mock expectations for multiple searches
	results1 := &github.SearchResults{
		Total: github.IntPtr(1),
		Items: []github.SearchItem{
			{
				Repository: github.Repository{
					FullName:        github.StringPtr("owner/repo1"),
					StargazersCount: github.IntPtr(100),
				},
				Path:    github.StringPtr("config.json"),
				HTMLURL: github.StringPtr("https://github.com/owner/repo1/blob/main/config.json"),
			},
		},
	}

	results2 := &github.SearchResults{
		Total: github.IntPtr(1),
		Items: []github.SearchItem{
			{
				Repository: github.Repository{
					FullName:        github.StringPtr("owner/repo2"),
					StargazersCount: github.IntPtr(50),
				},
				Path:    github.StringPtr("Dockerfile"),
				HTMLURL: github.StringPtr("https://github.com/owner/repo2/blob/main/Dockerfile"),
			},
		},
	}

	mockClient.SetSearchResults("config language:json stars:>=100", results1)
	mockClient.SetSearchResults("dockerfile language:dockerfile stars:>=50", results2)

	// Set mock client
	originalClient := batchClient
	batchClient = mockClient
	defer func() { batchClient = originalClient }()

	// Test configuration
	config := &BatchConfig{
		Name:        "Test Batch",
		Description: "Test batch execution",
		Output: BatchOutputConfig{
			Format:  "combined",
			Compare: true,
		},
		Searches: []BatchSearchConfig{
			{
				Name:       "search-1",
				Query:      "config",
				MaxResults: 10,
				Filters: search.SearchFilters{
					Language: "json",
					MinStars: 100,
				},
			},
			{
				Name:       "search-2",
				Query:      "dockerfile",
				MaxResults: 5,
				Filters: search.SearchFilters{
					Language: "dockerfile",
					MinStars: 50,
				},
			},
		},
	}

	// Execute batch searches
	err := executeBatchSearches(context.Background(), config)

	// Verify execution completed without error
	assert.NoError(t, err)

	// Verify mock was called for both searches
	assert.Equal(t, 2, mockClient.GetCallCount("SearchCode"))
}

func TestGenerateComparisons(t *testing.T) {
	results := []BatchSearchResult{
		{
			Name:        "search-1",
			Query:       "config language:json",
			ResultCount: 10,
		},
		{
			Name:        "search-2",
			Query:       "dockerfile language:dockerfile",
			ResultCount: 5,
		},
	}

	comparisons := generateComparisons(results)

	require.Len(t, comparisons, 1)
	comparison := comparisons[0]

	assert.Equal(t, "Overall Analysis", comparison.Name)
	assert.Equal(t, []string{"search-1", "search-2"}, comparison.SearchNames)
	assert.Contains(t, comparison.Summary, "2 searches")
	assert.Contains(t, comparison.Summary, "15 total results")
	assert.NotEmpty(t, comparison.CommonPatterns)
	assert.NotEmpty(t, comparison.KeyDifferences)
}

func TestRunBatch_DryRun(t *testing.T) {
	// Create temporary config file
	tmpFile := filepath.Join(t.TempDir(), "test-config.yaml")
	err := os.WriteFile(tmpFile, []byte(testBatchConfigYAML), 0644)
	require.NoError(t, err)

	// Set dry run flag
	originalDryRun := dryRun
	dryRun = true
	defer func() { dryRun = originalDryRun }()

	// Create mock command and args
	cmd := batchCmd
	args := []string{tmpFile}

	// Execute command
	err = runBatch(cmd, args)

	// Should complete without error in dry run mode
	assert.NoError(t, err)
}

func TestRunBatch_InvalidConfig(t *testing.T) {
	// Create temporary invalid config file
	tmpFile := filepath.Join(t.TempDir(), "invalid-config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidBatchConfigYAML), 0644)
	require.NoError(t, err)

	// Ensure not in dry run mode
	originalDryRun := dryRun
	dryRun = false
	defer func() { dryRun = originalDryRun }()

	// Create mock command and args
	cmd := batchCmd
	args := []string{tmpFile}

	// Execute command
	err = runBatch(cmd, args)

	// Should return configuration error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

// Benchmark tests
func BenchmarkReadBatchConfig(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "bench-config.yaml")
	err := os.WriteFile(tmpFile, []byte(testBatchConfigYAML), 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readBatchConfig(tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteSingleBatchSearch(b *testing.B) {
	// Create mock client
	mockClient := github.NewMockClient()
	results := &github.SearchResults{
		Total: github.IntPtr(1),
		Items: []github.SearchItem{
			{
				Repository: github.Repository{
					FullName: github.StringPtr("owner/repo"),
				},
				Path: github.StringPtr("config.json"),
			},
		},
	}

	mockClient.SetSearchResults("config language:json stars:>=100", results)

	// Set mock client
	originalClient := batchClient
	batchClient = mockClient
	defer func() { batchClient = originalClient }()

	searchConfig := BatchSearchConfig{
		Name:       "bench-search",
		Query:      "config",
		MaxResults: 10,
		Filters: search.SearchFilters{
			Language: "json",
			MinStars: 100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executeSingleBatchSearch(context.Background(), searchConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}
