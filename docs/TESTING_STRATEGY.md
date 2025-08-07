# gh-code-search Testing Strategy

**Target**: 85%+ test coverage (matching gh-comment's excellence)
**Approach**: Mock-first, table-driven, comprehensive error testing
**Tools**: Go testing, testify, testscript for integration

## 🎯 **Testing Philosophy (From gh-comment)**

### **Proven Patterns from gh-comment's Success:**
1. **Dependency Injection**: Every command uses interfaces for complete testability
2. **Mock-First Testing**: Isolated, fast, reliable tests without external dependencies
3. **Table-Driven Tests**: Comprehensive scenario coverage with clear test cases
4. **Error Path Testing**: Every error condition tested with helpful message validation
5. **Integration Testing**: End-to-end workflows with testscript
6. **Golden File Testing**: Output format verification
7. **Benchmark Testing**: Performance regression prevention

## 🏗️ **Test Architecture**

### **Test Structure (Following gh-comment pattern)**
```
test coverage by package:
├── cmd/                    # 85%+ target (command logic)
├── internal/github/        # 80%+ target (API integration)
├── internal/search/        # 90%+ target (core search logic)
├── internal/analysis/      # 85%+ target (pattern analysis)
├── internal/config/        # 95%+ target (configuration)
├── internal/output/        # 90%+ target (formatting)
└── test/                   # Integration tests (workflow coverage)
```

### **Testing Layers**

#### **Layer 1: Unit Tests - Isolated Function Testing**
```go
// cmd/search_test.go - Command logic testing
func TestBuildSearchQuery(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        flags    map[string]interface{}
        expected string
    }{
        {
            name:     "basic search terms",
            args:     []string{"tsconfig", "json"},
            flags:    map[string]interface{}{},
            expected: "tsconfig json",
        },
        {
            name:     "with language filter",
            args:     []string{"config"},
            flags:    map[string]interface{}{"language": "javascript"},
            expected: "config language:javascript",
        },
        {
            name:     "with multiple repos",
            args:     []string{"dockerfile"},
            flags:    map[string]interface{}{"repo": []string{"facebook/react", "vercel/next.js"}},
            expected: "dockerfile repo:facebook/react repo:vercel/next.js",
        },
        {
            name:     "complex query with all filters",
            args:     []string{"vite", "config"},
            flags:    map[string]interface{}{
                "language":  "typescript",
                "filename":  "vite.config.ts",
                "extension": "ts",
                "path":      "src/",
            },
            expected: "vite config language:typescript filename:vite.config.ts extension:ts path:src/",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := buildSearchQuery(tt.args, tt.flags)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### **Layer 2: Mock Client Testing - API Integration**
```go
// internal/github/mock_client.go - Complete API simulation
type MockClient struct {
    SearchResults  map[string]*SearchResults
    FileContents   map[string][]byte
    RateLimits     map[string]*RateLimit
    Errors         map[string]error
    CallLog        []MockCall
    ResponseDelay  time.Duration
}

type MockCall struct {
    Method    string
    Args      []interface{}
    Timestamp time.Time
}

func NewMockClient() *MockClient {
    return &MockClient{
        SearchResults: make(map[string]*SearchResults),
        FileContents:  make(map[string][]byte),
        RateLimits:    make(map[string]*RateLimit),
        Errors:        make(map[string]error),
        CallLog:       []MockCall{},
    }
}

// Set up mock responses
func (m *MockClient) SetSearchResults(query string, results *SearchResults) {
    m.SearchResults[query] = results
}

func (m *MockClient) SetFileContent(key string, content []byte) {
    m.FileContents[key] = content
}

func (m *MockClient) SetError(method string, err error) {
    m.Errors[method] = err
}

// API implementation with logging
func (m *MockClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
    m.logCall("SearchCode", query, opts)

    if err, exists := m.Errors["SearchCode"]; exists {
        return nil, err
    }

    if m.ResponseDelay > 0 {
        time.Sleep(m.ResponseDelay)
    }

    if results, exists := m.SearchResults[query]; exists {
        return results, nil
    }

    // Default empty results
    return &SearchResults{TotalCount: 0, Items: []SearchItem{}}, nil
}
```

#### **Layer 3: Command Testing - Full Command Execution**
```go
// cmd/search_test.go - Complete command testing with dependency injection
func TestSearchCommandExecution(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        setupMock      func(*github.MockClient)
        setupGlobals   func()
        wantErr        bool
        expectedOutput []string
        expectedFiles  []string
        verifyMock     func(*testing.T, *github.MockClient)
    }{
        {
            name: "successful search with results output",
            args: []string{"tsconfig.json", "--language", "json", "--limit", "5"},
            setupMock: func(mock *github.MockClient) {
                mock.SetSearchResults("tsconfig.json language:json", &github.SearchResults{
                    TotalCount: 3,
                    Items: []github.SearchItem{
                        {
                            Name: "tsconfig.json",
                            Path: "tsconfig.json",
                            Repository: github.Repository{
                                FullName: "facebook/react",
                                HTMLURL:  "https://github.com/facebook/react",
                                StargazersCount: 220000,
                            },
                            HTMLURL: "https://github.com/facebook/react/blob/main/tsconfig.json",
                            TextMatches: []github.TextMatch{
                                {
                                    Fragment: `"strict": true`,
                                    Matches: []github.Match{{Text: "strict"}},
                                },
                            },
                        },
                        // More test results...
                    },
                })
                mock.SetFileContent("facebook/react:tsconfig.json", []byte(`{
  "compilerOptions": {
    "strict": true,
    "target": "es2019",
    "module": "commonjs"
  }
}`))
            },
            setupGlobals: func() {
                searchLimit = 5
                searchLanguage = "json"
                outputFormat = "default"
                verbose = false
            },
            wantErr: false,
            expectedOutput: []string{
                "Found 3 results",
                "facebook/react",
                "tsconfig.json",
                "strict",
            },
            verifyMock: func(t *testing.T, mock *github.MockClient) {
                calls := mock.GetCallLog()
                assert.Len(t, calls, 2) // SearchCode + GetFileContent
                assert.Equal(t, "SearchCode", calls[0].Method)
                assert.Equal(t, "GetFileContent", calls[1].Method)
            },
        },
        {
            name: "rate limit error with intelligent response",
            args: []string{"popular-term"},
            setupMock: func(mock *github.MockClient) {
                mock.SetError("SearchCode", &github.RateLimitError{
                    Message: "API rate limit exceeded",
                    Rate: github.Rate{
                        Limit:     30,
                        Remaining: 0,
                        Reset:     github.Timestamp{Time: time.Now().Add(15 * time.Minute)},
                    },
                })
            },
            wantErr: true,
            expectedOutput: []string{
                "rate limit exceeded",
                "Wait 15m",
                "Solutions:",
                "more specific search terms",
            },
        },
        {
            name: "no results with helpful suggestions",
            args: []string{"extremely-rare-search-term-12345"},
            setupMock: func(mock *github.MockClient) {
                mock.SetSearchResults("extremely-rare-search-term-12345", &github.SearchResults{
                    TotalCount: 0,
                    Items:      []github.SearchItem{},
                })
            },
            wantErr: false,
            expectedOutput: []string{
                "No results found",
                "Try These Approaches",
                "broaden search terms",
                "check spelling",
            },
        },
        // More comprehensive test scenarios...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Save original state
            originalClient := searchClient
            originalLimit := searchLimit
            originalLanguage := searchLanguage

            // Set up test environment
            mockClient := github.NewMockClient()
            if tt.setupMock != nil {
                tt.setupMock(mockClient)
            }
            searchClient = mockClient

            if tt.setupGlobals != nil {
                tt.setupGlobals()
            }

            // Restore state after test
            defer func() {
                searchClient = originalClient
                searchLimit = originalLimit
                searchLanguage = originalLanguage
            }()

            // Capture output
            var buf bytes.Buffer
            cmd := searchCmd
            cmd.SetOut(&buf)
            cmd.SetErr(&buf)
            cmd.SetArgs(tt.args)

            // Execute command
            err := cmd.Execute()
            output := buf.String()

            // Verify results
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }

            for _, expected := range tt.expectedOutput {
                assert.Contains(t, output, expected, "Output should contain: %s", expected)
            }

            if tt.verifyMock != nil {
                tt.verifyMock(t, mockClient)
            }
        })
    }
}
```

#### **Layer 4: Error Handling Testing (Critical for UX)**
```go
// cmd/helpers_test.go - Following gh-comment's intelligent error testing
func TestHandleSearchError(t *testing.T) {
    tests := []struct {
        name            string
        inputError      error
        query           string
        expectedMessage []string
        expectedSugges  []string
    }{
        {
            name:       "rate limit error",
            inputError: &github.RateLimitError{Message: "rate limit exceeded"},
            query:      "popular-query",
            expectedMessage: []string{
                "rate limit exceeded",
                "Solutions:",
            },
            expectedSugges: []string{
                "more specific search terms",
                "wait",
                "search specific repositories",
            },
        },
        {
            name:       "invalid query syntax",
            inputError: fmt.Errorf("invalid query syntax: unexpected token"),
            query:      "malformed AND OR query",
            expectedMessage: []string{
                "invalid search query syntax",
                "GitHub Search Syntax",
            },
            expectedSugges: []string{
                "exact phrases",
                "Boolean operators",
                "Examples:",
            },
        },
        {
            name:       "authentication error",
            inputError: fmt.Errorf("unauthorized: bad credentials"),
            query:      "any-query",
            expectedMessage: []string{
                "GitHub authentication required",
                "Fix Authentication",
            },
            expectedSugges: []string{
                "gh auth status",
                "gh auth login",
                "gh auth refresh",
            },
        },
        {
            name:       "network connectivity error",
            inputError: fmt.Errorf("network timeout: connection failed"),
            query:      "any-query",
            expectedMessage: []string{
                "network connectivity issue",
                "Troubleshooting",
            },
            expectedSugges: []string{
                "check internet connection",
                "GitHub status",
                "--verbose",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := handleSearchError(tt.inputError, tt.query)

            assert.Error(t, result)
            errorMsg := result.Error()

            // Verify error message contains expected content
            for _, expected := range tt.expectedMessage {
                assert.Contains(t, errorMsg, expected)
            }

            // Verify helpful suggestions are included
            for _, suggestion := range tt.expectedSugges {
                assert.Contains(t, errorMsg, suggestion)
            }

            // Verify error maintains original context
            assert.Contains(t, errorMsg, tt.inputError.Error())
        })
    }
}
```

#### **Layer 5: Integration Testing - End-to-End Workflows**
```go
// test/integration_test.go - Real workflow testing with testscript
func TestIntegrationWorkflows(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/scripts",
        Setup: func(env *testscript.Env) error {
            // Set up test environment
            env.Setenv("GH_TOKEN", "test-token")
            env.Setenv("GH_SEARCH_CONFIG", env.WorkDir+"/config.yaml")
            return nil
        },
        Condition: func(cond string) (bool, error) {
            switch cond {
            case "has-gh":
                _, err := exec.LookPath("gh")
                return err == nil, nil
            default:
                return false, fmt.Errorf("unknown condition: %s", cond)
            }
        },
    })
}
```

#### **Integration Test Scripts (testdata/scripts/)**
```bash
# testdata/scripts/basic_search.txtar
# Test basic search functionality
> gh code-search "tsconfig.json" --language json --limit 2 --dry-run
stdout 'Would search: tsconfig.json language:json'
! stderr .

# Test with real API (when available)
[has-gh] gh code-search "package.json" --limit 1 --format json
[has-gh] stdout '"total_count":'
[has-gh] stdout '"repository":'

# testdata/scripts/saved_searches.txtar
# Test saved search workflow
> gh code-search save "react-configs" "tsconfig.json" --repo "*react*" --language json
stdout 'Saved search: react-configs'

> gh code-search --saved react-configs --dry-run
stdout 'Would search: tsconfig.json repo:*react* language:json'

> gh code-search list-saved
stdout 'react-configs'

# testdata/scripts/error_handling.txtar
# Test error scenarios
> ! gh code-search ""
stderr 'search query cannot be empty'
stderr 'Examples:'

> ! gh code-search "invalid query syntax !!!"
stderr 'invalid search query syntax'
stderr 'GitHub Search Syntax'
```

#### **Layer 6: Performance & Benchmark Testing**
```go
// cmd/benchmark_test.go - Performance regression prevention
func BenchmarkSearchExecution(b *testing.B) {
    mockClient := github.NewMockClient()
    mockClient.SetSearchResults("benchmark-query", &github.SearchResults{
        TotalCount: 100,
        Items:      generateBenchmarkResults(100),
    })

    searchClient = mockClient

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := executeSearch(context.Background(), "benchmark-query")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkPatternAnalysis(b *testing.B) {
    results := generateBenchmarkResults(1000)
    analyzer := analysis.NewPatternAnalyzer(results)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := analyzer.AnalyzeConfigPatterns()
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkLargeResultProcessing(b *testing.B) {
    results := generateBenchmarkResults(5000)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := processSearchResults(results, "markdown")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### **Layer 7: Golden File Testing - Output Verification**
```go
// internal/output/formatter_test.go - Output format verification
func TestMarkdownFormatter(t *testing.T) {
    tests := []struct {
        name       string
        results    *github.SearchResults
        goldenFile string
    }{
        {
            name: "basic search results",
            results: &github.SearchResults{
                TotalCount: 2,
                Items: []github.SearchItem{
                    {
                        Name: "tsconfig.json",
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
            },
            goldenFile: "basic_search_results.md",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            formatter := output.NewMarkdownFormatter()
            result, err := formatter.Format(tt.results)
            require.NoError(t, err)

            golden := testutil.LoadGoldenFile(t, tt.goldenFile)
            testutil.AssertGoldenMatch(t, golden, result, tt.goldenFile)
        })
    }
}
```

## 📊 **Test Coverage Targets (Matching gh-comment)**

### **Package Coverage Requirements**
```bash
cmd/                85%+  # Command implementations
internal/github/    80%+  # API integration
internal/search/    90%+  # Core search logic
internal/analysis/  85%+  # Pattern analysis
internal/config/    95%+  # Configuration management
internal/output/    90%+  # Output formatting
```

### **Coverage Verification**
```go
// scripts/test-coverage.sh
#!/bin/bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Verify minimum coverage per package
go tool cover -func=coverage.out | awk '
/cmd\// { if ($3+0 < 85) { print "cmd coverage too low: " $3; exit 1 } }
/internal\/search\// { if ($3+0 < 90) { print "search coverage too low: " $3; exit 1 } }
/internal\/config\// { if ($3+0 < 95) { print "config coverage too low: " $3; exit 1 } }
'

echo "✅ All coverage targets met"
```

## 🚨 **Critical Test Categories**

### **1. Error Path Testing (Essential for UX)**
- Rate limiting scenarios with helpful recovery suggestions
- Authentication failures with clear resolution steps
- Network issues with troubleshooting guidance
- Invalid query syntax with corrected examples
- No results scenarios with alternative approaches

### **2. Edge Case Testing**
- Empty search queries
- Extremely large result sets
- Special characters in queries
- Unicode handling
- Concurrent search requests

### **3. Configuration Testing**
- Valid/invalid YAML parsing
- Default value application
- Environment variable override
- Saved search persistence
- Configuration migration

### **4. API Integration Testing**
- Rate limit handling
- Response parsing
- Error condition mapping
- Timeout scenarios
- Retry logic

---

**Next**: See COMMAND_SPEC.md for detailed command specifications and MIGRATION_GUIDE.md for converting from TypeScript to Go.
