package cmd

import (
	"fmt"
	"os"

	"github.com/silouanwright/gh-code-search/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose    bool
	dryRun     bool
	configFile string
	noColor    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-code-search",
	Short: "GitHub code search with intelligent filtering and analysis",
	Long: `Search GitHub's vast codebase to find working examples and configurations.

Perfect for discovering real-world usage patterns, configuration examples,
and best practices across millions of repositories.

Results include code context, repository information, and intelligent
ranking based on repository quality indicators.`,
	Example: `  # Find TypeScript configurations
  gh code-search "tsconfig.json" --language json --limit 10

  # Search React components with hooks
  gh code-search "useState" --language typescript --extension tsx

  # Find Docker configurations in popular repos
  gh code-search "dockerfile" --filename dockerfile --repo "**/react" --limit 5

  # Export results to file
  gh code-search "vite.config" --language javascript --output configs.md

  # Search multiple organizations
  gh code-search "eslint.config.js" --owner microsoft --owner google
  
  # Topic-based search workflow (combine with gh search repos)
  gh search repos --topic=react --json fullName -q '.[] | .fullName' > repos.txt
  gh code-search "hooks" --repos $(cat repos.txt | tr '\n' ',')`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags following gh-comment patterns
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output with detailed logging")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be searched without executing")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path (default: ~/.gh-code-search.yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Mark flags as hidden if they're primarily for debugging
	rootCmd.PersistentFlags().MarkHidden("dry-run")
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	var cfg *config.Config
	var err error
	
	// Load configuration from specified file or defaults
	if configFile != "" {
		cfg, err = config.LoadFromFile(configFile)
	} else {
		cfg, err = config.Load()
	}
	
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Config load failed, using defaults: %v\n", err)
		}
		// Load() already returns default config on failure, but we need a fallback
		cfg, _ = config.Load() // This will return defaults if no config exists
	}
	
	// Apply configuration defaults to CLI flags if not explicitly set
	applyConfigDefaults(cfg)
	
	if verbose {
		configPath := configFile
		if configPath == "" {
			configPath = "default locations"
		}
		fmt.Fprintf(os.Stderr, "gh-code-search initialized with config from: %s\n", configPath)
	}
}

// applyConfigDefaults applies configuration defaults to CLI flags if not explicitly set
func applyConfigDefaults(cfg *config.Config) {
	// Only apply config values if CLI flags weren't explicitly set
	// Note: This is a simplified implementation. Full implementation would check
	// if flags were explicitly set vs using defaults.
	
	// Apply search defaults
	if searchLimit == 50 && cfg.Defaults.MaxResults > 0 {
		searchLimit = cfg.Defaults.MaxResults
	}
	
	if outputFormat == "default" && cfg.Defaults.OutputFormat != "" {
		outputFormat = cfg.Defaults.OutputFormat
	}
	
	if searchLanguage == "" && cfg.Defaults.Language != "" {
		searchLanguage = cfg.Defaults.Language
	}
	
	if len(searchRepo) == 0 && len(cfg.Defaults.Repositories) > 0 {
		searchRepo = cfg.Defaults.Repositories
	}
	
	if minStars == 0 && cfg.Defaults.MinStars > 0 {
		minStars = cfg.Defaults.MinStars
	}
	
	// Apply output settings
	if !noColor && cfg.Output.ColorMode == "never" {
		noColor = true
	}
	
	// Note: Additional config applications can be added here as needed
	// This covers the most commonly used configuration options
}

// GetRootCmd returns the root command for testing
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// GetVerbose returns the verbose flag value
func GetVerbose() bool {
	return verbose
}

// GetDryRun returns the dry-run flag value
func GetDryRun() bool {
	return dryRun
}

// GetNoColor returns the no-color flag value
func GetNoColor() bool {
	return noColor
}