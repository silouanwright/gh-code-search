package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/silouanwright/gh-code-search/internal/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gh-code-search configuration",
	Long: `Manage gh-code-search configuration settings.

View, modify, and reset configuration values to customize
gh-code-search behavior and preferences.`,
	Example: `  # View current configuration
  gh code-search config show

  # Reset editor preference
  gh code-search config reset defaults.editor

  # Reset all configuration to defaults  
  gh code-search config reset --all

  # Set default language
  gh code-search config set defaults.language typescript`,
}

// configShowCmd shows current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Display the current configuration values.

Shows all configuration settings including defaults, saved searches,
and output preferences.`,
	RunE: runConfigShow,
}

// configResetCmd resets configuration values
var configResetCmd = &cobra.Command{
	Use:   "reset [key]",
	Short: "Reset configuration values to defaults",
	Long: `Reset specific configuration values or entire configuration to defaults.

Use --all flag to reset all configuration, or specify a specific key
to reset only that value.`,
	Example: `  # Reset editor preference (fixes ghx issue #4)
  gh code-search config reset defaults.editor

  # Reset output format
  gh code-search config reset defaults.output_format

  # Reset everything
  gh code-search config reset --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigReset,
}

// configSetCmd sets configuration values
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Long: `Set a specific configuration value.

Supported configuration keys include defaults.editor, defaults.language,
defaults.max_results, and more.`,
	Example: `  # Set default editor (addresses ghx issue #4)
  gh code-search config set defaults.editor "code"

  # Set default language
  gh code-search config set defaults.language "typescript"

  # Set default result limit
  gh code-search config set defaults.max_results 25`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var (
	resetAll bool
)

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Println("🔧 Current gh-code-search configuration:")
	fmt.Println()

	// Show defaults
	fmt.Println("📋 Defaults:")
	fmt.Printf("  language: %s\n", cfg.Defaults.Language)
	fmt.Printf("  max_results: %d\n", cfg.Defaults.MaxResults)
	fmt.Printf("  context_lines: %d\n", cfg.Defaults.ContextLines)
	fmt.Printf("  output_format: %s\n", cfg.Defaults.OutputFormat)
	fmt.Printf("  editor: %s\n", cfg.Defaults.Editor)
	fmt.Printf("  min_stars: %d\n", cfg.Defaults.MinStars)
	fmt.Printf("  sort_by: %s\n", cfg.Defaults.SortBy)
	fmt.Println()

	// Show output settings
	fmt.Println("🎨 Output:")
	fmt.Printf("  color_mode: %s\n", cfg.Output.ColorMode)
	fmt.Printf("  show_patterns: %t\n", cfg.Output.ShowPatterns)
	fmt.Printf("  show_stars: %t\n", cfg.Output.ShowStars)
	fmt.Printf("  show_line_numbers: %t\n", cfg.Output.ShowLineNumbers)
	fmt.Printf("  max_content_lines: %d\n", cfg.Output.MaxContentLines)
	fmt.Println()

	// Show GitHub settings
	fmt.Println("🐙 GitHub API:")
	fmt.Printf("  timeout: %s\n", cfg.GitHub.Timeout)
	fmt.Printf("  retry_count: %d\n", cfg.GitHub.RetryCount)
	fmt.Printf("  rate_limit_buffer: %d\n", cfg.GitHub.RateLimitBuffer)
	fmt.Println()

	// Show saved searches count
	if len(cfg.SavedSearches) > 0 {
		fmt.Printf("💾 Saved searches: %d\n", len(cfg.SavedSearches))
		for name, search := range cfg.SavedSearches {
			fmt.Printf("  - %s: %s\n", name, search.Description)
		}
	} else {
		fmt.Println("💾 Saved searches: none")
	}

	return nil
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	if resetAll {
		// Reset entire configuration by removing existing config files
		configPaths := []string{
			".gh-code-search.yaml",
			".gh-code-search.yml",
		}
		
		// Also check user config directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			configPaths = append(configPaths,
				homeDir+"/.config/gh-code-search/gh-code-search.yaml",
				homeDir+"/.config/gh-code-search/gh-code-search.yml",
				homeDir+"/.gh-code-search.yaml",
				homeDir+"/.gh-code-search.yml",
			)
		}
		
		// Remove existing config files
		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove config file %s: %w", path, err)
				}
			}
		}
		
		fmt.Println("✅ All configuration reset to defaults")
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a configuration key to reset or use --all flag")
	}

	key := args[0]
	
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Reset specific key to default value
	switch key {
	case "defaults.editor":
		cfg.Defaults.Editor = ""
		fmt.Println("✅ Editor preference reset")
	case "defaults.language":
		cfg.Defaults.Language = ""
		fmt.Println("✅ Default language reset")
	case "defaults.output_format":
		cfg.Defaults.OutputFormat = "default"
		fmt.Println("✅ Default output format reset")
	case "defaults.max_results":
		cfg.Defaults.MaxResults = 50
		fmt.Println("✅ Default max results reset to 50")
	case "defaults.context_lines":
		cfg.Defaults.ContextLines = 20
		fmt.Println("✅ Default context lines reset to 20")
	case "defaults.min_stars":
		cfg.Defaults.MinStars = 0
		fmt.Println("✅ Default min stars reset to 0")
	case "defaults.sort_by":
		cfg.Defaults.SortBy = "relevance"
		fmt.Println("✅ Default sort order reset to relevance")
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set specific key value
	switch key {
	case "defaults.editor":
		cfg.Defaults.Editor = value
		fmt.Printf("✅ Editor set to: %s\n", value)
	case "defaults.language":
		cfg.Defaults.Language = value
		fmt.Printf("✅ Default language set to: %s\n", value)
	case "defaults.output_format":
		validFormats := []string{"default", "json", "markdown", "compact"}
		if !contains(validFormats, value) {
			return fmt.Errorf("invalid output format: %s (valid: %s)", value, strings.Join(validFormats, ", "))
		}
		cfg.Defaults.OutputFormat = value
		fmt.Printf("✅ Default output format set to: %s\n", value)
	case "defaults.sort_by":
		validSorts := []string{"relevance", "stars", "updated", "created"}
		if !contains(validSorts, value) {
			return fmt.Errorf("invalid sort order: %s (valid: %s)", value, strings.Join(validSorts, ", "))
		}
		cfg.Defaults.SortBy = value
		fmt.Printf("✅ Default sort order set to: %s\n", value)
	default:
		return fmt.Errorf("unknown or unsupported configuration key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// Helper function to check if slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func init() {
	// Add config command to root
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configSetCmd)

	// Add flags
	configResetCmd.Flags().BoolVar(&resetAll, "all", false, "reset all configuration to defaults")
}