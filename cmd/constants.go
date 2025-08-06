package cmd

// Error messages and help text constants (refactored from ghx issue #3)
const (
	// Error message patterns
	ErrorMessageRateLimit         = "rate limit exceeded"
	ErrorMessageAuthentication   = "authentication required"
	ErrorMessageNoResults        = "No results found"
	ErrorMessageInvalidQuery     = "invalid query syntax"
	ErrorMessageNetworkTimeout   = "network timeout"
	ErrorMessageConnectionFailed = "connection failed"

	// Help text constants
	HelpTextRateLimitSolutions = `💡 **Solutions**:
  • Wait for rate limit reset
  • Use more specific filters to reduce API calls
  • Authenticate with GitHub CLI for higher limits

💡 **Examples**:
  gh code-search "config" --repo facebook/react --language json
  gh code-search "tsconfig" --filename tsconfig.json --limit 10

🔗 **More Info**: https://docs.github.com/en/rest/search#rate-limit`

	HelpTextAuthenticationGuidance = `🔧 **Quick Fix**:
  1. Install GitHub CLI: https://cli.github.com
  2. Authenticate: gh auth login --web
  3. Verify: gh auth status
  4. Retry your search

💡 **Examples**:
  gh code-search "specific term" --repo known/repo --language go

🔗 **GitHub CLI Setup**: https://docs.github.com/en/github-cli/github-cli/quickstart`

	HelpTextValidationSyntaxHelp = `💡 **GitHub Search Syntax**:
  • Exact phrases: "exact match"
  • Boolean operators: config AND typescript
  • Wildcards: *.config.js
  • Filters: language:go filename:main.go

💡 **Fixed Examples**:
  gh code-search "tsconfig.json" --language json
  gh code-search "useEffect" --language typescript --extension tsx
  gh code-search "dockerfile" --filename dockerfile --repo facebook/react

🔗 **Search Syntax Guide**: https://docs.github.com/en/search-github/searching-on-github/searching-code`

	HelpTextNoResultsGuidance = `💡 **Try These Alternatives**:
  • Broaden search terms: "config" instead of "configuration-file-name"
  • Remove filters that might be too restrictive
  • Check spelling and try related terms
  • Try related terms: "setup", "options", "settings"

💡 **Examples**:
  gh code-search "config" --language javascript --min-stars 100
  gh code-search "package.json" --path examples/ --limit 10
  gh code-search "typescript" --filename tsconfig.json

💡 **Pro Tip**: Start broad, then add filters to narrow down results`

	HelpTextNetworkRetryGuidance = `🔧 **Network Issue Detected**:
  • Check your internet connection
  • Verify GitHub.com is accessible
  • Try again in a few moments
  • Use --verbose for detailed network logs

💡 **Quick Test**:
  gh code-search "simple query" --limit 5 --verbose

🔗 **GitHub Status**: https://www.githubstatus.com`

	HelpTextClientSetupGuidance = `🔧 **GitHub CLI Setup Required**:
  1. Install GitHub CLI: https://cli.github.com
  2. Authenticate: gh auth login --web
  3. Verify access: gh auth status
  4. Retry: gh code-search "test query"

💡 **Alternative**: Set GITHUB_TOKEN environment variable

🔗 **Setup Guide**: https://docs.github.com/en/github-cli/github-cli/quickstart`

	// Search suggestions by query type
	SuggestionConfigQueries = `💡 **Config File Patterns**:
  gh code-search "config" --filename package.json
  gh code-search "tsconfig" --language json`

	SuggestionReactQueries = `💡 **React Development**:
  gh code-search "react" --language typescript --extension tsx
  gh code-search "useState" --repo facebook/react`

	// Common query patterns
	QueryPatternConfig = "config"
	QueryPatternReact  = "react"

	// Output format constants
	OutputFormatDefault  = "default"
	OutputFormatJSON     = "json"
	OutputFormatMarkdown = "markdown"
	OutputFormatCompact  = "compact"
	OutputFormatPipe     = "pipe"

	// GitHub API constants
	GitHubMaxResultsPerPage = 100
	GitHubSearchRateLimit   = 30

	// File extensions for language detection
	ExtensionGo         = ".go"
	ExtensionJavaScript = ".js"
	ExtensionTypeScript = ".ts"
	ExtensionTSX        = ".tsx"
	ExtensionPython     = ".py"
	ExtensionJSON       = ".json"
	ExtensionYAML       = ".yaml"
	ExtensionYML        = ".yml"
	ExtensionMarkdown   = ".md"
	ExtensionDockerfile = ".dockerfile"

	// Language identifiers
	LanguageGo         = "go"
	LanguageJavaScript = "javascript"
	LanguageTypeScript = "typescript"
	LanguagePython     = "python"
	LanguageJSON       = "json"
	LanguageYAML       = "yaml"
	LanguageMarkdown   = "markdown"
	LanguageDockerfile = "dockerfile"

	// Command success messages
	MessageConfigReset    = "✅ Configuration reset to defaults"
	MessageEditorReset    = "✅ Editor preference reset"
	MessageLanguageSet    = "✅ Default language set to: %s"
	MessageEditorSet      = "✅ Editor set to: %s"
	MessageFormatSet      = "✅ Default output format set to: %s"
	MessageSortSet        = "✅ Default sort order set to: %s"

	// Dry run messages
	MessageDryRunQuery = "Would search GitHub with query: %s"
	MessageVerboseQuery = "Searching GitHub with query: %s"
	MessageVerboseResults = "Found %d results"
)

// ValidOutputFormats contains all valid output format options
var ValidOutputFormats = []string{
	OutputFormatDefault,
	OutputFormatJSON,
	OutputFormatMarkdown,
	OutputFormatCompact,
}

// ValidSortOptions contains all valid sort options  
var ValidSortOptions = []string{
	"relevance",
	"stars", 
	"updated",
	"created",
}

// LanguageExtensionMap maps file extensions to language identifiers
var LanguageExtensionMap = map[string]string{
	ExtensionGo:         LanguageGo,
	ExtensionJavaScript: LanguageJavaScript,
	ExtensionTypeScript: LanguageTypeScript,
	ExtensionTSX:        LanguageTypeScript,
	ExtensionPython:     LanguagePython,
	ExtensionJSON:       LanguageJSON,
	ExtensionYAML:       LanguageYAML,
	ExtensionYML:        LanguageYAML,
	ExtensionMarkdown:   LanguageMarkdown,
}

// NetworkErrorPatterns contains patterns to identify network-related errors
var NetworkErrorPatterns = []string{
	"timeout", "connection", "network", "rate limit", "temporary", "retry",
}