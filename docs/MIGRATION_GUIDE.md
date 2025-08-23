# gh-scout Migration Guide

**From**: ghx TypeScript monolith
**To**: gh-scout Go CLI extension with gh-comment's proven patterns
**Goal**: Professional-grade GitHub search with 85%+ test coverage

## 🔄 **Migration Overview**

### **Current State: ghx (TypeScript)**
- **Architecture**: Single file (`src/index.ts`, 400+ lines)
- **Dependencies**: Heavy Node.js ecosystem (yargs, conf, execa, etc.)
- **Testing**: Basic integration tests with vitest
- **Distribution**: npm package with global install
- **Features**: GitHub code search with basic filtering

### **Target State: gh-scout (Go)**
- **Architecture**: Modular Go CLI with dependency injection
- **Dependencies**: Minimal (cobra, github.com/google/go-github)
- **Testing**: Comprehensive with 85%+ coverage, table-driven tests
- **Distribution**: GitHub CLI extension (gh extension install)
- **Features**: Enhanced search, pattern analysis, saved searches, templates

## 📋 **Step-by-Step Migration Process**

### **Phase 1: Foundation Setup (Week 1)**

#### **1.1 Initialize Go Project**
```bash
# Create new repository
mkdir gh-scout
cd gh-scout
go mod init github.com/silouanwright/gh-scout

# Set up basic structure
mkdir -p cmd internal/{github,search,config,output} test docs examples
```

#### **1.2 Core Dependencies**
```go
// go.mod
module github.com/silouanwright/gh-scout

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/google/go-github/v57 v57.0.0
    golang.org/x/oauth2 v0.15.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/stretchr/testify v1.8.4
    github.com/rogpeppe/go-internal v1.11.0 // for testscript
)
```

#### **1.3 GitHub API Client Interface**
```go
// internal/github/client.go
type GitHubAPI interface {
    SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error)
    GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error)
    GetRateLimit(ctx context.Context) (*RateLimit, error)
}

type SearchOptions struct {
    Sort      string
    Order     string
    ListOptions ListOptions
}

type SearchResults struct {
    Total *int            `json:"total_count,omitempty"`
    Items []SearchItem    `json:"items,omitempty"`
}

type SearchItem struct {
    Name        string     `json:"name,omitempty"`
    Path        string     `json:"path,omitempty"`
    SHA         string     `json:"sha,omitempty"`
    URL         string     `json:"url,omitempty"`
    GitURL      string     `json:"git_url,omitempty"`
    HTMLURL     string     `json:"html_url,omitempty"`
    Repository  Repository `json:"repository,omitempty"`
    Score       float64    `json:"score,omitempty"`
    TextMatches []TextMatch `json:"text_matches,omitempty"`
}
```

### **Phase 2: Command Migration (Week 1-2)**

#### **2.1 Root Command Structure**
```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "gh-scout",
    Short: "GitHub code search with intelligent filtering and analysis",
    Long: `Search GitHub's vast codebase to find working examples and configurations.

Perfect for discovering real-world usage patterns, configuration examples,
and best practices across millions of repositories.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Global flags
    rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
    rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be searched")
    rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path")
}
```

#### **2.2 Main Search Command**
```go
// cmd/search.go - Convert from ghx's main functionality
var (
    searchClient github.GitHubAPI

    // Flags migrated from ghx
    searchLanguage  string
    searchRepo      []string
    searchFilename  string
    searchExtension string
    searchPath      string
    searchOwner     []string
    searchSize      string
    searchLimit     int
    contextLines    int
    outputFormat    string
    saveAs          string
)

var searchCmd = &cobra.Command{
    Use:   "search <query> [flags]",
    Short: "Search GitHub code with intelligent filtering",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
    // Initialize client (dependency injection for testing)
    if searchClient == nil {
        client, err := createGitHubClient()
        if err != nil {
            return handleClientError(err)
        }
        searchClient = client
    }

    // Build query from args and flags (migrate ghx logic)
    query := buildSearchQuery(args)

    // Execute search with gh-comment style error handling
    results, err := executeSearch(cmd.Context(), query)
    if err != nil {
        return handleSearchError(err, query)
    }

    // Process and output results
    return outputResults(results)
}

func init() {
    // Migrate all ghx flags
    searchCmd.Flags().StringVarP(&searchLanguage, "language", "l", "", "programming language")
    searchCmd.Flags().StringSliceVarP(&searchRepo, "repo", "r", nil, "repository filter")
    searchCmd.Flags().StringVarP(&searchFilename, "filename", "f", "", "filename filter")
    searchCmd.Flags().StringVarP(&searchExtension, "extension", "e", "", "file extension")
    searchCmd.Flags().StringVarP(&searchPath, "path", "p", "", "file path filter")
    searchCmd.Flags().StringSliceVarP(&searchOwner, "owner", "o", nil, "repository owner")
    searchCmd.Flags().StringVar(&searchSize, "size", "", "file size filter")
    searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum results")
    searchCmd.Flags().IntVar(&contextLines, "context", 20, "context lines around matches")
    searchCmd.Flags().StringVar(&outputFormat, "format", "default", "output format")
    searchCmd.Flags().StringVar(&saveAs, "save", "", "save search with name")

    rootCmd.AddCommand(searchCmd)
}
```

### **Phase 3: Core Logic Migration (Week 2)**

#### **3.1 Query Building (from ghx)**
```go
// internal/search/query.go - Migrate ghx's query building logic
func buildSearchQuery(terms []string) string {
    var parts []string

    // Add search terms
    if len(terms) > 0 {
        parts = append(parts, strings.Join(terms, " "))
    }

    // Add language filter (from ghx --language)
    if searchLanguage != "" {
        parts = append(parts, fmt.Sprintf("language:%s", searchLanguage))
    }

    // Add filename filter (from ghx --filename)
    if searchFilename != "" {
        parts = append(parts, fmt.Sprintf("filename:%s", searchFilename))
    }

    // Add extension filter (from ghx --extension)
    if searchExtension != "" {
        parts = append(parts, fmt.Sprintf("extension:%s", searchExtension))
    }

    // Add repository filters (from ghx --repo)
    for _, repo := range searchRepo {
        parts = append(parts, fmt.Sprintf("repo:%s", repo))
    }

    // Add path filter (from ghx --path)
    if searchPath != "" {
        parts = append(parts, fmt.Sprintf("path:%s", searchPath))
    }

    // Add owner filters (from ghx --owner)
    for _, owner := range searchOwner {
        parts = append(parts, fmt.Sprintf("user:%s", owner))
    }

    // Add size filter (from ghx --size)
    if searchSize != "" {
        parts = append(parts, fmt.Sprintf("size:%s", searchSize))
    }

    return strings.Join(parts, " ")
}
```

#### **3.2 GitHub API Integration**
```go
// internal/github/real_client.go - Replace ghx's direct API calls
type RealClient struct {
    client *github.Client
}

func NewRealClient(token string) *RealClient {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(context.Background(), ts)
    return &RealClient{
        client: github.NewClient(tc),
    }
}

func (c *RealClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
    // Convert options
    searchOpts := &github.SearchOptions{
        Sort:        opts.Sort,
        Order:       opts.Order,
        ListOptions: github.ListOptions{Page: opts.ListOptions.Page, PerPage: opts.ListOptions.PerPage},
    }

    // Execute search
    result, resp, err := c.client.Search.Code(ctx, query, searchOpts)
    if err != nil {
        return nil, formatGitHubError(err, resp)
    }

    // Convert results
    return convertSearchResults(result), nil
}
```

### **Phase 4: Testing Migration (Week 2-3)**

#### **4.1 Mock Client (from gh-comment pattern)**
```go
// internal/github/mock_client.go
type MockClient struct {
    SearchResults map[string]*SearchResults
    FileContents  map[string][]byte
    Errors        map[string]error
    CallLog       []MockCall
}

func (m *MockClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
    m.logCall("SearchCode", query, opts)

    if err, exists := m.Errors["SearchCode"]; exists {
        return nil, err
    }

    if results, exists := m.SearchResults[query]; exists {
        return results, nil
    }

    return &SearchResults{Total: intPtr(0), Items: []SearchItem{}}, nil
}
```

#### **4.2 Test Migration from ghx**
```go
// cmd/search_test.go - Convert ghx integration tests to unit tests
func TestSearchCommand(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        setupMock      func(*github.MockClient)
        expectedQuery  string
        expectedOutput string
        wantErr        bool
    }{
        {
            name: "typescript config search (from ghx test)",
            args: []string{"strict"},
            setupMock: func(mock *github.MockClient) {
                searchCmd.Flags().Set("filename", "tsconfig.json")
                searchCmd.Flags().Set("limit", "2")

                mock.SetSearchResults("strict filename:tsconfig.json", &github.SearchResults{
                    Total: intPtr(2),
                    Items: []github.SearchItem{
                        {
                            Path: "tsconfig.json",
                            Repository: github.Repository{
                                FullName: "facebook/react",
                                HTMLURL:  "https://github.com/facebook/react",
                            },
                            TextMatches: []github.TextMatch{
                                {Fragment: `"strict": true`},
                            },
                        },
                    },
                })
            },
            expectedQuery:  "strict filename:tsconfig.json",
            expectedOutput: "strict",
            wantErr:        false,
        },
        {
            name: "react components search (from ghx test)",
            args: []string{"useState"},
            setupMock: func(mock *github.MockClient) {
                searchCmd.Flags().Set("language", "typescript")
                searchCmd.Flags().Set("extension", "tsx")
                searchCmd.Flags().Set("limit", "1")

                mock.SetSearchResults("useState language:typescript extension:tsx", &github.SearchResults{
                    Total: intPtr(1),
                    Items: []github.SearchItem{
                        {
                            Path: "components/Button.tsx",
                            Repository: github.Repository{
                                FullName: "vercel/next.js",
                            },
                        },
                    },
                })
            },
            expectedQuery: "useState language:typescript extension:tsx",
            wantErr:       false,
        },
        // Convert all ghx tests to table-driven format...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set up mock client
            originalClient := searchClient
            mockClient := github.NewMockClient()
            if tt.setupMock != nil {
                tt.setupMock(mockClient)
            }
            searchClient = mockClient
            defer func() { searchClient = originalClient }()

            // Execute command
            output, err := executeCommand(tt.args)

            // Verify results
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }

            if tt.expectedOutput != "" {
                assert.Contains(t, output, tt.expectedOutput)
            }

            // Verify correct API calls
            calls := mockClient.GetCallLog()
            assert.Len(t, calls, 1)
            assert.Equal(t, "SearchCode", calls[0].Method)
        })
    }
}
```

### **Phase 5: Enhanced Features (Week 3-4)**

#### **5.1 Pattern Analysis (New Feature)**
```go
// cmd/patterns.go - New command for pattern analysis
var patternsCmd = &cobra.Command{
    Use:   "patterns <query> [flags]",
    Short: "Analyze common configuration patterns",
    RunE:  runPatterns,
}

func runPatterns(cmd *cobra.Command, args []string) error {
    // Execute search
    results, err := executeSearch(cmd.Context(), buildSearchQuery(args))
    if err != nil {
        return err
    }

    // Analyze patterns
    analyzer := analysis.NewPatternAnalyzer(results.Items)
    patterns, err := analyzer.AnalyzeConfigPatterns()
    if err != nil {
        return fmt.Errorf("pattern analysis failed: %w", err)
    }

    // Output patterns
    return outputPatterns(patterns)
}
```

#### **5.2 Saved Searches (Enhanced from ghx concept)**
```go
// cmd/saved.go - Enhanced saved search management
var savedCmd = &cobra.Command{
    Use:   "saved <command>",
    Short: "Manage saved searches",
}

var savedRunCmd = &cobra.Command{
    Use:   "run <name>",
    Short: "Execute a saved search",
    Args:  cobra.ExactArgs(1),
    RunE:  runSavedSearch,
}

func runSavedSearch(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    savedSearch, exists := cfg.SavedSearches[args[0]]
    if !exists {
        return fmt.Errorf("saved search '%s' not found", args[0])
    }

    // Apply saved search parameters
    applySearchFilters(savedSearch.Filters)

    // Execute search
    return runSearch(cmd, []string{savedSearch.Query})
}
```

### **Phase 6: Output Migration (Week 4)**

#### **6.1 Markdown Output (Enhanced from ghx)**
```go
// internal/output/markdown.go - Enhanced from ghx's basic output
type MarkdownFormatter struct {
    ShowLineNumbers bool
    ContextLines    int
    HighlightTerms  []string
}

func (f *MarkdownFormatter) Format(results *github.SearchResults) (string, error) {
    var buf strings.Builder

    // Header with summary
    buf.WriteString(fmt.Sprintf("🔍 Found %d results\n\n", *results.Total))

    // Format each result
    for _, item := range results.Items {
        // Repository header with stars
        buf.WriteString(fmt.Sprintf("## 📁 [%s](%s) ⭐ %d\n",
            item.Repository.FullName,
            item.Repository.HTMLURL,
            item.Repository.StargazersCount))

        // File path
        buf.WriteString(fmt.Sprintf("📄 **%s**\n\n", item.Path))

        // Code content with syntax highlighting
        if len(item.TextMatches) > 0 {
            for _, match := range item.TextMatches {
                buf.WriteString("```" + detectLanguage(item.Path) + "\n")
                buf.WriteString(f.formatTextMatch(match))
                buf.WriteString("\n```\n")
            }
        }

        // Link to file
        buf.WriteString(fmt.Sprintf("🔗 [View on GitHub](%s)\n\n", item.HTMLURL))
    }

    return buf.String(), nil
}
```

### **Phase 7: Configuration Migration (Week 4)**

#### **7.1 Config File Format**
```go
// internal/config/config.go - Enhanced from ghx's simple config
type Config struct {
    Defaults struct {
        Language     string   `yaml:"language"`
        MaxResults   int      `yaml:"max_results"`
        ContextLines int      `yaml:"context_lines"`
        OutputFormat string   `yaml:"output_format"`
        Repositories []string `yaml:"repositories"`
    } `yaml:"defaults"`

    SavedSearches map[string]SavedSearch `yaml:"saved_searches"`

    Analysis struct {
        MinPatternCount int `yaml:"min_pattern_count"`
        EnablePatterns  bool `yaml:"enable_patterns"`
    } `yaml:"analysis"`
}

func Load() (*Config, error) {
    // Load from ~/.gh-scout.yaml or .gh-scout.yaml
    configPaths := []string{
        ".gh-scout.yaml",
        filepath.Join(os.Getenv("HOME"), ".gh-scout.yaml"),
    }

    for _, path := range configPaths {
        if _, err := os.Stat(path); err == nil {
            return loadFromFile(path)
        }
    }

    // Return default config
    return defaultConfig(), nil
}
```

## 🚨 **Critical Migration Points**

### **Error Handling Enhancement**
```go
// Enhanced from ghx's basic error handling
func handleSearchError(err error, query string) error {
    errMsg := strings.ToLower(err.Error())

    // Rate limiting (more detailed than ghx)
    if strings.Contains(errMsg, "rate limit") {
        return fmt.Errorf(`GitHub search rate limit exceeded: %w

💡 **Solutions**:
  • Wait for rate limit reset (check: gh scout --rate-limit)
  • Use more specific search terms: --language, --repo, --filename
  • Search specific repositories: --repo owner/repo
  • Use saved searches: gh scout saved list

📊 **Current Limits**:
  • Authenticated: 30 searches/minute
  • Unauthenticated: 10 searches/minute

🔧 **Try These Alternatives**:
  gh scout "config" --repo facebook/react --language json
  gh scout saved run popular-configs`, err)
    }

    // More intelligent error handling...
    return err
}
```

### **Testing Strategy Migration**
```go
// Convert ghx's integration tests to comprehensive unit tests
func TestMigratedGhxFunctionality(t *testing.T) {
    // All ghx test cases converted to table-driven tests
    ghxTests := []struct {
        description string
        ghxCommand  string
        goArgs      []string
        setupMock   func(*github.MockClient)
        verify      func(*testing.T, string)
    }{
        {
            description: "TypeScript config search",
            ghxCommand:  "pnpm node dist/index.js --filename tsconfig.json --pipe strict --limit 2",
            goArgs:      []string{"strict", "--filename", "tsconfig.json", "--limit", "2"},
            // ... setup and verification
        },
        // Convert all 20+ ghx tests...
    }

    for _, tt := range ghxTests {
        t.Run(tt.description, func(t *testing.T) {
            // Execute with same expected behavior as ghx
        })
    }
}
```

## 📈 **Progress Tracking**

### **Migration Checklist**
- [ ] **Week 1**: Go project setup, basic commands, GitHub API integration
- [ ] **Week 2**: Core search functionality, query building, basic testing
- [ ] **Week 3**: Enhanced features, pattern analysis, saved searches
- [ ] **Week 4**: Output formatting, configuration, comprehensive testing
- [ ] **Week 5**: Integration testing, documentation, CI/CD setup
- [ ] **Week 6**: Performance optimization, error handling refinement
- [ ] **Week 7**: Beta testing, feedback integration, polish
- [ ] **Week 8**: Release preparation, gh extension packaging

### **Feature Parity Verification**
```bash
# Verify all ghx functionality works in gh-scout
gh scout "tsconfig.json" --filename tsconfig.json --limit 2
gh scout "useState" --language typescript --extension tsx --limit 1
gh scout "useState" --repo facebook/react --limit 1
gh scout "hooks" --language typescript --repo facebook/react --limit 1
gh scout "dependencies" --filename package.json --limit 1
gh scout "class" --size ">1000" --language typescript --limit 1
gh scout "Button" --path src/components --extension tsx --limit 1
gh scout "interface" --context 50 --language typescript --limit 1
```

### **Quality Gates**
- ✅ **All ghx functionality replicated**
- ✅ **85%+ test coverage achieved**
- ✅ **Error handling enhanced with actionable guidance**
- ✅ **Performance matches or exceeds ghx**
- ✅ **Documentation comprehensive with examples**
- ✅ **CI/CD pipeline established**

---

**Result**: A production-ready GitHub CLI extension that maintains ghx's core value while adding professional architecture, comprehensive testing, and enhanced features following gh-comment's proven patterns.
