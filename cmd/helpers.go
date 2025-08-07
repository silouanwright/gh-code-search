package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/silouanwright/gh-code-search/internal/github"
)

// handleSearchError provides intelligent error handling with actionable guidance
// Following gh-comment's formatActionableError pattern
func handleSearchError(err error, query string) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// Rate limiting (most common issue)
	if rateLimitErr, ok := err.(*github.RateLimitError); ok {
		resetTime := formatDuration(rateLimitErr.ResetTime)
		return fmt.Errorf(`GitHub search rate limit exceeded: %w

💡 **Solutions**:
  • Wait %s for automatic reset
  • Use more specific search terms: --language, --repo, --filename
  • Search specific repositories: --repo owner/repo
  • Use saved searches: gh code-search saved list

📊 **Rate Limit Status**:
  • Limit: %d searches per hour
  • Remaining: %d
  • Reset: %s

🔧 **Try These Alternatives**:
  gh code-search "config" --repo facebook/react --language json
  gh code-search "tsconfig" --filename tsconfig.json --limit 10
  gh code-search --saved popular-configs`, err, resetTime, rateLimitErr.Limit, rateLimitErr.Remaining, resetTime)
	}

	// Abuse rate limiting
	if abuseErr, ok := err.(*github.AbuseRateLimitError); ok {
		retryAfter := "a few minutes"
		if abuseErr.RetryAfter != nil {
			retryAfter = formatDuration(*abuseErr.RetryAfter)
		}
		return fmt.Errorf(`GitHub abuse detection triggered: %w

💡 **What This Means**:
  • Your requests are being rate limited due to unusual activity
  • This is temporary and will resolve automatically

⏰ **Next Steps**:
  • Wait %s before retrying
  • Reduce search frequency
  • Use more specific queries to reduce load

🔧 **Optimize Your Searches**:
  gh code-search "specific term" --repo known/repo --language go
  gh code-search --filename package.json --path examples/`, err, retryAfter)
	}

	// Authentication errors
	if _, ok := err.(*github.AuthenticationError); ok {
		return fmt.Errorf(`GitHub authentication required: %w

💡 **Fix Authentication**:
  • Check current status: gh auth status
  • Login to GitHub: gh auth login
  • Refresh if needed: gh auth refresh --scopes repo

📈 **Benefits of Authentication**:
  • Higher rate limits (5,000 vs 60 requests/hour)
  • Access to private repositories (if permissions allow)
  • Detailed repository metadata

🚀 **After Authentication**:
  gh code-search "your query here"`, err)
	}

	// Authorization/Permission errors
	if _, ok := err.(*github.AuthorizationError); ok {
		return fmt.Errorf(`Access forbidden: %w

💡 **Possible Causes**:
  • Repository is private and you don't have access
  • Organization has restricted access policies
  • Token lacks required permissions

🔧 **Troubleshooting**:
  • Verify repository is public: visit the GitHub URL
  • Check organization membership
  • Re-authenticate: gh auth login --scopes repo
  • Contact repository owner for access

🔍 **Alternative Searches**:
  gh code-search "similar terms" --repo public/repo
  gh code-search "pattern" --language javascript --min-stars 100`, err)
	}

	// Resource not found
	if _, ok := err.(*github.NotFoundError); ok {
		return fmt.Errorf(`Resource not found: %w

💡 **Common Causes**:
  • Repository has been deleted or made private
  • File path has changed
  • Repository name has changed

🔧 **Try These Options**:
  • Check if repository still exists on GitHub
  • Search with broader terms: remove --repo filter
  • Use wildcards in repository names: --repo "**/react"
  • Search similar repositories: --repo facebook/* --repo vercel/*

🔍 **Broader Search**:
  gh code-search "your terms" --language typescript --min-stars 50`, err)
	}

	// Query validation errors
	if validationErr, ok := err.(*github.ValidationError); ok {
		return fmt.Errorf(`Invalid search query: %w

🔍 **Your Query**: %s
%s

💡 **GitHub Search Syntax**:
  • Exact phrases: "exact match"
  • Boolean operators: config AND typescript
  • Exclusions: config NOT test
  • Wildcards: *.config.js
  • File filters: filename:package.json
  • Language filters: language:go

📖 **Corrected Examples**:
  gh code-search "tsconfig.json" --language json
  gh code-search "useEffect" --language typescript --extension tsx
  gh code-search "dockerfile" --filename dockerfile --repo facebook/react

🚀 **Quick Fixes**:
  gh code-search "simplified terms" --language typescript
  gh code-search config --filename tsconfig.json
  gh code-search pattern --repo owner/repo --language go`, err, query, formatValidationErrors(validationErr.Errors))
	}

	// No results found
	if strings.Contains(errMsg, "no results") || strings.Contains(errMsg, "0 results") {
		return fmt.Errorf(`No results found for query: %s

💡 **Try These Approaches**:
  • Broaden search terms: "config" instead of "configuration-file-name"
  • Check spelling and remove typos
  • Search popular repositories: --repo facebook/react --repo microsoft/vscode
  • Use broader language filters: --language javascript (not typescript)
  • Try related terms: "setup", "options", "settings"

🔍 **Alternative Searches**:
  gh code-search "config" --language javascript --min-stars 100
  gh code-search "package.json" --path examples/ --limit 10
  gh code-search "typescript" --filename tsconfig.json

📖 **Browse Common Patterns**:
  gh code-search patterns --help    # See pattern analysis features
  gh code-search saved list         # Browse saved searches`, query)
	}

	// Network/connectivity issues
	if strings.Contains(errMsg, "network") || strings.Contains(errMsg, "timeout") ||
	   strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") {
		return fmt.Errorf(`Network connectivity issue: %w

💡 **Troubleshooting**:
  • Check internet connection
  • Verify GitHub status: https://status.github.com
  • Try with --verbose for detailed logging
  • Reduce request size: --limit 10

🔧 **If Persistent**:
  • Check firewall/proxy settings
  • Try different network connection
  • Wait a few minutes and retry

🚀 **Retry Commands**:
  gh code-search "simple query" --limit 5 --verbose
  gh code-search --rate-limit  # Check API status`, err)
	}

	// Client creation errors
	if strings.Contains(errMsg, "client") && strings.Contains(errMsg, "create") {
		return fmt.Errorf(`Failed to create GitHub client: %w

💡 **Authentication Setup**:
  • Install GitHub CLI: https://cli.github.com/
  • Authenticate: gh auth login
  • Verify: gh auth status

🔧 **Common Issues**:
  • GitHub CLI not installed or not in PATH
  • No GitHub authentication configured
  • Expired or invalid authentication token

📖 **Setup Steps**:
  1. Install gh CLI: brew install gh (macOS) or see docs
  2. Login: gh auth login --web
  3. Test: gh auth status
  4. Retry: gh code-search "test query"`, err)
	}

	// Generic fallback with helpful context
	return fmt.Errorf(`Search failed: %w

💡 **General Troubleshooting**:
  • Try with --verbose for detailed output
  • Check GitHub status: https://status.github.com
  • Verify authentication: gh auth status
  • Use simpler query: remove complex filters
  • Reduce result limit: --limit 10

📖 **Get Help**:
  gh code-search --help           # Command documentation
  gh code-search patterns --help  # Pattern analysis features
  gh code-search saved --help     # Saved searches management

🚀 **Common Solutions**:
  gh code-search "simple terms" --language go --limit 5
  gh code-search config --filename package.json
  gh code-search pattern --repo popular/repo`, err)
}

// handleClientError provides guidance for GitHub client creation failures
func handleClientError(err error) error {
	return fmt.Errorf(`Failed to create GitHub API client: %w

💡 **Authentication Required**:
  GitHub CLI must be installed and authenticated for gh-code-search to work.

🔧 **Setup Steps**:
  1. Install GitHub CLI: https://cli.github.com/
     • macOS: brew install gh
     • Windows: winget install GitHub.cli
     • Linux: See installation docs

  2. Authenticate with GitHub:
     gh auth login --web

  3. Verify authentication:
     gh auth status

  4. Test the connection:
     gh code-search "test query" --limit 1

📈 **Why Authentication?**:
  • Higher rate limits (5,000 vs 60 requests/hour)
  • Access to detailed repository metadata
  • Better error handling and diagnostics

🚀 **After Setup**:
  gh code-search "your search terms here"`, err)
}

// Helper functions for formatting

// formatDuration formats a duration in a human-friendly way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(d.Hours()/24), int(d.Hours())%24)
}

// formatValidationErrors formats validation error details
func formatValidationErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}

	var formatted []string
	for _, err := range errors {
		formatted = append(formatted, fmt.Sprintf("  ❌ %s", err))
	}
	return strings.Join(formatted, "\n")
}
