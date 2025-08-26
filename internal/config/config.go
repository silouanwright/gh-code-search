package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/silouanwright/gh-scout/internal/search"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration structure
type Config struct {
	Defaults      DefaultSettings        `yaml:"defaults" json:"defaults"`
	SavedSearches map[string]SavedSearch `yaml:"saved_searches" json:"saved_searches"`
	Analysis      AnalysisSettings       `yaml:"analysis" json:"analysis"`
	Output        OutputSettings         `yaml:"output" json:"output"`
	GitHub        GitHubSettings         `yaml:"github" json:"github"`
}

// DefaultSettings contains default values for search operations
type DefaultSettings struct {
	Language     string   `yaml:"language" json:"language"`
	MaxResults   int      `yaml:"max_results" json:"max_results"`
	ContextLines int      `yaml:"context_lines" json:"context_lines"`
	OutputFormat string   `yaml:"output_format" json:"output_format"`
	Editor       string   `yaml:"editor" json:"editor"`
	Repositories []string `yaml:"repositories" json:"repositories"`
	MinStars     int      `yaml:"min_stars" json:"min_stars"`
	SortBy       string   `yaml:"sort_by" json:"sort_by"`
}

// SavedSearch represents a saved search configuration
type SavedSearch struct {
	Name        string               `yaml:"name" json:"name"`
	Query       string               `yaml:"query" json:"query"`
	Filters     search.SearchFilters `yaml:"filters" json:"filters"`
	Description string               `yaml:"description" json:"description"`
	Created     time.Time            `yaml:"created" json:"created"`
	LastUsed    time.Time            `yaml:"last_used" json:"last_used"`
	UseCount    int                  `yaml:"use_count" json:"use_count"`
	Tags        []string             `yaml:"tags" json:"tags"`
}

// AnalysisSettings configures pattern analysis features
type AnalysisSettings struct {
	EnablePatterns   bool     `yaml:"enable_patterns" json:"enable_patterns"`
	MinPatternCount  int      `yaml:"min_pattern_count" json:"min_pattern_count"`
	ExcludeLanguages []string `yaml:"exclude_languages" json:"exclude_languages"`
	ExcludeTests     bool     `yaml:"exclude_tests" json:"exclude_tests"`
	PatternThreshold float64  `yaml:"pattern_threshold" json:"pattern_threshold"`
}

// OutputSettings configures output formatting and display
type OutputSettings struct {
	ColorMode       string `yaml:"color_mode" json:"color_mode"` // auto, always, never
	EditorCommand   string `yaml:"editor_command" json:"editor_command"`
	SavePath        string `yaml:"save_path" json:"save_path"`
	ShowPatterns    bool   `yaml:"show_patterns" json:"show_patterns"`
	MaxContentLines int    `yaml:"max_content_lines" json:"max_content_lines"`
	ShowRepository  bool   `yaml:"show_repository" json:"show_repository"`
	ShowStars       bool   `yaml:"show_stars" json:"show_stars"`
	ShowLineNumbers bool   `yaml:"show_line_numbers" json:"show_line_numbers"`
}

// GitHubSettings configures GitHub API behavior
type GitHubSettings struct {
	RateLimitBuffer int    `yaml:"rate_limit_buffer" json:"rate_limit_buffer"`
	Timeout         string `yaml:"timeout" json:"timeout"`
	RetryCount      int    `yaml:"retry_count" json:"retry_count"`
	CacheResults    bool   `yaml:"cache_results" json:"cache_results"`
	CacheTTL        string `yaml:"cache_ttl" json:"cache_ttl"`
}

// Load loads configuration from the standard locations
func Load() (*Config, error) {
	configPaths := getConfigPaths()

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			config, err := loadFromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
			}
			return config, nil
		}
	}

	// Return default configuration if no config file found
	return defaultConfig(), nil
}

// LoadFromFile loads configuration from a specific file path
func LoadFromFile(path string) (*Config, error) {
	return loadFromFile(path)
}

// Save saves the configuration to the default location
func (c *Config) Save() error {
	configDir := getConfigDir()
	// Use restrictive permissions for config directory (user-only)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "gh-scout.yaml")
	return c.SaveToFile(configPath)
}

// SaveToFile saves the configuration to a specific file path
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use restrictive permissions for config file (user read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddSavedSearch adds or updates a saved search
func (c *Config) AddSavedSearch(search SavedSearch) {
	if c.SavedSearches == nil {
		c.SavedSearches = make(map[string]SavedSearch)
	}

	// Update timestamps
	if existing, exists := c.SavedSearches[search.Name]; exists {
		search.Created = existing.Created
		search.UseCount = existing.UseCount
	} else {
		search.Created = time.Now()
		search.UseCount = 0
	}

	c.SavedSearches[search.Name] = search
}

// GetSavedSearch retrieves a saved search by name
func (c *Config) GetSavedSearch(name string) (SavedSearch, bool) {
	search, exists := c.SavedSearches[name]
	return search, exists
}

// UseSavedSearch marks a saved search as used (updates last used time and count)
func (c *Config) UseSavedSearch(name string) error {
	search, exists := c.SavedSearches[name]
	if !exists {
		return fmt.Errorf("saved search '%s' not found", name)
	}

	search.LastUsed = time.Now()
	search.UseCount++
	c.SavedSearches[name] = search

	return nil
}

// DeleteSavedSearch removes a saved search
func (c *Config) DeleteSavedSearch(name string) error {
	if _, exists := c.SavedSearches[name]; !exists {
		return fmt.Errorf("saved search '%s' not found", name)
	}

	delete(c.SavedSearches, name)
	return nil
}

// ListSavedSearches returns all saved searches sorted by last used
func (c *Config) ListSavedSearches() []SavedSearch {
	searches := make([]SavedSearch, 0, len(c.SavedSearches))

	for _, search := range c.SavedSearches {
		searches = append(searches, search)
	}

	// Sort by last used (most recent first)
	for i := 0; i < len(searches); i++ {
		for j := i + 1; j < len(searches); j++ {
			if searches[j].LastUsed.After(searches[i].LastUsed) {
				searches[i], searches[j] = searches[j], searches[i]
			}
		}
	}

	return searches
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate defaults
	if c.Defaults.MaxResults < 1 || c.Defaults.MaxResults > 1000 {
		return fmt.Errorf("defaults.max_results must be between 1 and 1000")
	}

	if c.Defaults.ContextLines < 0 || c.Defaults.ContextLines > 100 {
		return fmt.Errorf("defaults.context_lines must be between 0 and 100")
	}

	validFormats := map[string]bool{
		"default": true, "json": true, "markdown": true, "compact": true,
	}
	if !validFormats[c.Defaults.OutputFormat] {
		return fmt.Errorf("defaults.output_format must be one of: default, json, markdown, compact")
	}

	validSortBy := map[string]bool{
		"relevance": true, "stars": true, "updated": true, "created": true,
	}
	if !validSortBy[c.Defaults.SortBy] {
		return fmt.Errorf("defaults.sort_by must be one of: relevance, stars, updated, created")
	}

	// Validate output settings
	validColorModes := map[string]bool{
		"auto": true, "always": true, "never": true,
	}
	if !validColorModes[c.Output.ColorMode] {
		return fmt.Errorf("output.color_mode must be one of: auto, always, never")
	}

	if c.Output.MaxContentLines < 0 {
		return fmt.Errorf("output.max_content_lines must be non-negative")
	}

	// Validate analysis settings
	if c.Analysis.MinPatternCount < 1 {
		return fmt.Errorf("analysis.min_pattern_count must be at least 1")
	}

	if c.Analysis.PatternThreshold < 0 || c.Analysis.PatternThreshold > 1 {
		return fmt.Errorf("analysis.pattern_threshold must be between 0 and 1")
	}

	// Validate GitHub settings
	if c.GitHub.RateLimitBuffer < 0 {
		return fmt.Errorf("github.rate_limit_buffer must be non-negative")
	}

	if c.GitHub.RetryCount < 0 || c.GitHub.RetryCount > 10 {
		return fmt.Errorf("github.retry_count must be between 0 and 10")
	}

	return nil
}

// Private helper functions

// getConfigPaths returns the ordered list of config file paths to check
func getConfigPaths() []string {
	configDir := getConfigDir()
	homeDir, _ := os.UserHomeDir()

	paths := []string{
		".gh-scout.yaml", // Current directory
		".gh-scout.yml",  // Current directory (alternative)
		filepath.Join(configDir, "gh-scout.yaml"), // User config directory
		filepath.Join(configDir, "gh-scout.yml"),  // User config directory (alternative)
	}

	if homeDir != "" {
		paths = append(paths,
			filepath.Join(homeDir, ".gh-scout.yaml"), // Home directory
			filepath.Join(homeDir, ".gh-scout.yml"),  // Home directory (alternative)
		)
	}

	return paths
}

// getConfigDir returns the user configuration directory
func getConfigDir() string {
	if configDir := os.Getenv("GH_SEARCH_CONFIG_DIR"); configDir != "" {
		return configDir
	}

	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "gh-scout")
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".config", "gh-scout")
	}

	return "."
}

// loadFromFile loads configuration from a file
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values
	applyDefaults(&config)

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// defaultConfig returns a configuration with sensible defaults
func defaultConfig() *Config {
	config := &Config{
		Defaults: DefaultSettings{
			Language:     "",
			MaxResults:   50,
			ContextLines: 20,
			OutputFormat: "default",
			Editor:       "",
			Repositories: []string{},
			MinStars:     0,
			SortBy:       "relevance",
		},
		SavedSearches: make(map[string]SavedSearch),
		Analysis: AnalysisSettings{
			EnablePatterns:   true,
			MinPatternCount:  2,
			ExcludeLanguages: []string{},
			ExcludeTests:     true,
			PatternThreshold: 0.3,
		},
		Output: OutputSettings{
			ColorMode:       "auto",
			EditorCommand:   "",
			SavePath:        "",
			ShowPatterns:    true,
			MaxContentLines: 50,
			ShowRepository:  true,
			ShowStars:       true,
			ShowLineNumbers: false,
		},
		GitHub: GitHubSettings{
			RateLimitBuffer: 5,
			Timeout:         "30s",
			RetryCount:      3,
			CacheResults:    false,
			CacheTTL:        "1h",
		},
	}

	return config
}

// applyDefaults fills in missing values with defaults
func applyDefaults(config *Config) {
	defaults := defaultConfig()

	// Apply default values for zero/empty fields
	if config.Defaults.MaxResults == 0 {
		config.Defaults.MaxResults = defaults.Defaults.MaxResults
	}
	if config.Defaults.ContextLines == 0 {
		config.Defaults.ContextLines = defaults.Defaults.ContextLines
	}
	if config.Defaults.OutputFormat == "" {
		config.Defaults.OutputFormat = defaults.Defaults.OutputFormat
	}
	if config.Defaults.SortBy == "" {
		config.Defaults.SortBy = defaults.Defaults.SortBy
	}

	if config.Output.ColorMode == "" {
		config.Output.ColorMode = defaults.Output.ColorMode
	}
	if config.Output.MaxContentLines == 0 {
		config.Output.MaxContentLines = defaults.Output.MaxContentLines
	}

	if config.Analysis.MinPatternCount == 0 {
		config.Analysis.MinPatternCount = defaults.Analysis.MinPatternCount
	}
	if config.Analysis.PatternThreshold == 0 {
		config.Analysis.PatternThreshold = defaults.Analysis.PatternThreshold
	}

	if config.GitHub.Timeout == "" {
		config.GitHub.Timeout = defaults.GitHub.Timeout
	}
	if config.GitHub.CacheTTL == "" {
		config.GitHub.CacheTTL = defaults.GitHub.CacheTTL
	}

	// Initialize maps if nil
	if config.SavedSearches == nil {
		config.SavedSearches = make(map[string]SavedSearch)
	}
}
