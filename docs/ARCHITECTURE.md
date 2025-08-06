# gh-code-search Architecture Specification

**Based on**: gh-comment's proven patterns and architecture  
**Language**: Go (following gh-comment's mature patterns)  
**Target**: Production-grade GitHub CLI extension  

## 🏗️ **Project Structure**

### **Directory Layout**
```
gh-code-search/
├── main.go                          # Entry point
├── go.mod, go.sum                   # Go modules
├── README.md                        # User documentation
├── LICENSE                          # MIT License
├── .gitignore                       # Go-specific ignores
├── .github/                         # GitHub Actions workflows
│   ├── workflows/
│   │   ├── test.yml                 # Test automation
│   │   ├── release.yml              # Release automation
│   │   └── benchmark.yml            # Performance tracking
├── cmd/                             # Command implementations
│   ├── root.go                      # Root command & global flags
│   ├── search.go                    # Main search command
│   ├── patterns.go                  # Pattern analysis
│   ├── saved.go                     # Saved searches management
│   ├── compare.go                   # Configuration comparison
│   ├── template.go                  # Template generation
│   ├── config.go                    # Configuration commands
│   ├── helpers.go                   # Shared command utilities
│   ├── colors.go                    # Color/UI utilities
│   └── *_test.go                    # Comprehensive test files
├── internal/                        # Internal packages
│   ├── github/                      # GitHub API integration
│   │   ├── client.go                # Interface definition
│   │   ├── real_client.go           # Production implementation
│   │   ├── mock_client.go           # Test implementation
│   │   ├── search.go                # Search API logic
│   │   ├── error_helper.go          # Intelligent error handling
│   │   └── *_test.go                # Client tests
│   ├── search/                      # Search engine
│   │   ├── query.go                 # Query building/parsing
│   │   ├── filters.go               # Search filters
│   │   ├── ranking.go               # Result ranking algorithms
│   │   ├── context.go               # Context extraction
│   │   └── *_test.go                # Search logic tests
│   ├── analysis/                    # Pattern analysis
│   │   ├── patterns.go              # Pattern detection
│   │   ├── comparison.go            # Configuration comparison
│   │   ├── templates.go             # Template generation
│   │   └── *_test.go                # Analysis tests
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Config structure & loader
│   │   ├── saved.go                 # Saved searches
│   │   ├── validation.go            # Config validation
│   │   └── *_test.go                # Config tests
│   ├── output/                      # Output formatting
│   │   ├── formatter.go             # Interface & implementations
│   │   ├── markdown.go              # Markdown output
│   │   ├── json.go                  # JSON output
│   │   ├── table.go                 # Table output
│   │   └── *_test.go                # Output tests
│   └── testutil/                    # Test utilities
│       ├── helpers.go               # Test helper functions
│       ├── fixtures.go              # Test data fixtures
│       └── golden.go                # Golden file testing
├── docs/                            # Documentation
│   ├── ADVANCED_USAGE.md            # Power user guide
│   ├── CONTRIBUTING.md              # Development guide
│   ├── PATTERNS.md                  # Common search patterns
│   └── API.md                       # GitHub API integration notes
├── examples/                        # Configuration examples
│   ├── .gh-code-search.yaml              # Example config
│   ├── saved-searches.yaml          # Example saved searches
│   └── templates/                   # Generated templates
├── scripts/                         # Build & utility scripts
│   ├── build.sh                     # Build script
│   ├── test.sh                      # Test script
│   └── install.sh                   # Installation script
└── test/                            # Integration tests
    ├── integration_test.go          # End-to-end tests
    └── testdata/                    # Test fixtures
        ├── searches/                # Sample search results
        └── configs/                 # Sample configurations
```

## 🔧 **Core Architecture Patterns**

### **1. Dependency Injection (From gh-comment)**
```go
// internal/github/client.go - Interface-based architecture
type GitHubAPI interface {
    SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error)
    GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error)
    GetRateLimit(ctx context.Context) (*RateLimit, error)
}

// Real implementation for production
type RealClient struct {
    client      *github.Client
    restClient  *http.Client
}

// Mock implementation for testing
type MockClient struct {
    SearchResults map[string]*SearchResults
    FileContents  map[string][]byte
    Errors        map[string]error
    CallLog       []string
}
```

### **2. Command Structure (Following gh-comment pattern)**
```go
// cmd/search.go - Main command with dependency injection
var (
    // Global client for dependency injection
    searchClient github.GitHubAPI
    
    // Command flags
    searchLimit     int
    searchLanguage  string
    searchRepo      []string
    searchFilename  string
    searchExtension string
    contextLines    int
    outputFormat    string
    saveAs          string
    dryRun          bool
    verbose         bool
)

var searchCmd = &cobra.Command{
    Use:   "search <query> [flags]",
    Short: "Search GitHub code with intelligent filtering and analysis",
    Long: heredoc.Doc(`
        Search GitHub's vast codebase to find working examples and configurations.
        
        Perfect for discovering real-world usage patterns, configuration examples,
        and best practices across millions of repositories.
        
        Results include code context, repository information, and intelligent
        ranking based on repository quality indicators.
    `),
    Example: heredoc.Doc(`
        # Find TypeScript configurations
        gh code-search "tsconfig.json" --language json --limit 10
        
        # Search React components with hooks
        gh code-search "useState" --language typescript --extension tsx
        
        # Find Docker configurations in popular repos
        gh code-search "dockerfile" --filename dockerfile --repo "**/react" --limit 5
        
        # Save a search for reuse
        gh code-search "vite.config" --language javascript --save vite-configs
        
        # Compare different approaches
        gh code-search "eslint.config.js" --compare --highlight-differences
    `),
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
    
    // Build search query from args and flags
    query := buildSearchQuery(args, cmd)
    
    // Execute search with error handling
    results, err := executeSearch(cmd.Context(), query)
    if err != nil {
        return handleSearchError(err, query)
    }
    
    // Process and output results
    return outputResults(results, outputFormat)
}
```

### **3. Intelligent Error Handling (Enhanced from gh-comment)**
```go
// cmd/helpers.go - Following gh-comment's formatActionableError pattern
func handleSearchError(err error, query string) error {
    errMsg := err.Error()
    
    // Rate limiting (most common issue)
    if strings.Contains(errMsg, "rate limit") {
        resetTime := extractRateLimitReset(err)
        return fmt.Errorf("GitHub search rate limit exceeded: %w\n\n💡 **Solutions**:\n  • Wait %s for automatic reset\n  • Use more specific search terms: --language, --repo, --filename\n  • Search specific repositories: --repo owner/repo\n  • Use saved searches: gh code-search --saved <name>\n\n📊 **Rate Limit Status**:\n  Run: gh code-search --rate-limit", err, resetTime)
    }
    
    // Invalid query syntax
    if strings.Contains(errMsg, "query") && strings.Contains(errMsg, "invalid") {
        return fmt.Errorf("invalid search query syntax: %w\n\n💡 **GitHub Search Syntax**:\n  • Exact phrases: \"exact match\"\n  • Boolean operators: config AND typescript\n  • Exclusions: config NOT test\n  • Wildcards: *.config.js\n  • File filters: filename:package.json\n  • Language filters: language:go\n\n📖 **Examples**:\n  gh code-search \"tsconfig.json\" --language json\n  gh code-search \"useEffect\" --language typescript --extension tsx", err)
    }
    
    // No results found
    if strings.Contains(errMsg, "no results") || strings.Contains(errMsg, "0 results") {
        return fmt.Errorf("no results found for query: %s\n\n💡 **Try These Approaches**:\n  • Broaden search terms: remove specific filters\n  • Check spelling and syntax\n  • Search popular repositories: --repo facebook/react\n  • Use broader language filters: --language javascript (not typescript)\n  • Try related terms: \"config\" instead of \"configuration\"\n\n🔍 **Search Tips**:\n  gh code-search --help    # See all available filters\n  gh code-search patterns  # Browse common search patterns", query)
    }
    
    // Permission/authentication errors  
    if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "permission") {
        return fmt.Errorf("GitHub authentication required: %w\n\n💡 **Fix Authentication**:\n  • Check status: gh auth status\n  • Re-authenticate: gh auth login\n  • Verify scopes: gh auth refresh\n\n📖 **Note**: Search requires authenticated GitHub CLI for higher rate limits", err)
    }
    
    // Network/connectivity issues
    if strings.Contains(errMsg, "network") || strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "connection") {
        return fmt.Errorf("network connectivity issue: %w\n\n💡 **Troubleshooting**:\n  • Check internet connection\n  • Verify GitHub status: https://status.github.com\n  • Try with --verbose for detailed logging\n  • Reduce request size: --limit 10\n\n🔧 **If persistent**:\n  gh code-search --debug <query>  # Enable debug logging", err)
    }
    
    // Generic fallback with helpful context
    return fmt.Errorf("search failed: %w\n\n💡 **General Troubleshooting**:\n  • Try with --verbose for detailed output\n  • Check GitHub status: https://status.github.com\n  • Verify authentication: gh auth status\n  • Use simpler query: remove complex filters\n\n📖 **Get Help**:\n  gh code-search --help     # Command documentation\n  gh code-search patterns   # Common search examples", err)
}
```

### **4. Configuration Management (Enhanced from gh-comment)**
```go
// internal/config/config.go
type Config struct {
    Defaults struct {
        Language     string   `yaml:"language" json:"language"`
        MaxResults   int      `yaml:"max_results" json:"max_results"`
        ContextLines int      `yaml:"context_lines" json:"context_lines"`
        OutputFormat string   `yaml:"output_format" json:"output_format"`
        Editor       string   `yaml:"editor" json:"editor"`
        Repositories []string `yaml:"repositories" json:"repositories"`
    } `yaml:"defaults" json:"defaults"`
    
    SavedSearches map[string]SavedSearch `yaml:"saved_searches" json:"saved_searches"`
    
    Analysis struct {
        EnablePatterns   bool     `yaml:"enable_patterns" json:"enable_patterns"`
        MinPatternCount  int      `yaml:"min_pattern_count" json:"min_pattern_count"`
        ExcludeLanguages []string `yaml:"exclude_languages" json:"exclude_languages"`
    } `yaml:"analysis" json:"analysis"`
    
    Output struct {
        ColorMode     string `yaml:"color_mode" json:"color_mode"`        // auto, always, never
        EditorCommand string `yaml:"editor_command" json:"editor_command"`
        SavePath      string `yaml:"save_path" json:"save_path"`
    } `yaml:"output" json:"output"`
    
    GitHub struct {
        RateLimitBuffer int    `yaml:"rate_limit_buffer" json:"rate_limit_buffer"`
        Timeout         string `yaml:"timeout" json:"timeout"`
        RetryCount      int    `yaml:"retry_count" json:"retry_count"`
    } `yaml:"github" json:"github"`
}

type SavedSearch struct {
    Name        string            `yaml:"name" json:"name"`
    Query       string            `yaml:"query" json:"query"`
    Filters     SearchFilters     `yaml:"filters" json:"filters"`
    Description string            `yaml:"description" json:"description"`
    Created     time.Time         `yaml:"created" json:"created"`
    LastUsed    time.Time         `yaml:"last_used" json:"last_used"`
    UseCount    int               `yaml:"use_count" json:"use_count"`
}

type SearchFilters struct {
    Language    string   `yaml:"language" json:"language"`
    Filename    string   `yaml:"filename" json:"filename"`
    Extension   string   `yaml:"extension" json:"extension"`
    Repository  []string `yaml:"repository" json:"repository"`
    Path        string   `yaml:"path" json:"path"`
    Size        string   `yaml:"size" json:"size"`
    Owner       []string `yaml:"owner" json:"owner"`
    MaxResults  int      `yaml:"max_results" json:"max_results"`
    ContextLines int     `yaml:"context_lines" json:"context_lines"`
}
```

### **5. Search Engine Architecture**
```go
// internal/search/query.go - Query building system
type QueryBuilder struct {
    terms     []string
    filters   map[string][]string
    qualifiers map[string]string
}

func NewQueryBuilder(terms []string) *QueryBuilder {
    return &QueryBuilder{
        terms:      terms,
        filters:    make(map[string][]string),
        qualifiers: make(map[string]string),
    }
}

func (qb *QueryBuilder) WithLanguage(lang string) *QueryBuilder {
    qb.qualifiers["language"] = lang
    return qb
}

func (qb *QueryBuilder) WithRepository(repos []string) *QueryBuilder {
    qb.filters["repo"] = repos
    return qb
}

func (qb *QueryBuilder) Build() string {
    var parts []string
    
    // Add main search terms
    if len(qb.terms) > 0 {
        parts = append(parts, strings.Join(qb.terms, " "))
    }
    
    // Add qualifiers (language:go, filename:config.json)
    for key, value := range qb.qualifiers {
        parts = append(parts, fmt.Sprintf("%s:%s", key, value))
    }
    
    // Add filters with multiple values
    for key, values := range qb.filters {
        for _, value := range values {
            parts = append(parts, fmt.Sprintf("%s:%s", key, value))
        }
    }
    
    return strings.Join(parts, " ")
}
```

### **6. Pattern Analysis Engine**
```go
// internal/analysis/patterns.go - Configuration pattern detection
type PatternAnalyzer struct {
    results []SearchResult
    config  *config.Config
}

type Pattern struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Frequency   int                    `json:"frequency"`
    Examples    []PatternExample       `json:"examples"`
    CommonProps map[string]interface{} `json:"common_properties"`
}

type PatternExample struct {
    Repository string `json:"repository"`
    Path       string `json:"path"`
    Snippet    string `json:"snippet"`
    Stars      int    `json:"stars"`
}

func (pa *PatternAnalyzer) AnalyzeConfigPatterns() ([]Pattern, error) {
    patterns := make(map[string]*Pattern)
    
    for _, result := range pa.results {
        // Extract configuration patterns from each result
        if configPattern := pa.extractConfigPattern(result); configPattern != nil {
            key := configPattern.Name
            if existing, found := patterns[key]; found {
                existing.Frequency++
                existing.Examples = append(existing.Examples, configPattern.Examples...)
            } else {
                patterns[key] = configPattern
            }
        }
    }
    
    return pa.rankPatterns(patterns), nil
}
```

## 🧪 **Testing Architecture (Following gh-comment's 85% coverage)**

### **Test Structure**
```go
// cmd/search_test.go - Table-driven tests for all scenarios
func TestSearchCommand(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        setupMock      func(*github.MockClient)
        wantErr        bool
        expectedOutput string
        expectedCalls  []string
    }{
        {
            name: "successful basic search",
            args: []string{"tsconfig.json"},
            setupMock: func(mock *github.MockClient) {
                mock.SetSearchResults("tsconfig.json", &github.SearchResults{
                    TotalCount: 1,
                    Items: []github.SearchItem{
                        {
                            Path: "tsconfig.json",
                            Repository: github.Repository{
                                FullName: "facebook/react",
                                HTMLURL:  "https://github.com/facebook/react",
                            },
                        },
                    },
                })
            },
            wantErr: false,
            expectedOutput: "Found 1 results",
            expectedCalls: []string{"SearchCode"},
        },
        {
            name: "rate limit error with helpful message",
            args: []string{"popular-query"},
            setupMock: func(mock *github.MockClient) {
                mock.SetError("SearchCode", &github.RateLimitError{
                    Message: "rate limit exceeded",
                    Rate: github.Rate{
                        Reset: github.Timestamp{Time: time.Now().Add(15 * time.Minute)},
                    },
                })
            },
            wantErr: true,
            expectedOutput: "rate limit exceeded",
        },
        // More comprehensive test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation with mock client
            originalClient := searchClient
            mockClient := github.NewMockClient()
            if tt.setupMock != nil {
                tt.setupMock(mockClient)
            }
            searchClient = mockClient
            defer func() { searchClient = originalClient }()
            
            // Execute command and verify results
            output, err := executeCommand(tt.args)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            if tt.expectedOutput != "" {
                assert.Contains(t, output, tt.expectedOutput)
            }
            
            // Verify expected API calls were made
            for _, expectedCall := range tt.expectedCalls {
                assert.Contains(t, mockClient.GetCallLog(), expectedCall)
            }
        })
    }
}
```

### **Integration Tests**
```go
// test/integration_test.go - End-to-end workflow testing
func TestSearchWorkflow(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/scripts",
        Setup: func(env *testscript.Env) error {
            env.Setenv("GH_TOKEN", "test-token")
            return nil
        },
    })
}
```

## 📊 **Performance & Quality Standards**

### **Performance Targets**
- **Search Response**: <2 seconds for typical queries
- **Pattern Analysis**: <5 seconds for 100 results
- **Memory Usage**: <50MB for large result sets
- **Startup Time**: <100ms

### **Quality Metrics** 
- **Test Coverage**: 85%+ (matching gh-comment)
- **Cyclomatic Complexity**: <15 per function
- **Documentation**: 100% public API coverage
- **Error Coverage**: All error paths tested

---

**Next**: See TESTING_STRATEGY.md for comprehensive testing approach and COMMAND_SPEC.md for detailed command specifications.