package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/silouanwright/gh-scout/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 50, config.Defaults.MaxResults)
	assert.Equal(t, 20, config.Defaults.ContextLines)
	assert.Equal(t, "default", config.Defaults.OutputFormat)
	assert.Equal(t, "relevance", config.Defaults.SortBy)
	assert.Equal(t, "auto", config.Output.ColorMode)
	assert.Equal(t, 50, config.Output.MaxContentLines)
	assert.True(t, config.Analysis.EnablePatterns)
	assert.Equal(t, 2, config.Analysis.MinPatternCount)
	assert.Equal(t, 0.3, config.Analysis.PatternThreshold)
	assert.Equal(t, 5, config.GitHub.RateLimitBuffer)
	assert.Equal(t, "30s", config.GitHub.Timeout)
	assert.Equal(t, 3, config.GitHub.RetryCount)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*Config)
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "valid config",
			setupFunc: func(c *Config) {
				// Use defaults - should be valid
			},
			wantErr: false,
		},
		{
			name: "invalid max results - too low",
			setupFunc: func(c *Config) {
				c.Defaults.MaxResults = 0
			},
			wantErr:  true,
			errorMsg: "defaults.max_results must be between 1 and 1000",
		},
		{
			name: "invalid max results - too high",
			setupFunc: func(c *Config) {
				c.Defaults.MaxResults = 1001
			},
			wantErr:  true,
			errorMsg: "defaults.max_results must be between 1 and 1000",
		},
		{
			name: "invalid context lines - negative",
			setupFunc: func(c *Config) {
				c.Defaults.ContextLines = -1
			},
			wantErr:  true,
			errorMsg: "defaults.context_lines must be between 0 and 100",
		},
		{
			name: "invalid output format",
			setupFunc: func(c *Config) {
				c.Defaults.OutputFormat = "invalid"
			},
			wantErr:  true,
			errorMsg: "defaults.output_format must be one of: default, json, markdown, compact",
		},
		{
			name: "invalid sort by",
			setupFunc: func(c *Config) {
				c.Defaults.SortBy = "invalid"
			},
			wantErr:  true,
			errorMsg: "defaults.sort_by must be one of: relevance, stars, updated, created",
		},
		{
			name: "invalid color mode",
			setupFunc: func(c *Config) {
				c.Output.ColorMode = "invalid"
			},
			wantErr:  true,
			errorMsg: "output.color_mode must be one of: auto, always, never",
		},
		{
			name: "invalid pattern threshold - too low",
			setupFunc: func(c *Config) {
				c.Analysis.PatternThreshold = -0.1
			},
			wantErr:  true,
			errorMsg: "analysis.pattern_threshold must be between 0 and 1",
		},
		{
			name: "invalid pattern threshold - too high",
			setupFunc: func(c *Config) {
				c.Analysis.PatternThreshold = 1.1
			},
			wantErr:  true,
			errorMsg: "analysis.pattern_threshold must be between 0 and 1",
		},
		{
			name: "invalid retry count",
			setupFunc: func(c *Config) {
				c.GitHub.RetryCount = 11
			},
			wantErr:  true,
			errorMsg: "github.retry_count must be between 0 and 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := defaultConfig()
			tt.setupFunc(config)

			err := config.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSavedSearchOperations(t *testing.T) {
	config := defaultConfig()

	// Test adding a saved search
	search := SavedSearch{
		Name:        "test-search",
		Query:       "useState",
		Filters:     search.SearchFilters{Language: "typescript"},
		Description: "Find React hooks",
		Tags:        []string{"react", "hooks"},
	}

	config.AddSavedSearch(search)

	// Test getting saved search
	retrieved, exists := config.GetSavedSearch("test-search")
	assert.True(t, exists)
	assert.Equal(t, "test-search", retrieved.Name)
	assert.Equal(t, "useState", retrieved.Query)
	assert.Equal(t, "typescript", retrieved.Filters.Language)
	assert.Equal(t, "Find React hooks", retrieved.Description)
	assert.Equal(t, []string{"react", "hooks"}, retrieved.Tags)
	assert.False(t, retrieved.Created.IsZero())
	assert.Equal(t, 0, retrieved.UseCount)

	// Test updating saved search (should preserve created time and use count)
	originalCreated := retrieved.Created
	updatedSearch := SavedSearch{
		Name:        "test-search",
		Query:       "useEffect",
		Description: "Updated description",
	}

	config.AddSavedSearch(updatedSearch)
	updated, exists := config.GetSavedSearch("test-search")
	assert.True(t, exists)
	assert.Equal(t, "useEffect", updated.Query)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, originalCreated, updated.Created) // Should preserve
	assert.Equal(t, 0, updated.UseCount)             // Should preserve

	// Test using saved search
	err := config.UseSavedSearch("test-search")
	assert.NoError(t, err)

	used, exists := config.GetSavedSearch("test-search")
	assert.True(t, exists)
	assert.Equal(t, 1, used.UseCount)
	assert.False(t, used.LastUsed.IsZero())

	// Test using non-existent search
	err = config.UseSavedSearch("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting saved search
	err = config.DeleteSavedSearch("test-search")
	assert.NoError(t, err)

	_, exists = config.GetSavedSearch("test-search")
	assert.False(t, exists)

	// Test deleting non-existent search
	err = config.DeleteSavedSearch("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListSavedSearches(t *testing.T) {
	config := defaultConfig()

	// Add multiple searches with different last used times
	search1 := SavedSearch{Name: "old-search", Query: "old"}
	search2 := SavedSearch{Name: "new-search", Query: "new"}
	search3 := SavedSearch{Name: "middle-search", Query: "middle"}

	config.AddSavedSearch(search1)
	time.Sleep(time.Millisecond) // Ensure different timestamps
	config.AddSavedSearch(search2)
	time.Sleep(time.Millisecond)
	config.AddSavedSearch(search3)

	// Use them in different order to set LastUsed times
	_ = config.UseSavedSearch("old-search")
	time.Sleep(time.Millisecond)
	_ = config.UseSavedSearch("new-search")
	time.Sleep(time.Millisecond)
	_ = config.UseSavedSearch("middle-search")

	// List should be sorted by last used (most recent first)
	searches := config.ListSavedSearches()
	assert.Len(t, searches, 3)
	assert.Equal(t, "middle-search", searches[0].Name) // Most recently used
	assert.Equal(t, "new-search", searches[1].Name)
	assert.Equal(t, "old-search", searches[2].Name)   // Least recently used
}

func TestConfigFileOperations(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create test config
	config := defaultConfig()
	config.Defaults.MaxResults = 100
	config.AddSavedSearch(SavedSearch{
		Name:        "test",
		Query:       "test query",
		Description: "test description",
	})

	// Test saving config
	err := config.SaveToFile(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Test loading config
	loadedConfig, err := LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, 100, loadedConfig.Defaults.MaxResults)
	search, exists := loadedConfig.GetSavedSearch("test")
	assert.True(t, exists)
	assert.Equal(t, "test query", search.Query)
	assert.Equal(t, "test description", search.Description)
}

func TestConfigPathResolution(t *testing.T) {
	paths := getConfigPaths()

	assert.NotEmpty(t, paths)

	// Should check current directory first
	assert.Contains(t, paths, ".gh-scout.yaml")
	assert.Contains(t, paths, ".gh-scout.yml")

	// Should include user config directory
	configDir := getConfigDir()
	expectedPath := filepath.Join(configDir, "gh-scout.yaml")
	assert.Contains(t, paths, expectedPath)
}

func TestGetConfigDir(t *testing.T) {
	// Test with environment variable
	originalEnv := os.Getenv("GH_SEARCH_CONFIG_DIR")
	defer os.Setenv("GH_SEARCH_CONFIG_DIR", originalEnv)

	testDir := "/custom/config/dir"
	os.Setenv("GH_SEARCH_CONFIG_DIR", testDir)

	configDir := getConfigDir()
	assert.Equal(t, testDir, configDir)

	// Test with XDG_CONFIG_HOME
	os.Setenv("GH_SEARCH_CONFIG_DIR", "")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	xdgHome := "/custom/xdg/config"
	os.Setenv("XDG_CONFIG_HOME", xdgHome)

	configDir = getConfigDir()
	assert.Equal(t, filepath.Join(xdgHome, "gh-scout"), configDir)

	// Clean up environment
	os.Setenv("XDG_CONFIG_HOME", originalXDG)
}

func TestApplyDefaults(t *testing.T) {
	// Create config with missing values
	config := &Config{
		Defaults: DefaultSettings{
			// Leave MaxResults as 0 (should be filled)
			ContextLines: 10, // This should be kept
		},
		Output: OutputSettings{
			// Leave ColorMode empty (should be filled)
			MaxContentLines: 25, // This should be kept
		},
	}

	applyDefaults(config)

	// Check that missing values were filled with defaults
	assert.Equal(t, 50, config.Defaults.MaxResults)     // Should be filled
	assert.Equal(t, 10, config.Defaults.ContextLines)   // Should be preserved
	assert.Equal(t, "auto", config.Output.ColorMode)    // Should be filled
	assert.Equal(t, 25, config.Output.MaxContentLines)  // Should be preserved

	// Check that SavedSearches map was initialized
	assert.NotNil(t, config.SavedSearches)
}

func TestLoad_WithoutConfigFile(t *testing.T) {
	// Save current working directory
	originalWd, _ := os.Getwd()

	// Change to temporary directory where no config files exist
	tempDir := t.TempDir()
	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Restore working directory after test
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Load should return default config when no file exists
	config, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Should have default values
	assert.Equal(t, 50, config.Defaults.MaxResults)
	assert.Equal(t, "default", config.Defaults.OutputFormat)
	assert.NotNil(t, config.SavedSearches)
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	// Should return error
	_, err = LoadFromFile(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestConfigSave_Default(t *testing.T) {
	// This test is tricky because Save() writes to user's config directory
	// We'll just verify that the method exists and basic error handling works
	config := defaultConfig()

	// Test SaveToFile with an invalid path (directory doesn't exist)
	err := config.SaveToFile("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}
