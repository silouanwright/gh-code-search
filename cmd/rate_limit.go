package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/silouanwright/gh-search/internal/github"
	"github.com/spf13/cobra"
)

// rateLimitCmd represents the rate-limit command
var rateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "Check current GitHub API rate limit status",
	Long: `Display current GitHub API rate limit status and usage information.

Shows remaining requests, limit, and reset time for the GitHub Search API.
Useful for understanding when you can make more requests after hitting limits.`,
	Example: `  # Check current rate limit status
  gh search rate-limit

  # Check rate limits after hitting a limit
  gh search rate-limit --verbose`,
	RunE: runRateLimit,
}

func runRateLimit(cmd *cobra.Command, args []string) error {
	// Initialize client if not set
	var client github.GitHubAPI
	if searchClient != nil {
		client = searchClient
	} else {
		realClient, err := github.NewRealClient()
		if err != nil {
			return handleClientError(err)
		}
		client = realClient
	}

	// Get rate limit info
	rateLimit, err := client.GetRateLimit(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get rate limit information: %w", err)
	}

	// Display rate limit status
	fmt.Println("📊 GitHub API Rate Limit Status")
	fmt.Println()

	// Basic status
	fmt.Printf("🔍 **Search API**:\n")
	fmt.Printf("  • Limit: %d requests per hour\n", rateLimit.Limit)
	fmt.Printf("  • Remaining: %d requests\n", rateLimit.Remaining)
	fmt.Printf("  • Reset: %s\n", rateLimit.Reset.Format("15:04:05 MST"))

	// Calculate time until reset
	timeUntilReset := time.Until(rateLimit.Reset)
	if timeUntilReset > 0 {
		fmt.Printf("  • Time until reset: %s\n", formatDuration(timeUntilReset))
	} else {
		fmt.Printf("  • Status: ✅ Reset time has passed\n")
	}

	// Usage percentage
	usedPercent := float64(rateLimit.Limit-rateLimit.Remaining) / float64(rateLimit.Limit) * 100
	fmt.Printf("  • Usage: %.1f%% (%d/%d)\n", usedPercent, rateLimit.Limit-rateLimit.Remaining, rateLimit.Limit)

	fmt.Println()

	// Status indicators and advice
	if rateLimit.Remaining == 0 {
		fmt.Println("🚨 **Rate Limit Exhausted**")
		fmt.Printf("You've used all %d requests. Wait %s for reset.\n", 
			rateLimit.Limit, formatDuration(timeUntilReset))
		fmt.Println()
		fmt.Println("💡 **While You Wait**:")
		fmt.Println("  • Use more specific filters: --language, --repo, --filename")
		fmt.Println("  • Try saved searches: gh search saved list")
		fmt.Println("  • Plan your searches to be more targeted")
	} else if rateLimit.Remaining < 5 {
		fmt.Println("⚠️  **Low on Requests**")
		fmt.Printf("Only %d requests remaining. Use them wisely!\n", rateLimit.Remaining)
		fmt.Println()
		fmt.Println("💡 **Conservation Tips**:")
		fmt.Println("  • Use --page instead of high --limit values")
		fmt.Println("  • Add filters to reduce result sets")
		fmt.Println("  • Save frequently used searches")
	} else if rateLimit.Remaining < rateLimit.Limit/2 {
		fmt.Println("📈 **Moderate Usage**")
		fmt.Printf("You have %d requests remaining (%.1f%% used).\n", 
			rateLimit.Remaining, usedPercent)
	} else {
		fmt.Println("✅ **Plenty of Requests Available**")
		fmt.Printf("You have %d requests remaining. Happy searching!\n", rateLimit.Remaining)
	}

	if verbose {
		fmt.Println()
		fmt.Println("🔧 **Technical Details**:")
		fmt.Printf("  • API Endpoint: Search API (code search)\n")
		fmt.Printf("  • Rate Limit Type: Per-user, per-hour\n")
		fmt.Printf("  • Reset Time: %s\n", rateLimit.Reset.Format(time.RFC3339))
		fmt.Printf("  • Current Time: %s\n", time.Now().Format(time.RFC3339))
		
		if timeUntilReset > 0 {
			fmt.Printf("  • Seconds until reset: %.0f\n", timeUntilReset.Seconds())
		}

		// Authentication status
		fmt.Println()
		fmt.Println("🔐 **Authentication Info**:")
		fmt.Println("  • Using GitHub CLI token authentication")
		fmt.Println("  • Higher limits available with authentication")
		fmt.Println("  • Run 'gh auth status' to verify authentication")
	}

	return nil
}

func init() {
	// Add rate-limit command to root
	rootCmd.AddCommand(rateLimitCmd)
}