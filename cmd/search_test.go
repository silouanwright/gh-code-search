package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/silouanwright/gh-code-search/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchCommand tests the main search command functionality
func TestSearchCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupMock      func(*github.MockClient)
		setupFlags     func()
		expectedQuery  string
		expectedOutput string
		wantErr        bool
		checkCalls     func(*testing.T, *github.MockClient)
	}{
		{
			name: "basic search with simple query",
			args: []string{"typescript"},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("typescript", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("microsoft/TypeScript", "src/compiler/types.ts", "interface CompilerOptions"),
				))
			},
			expectedQuery:  "typescript",
			expectedOutput: "microsoft/TypeScript",
			wantErr:        false,
		},
		{
			name: "search with language filter (migrated from ghx)",
			args: []string{"config"},
			setupFlags: func() {
				searchLanguage = "json"
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("config language:json", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("facebook/react", "tsconfig.json", `{"compilerOptions": {"strict": true}}`),
				))
			},
			expectedQuery:  "config language:json",
			expectedOutput: "facebook/react",
			wantErr:        false,
		},
		{
			name: "search with filename filter (migrated from ghx)",
			args: []string{"strict"},
			setupFlags: func() {
				searchFilename = "tsconfig.json"
				searchLimit = 2
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("strict filename:tsconfig.json", github.CreateTestSearchResults(2,
					github.CreateTestSearchItem("facebook/react", "tsconfig.json", `"strict": true`),
					github.CreateTestSearchItem("vercel/next.js", "packages/next/tsconfig.json", `"strict": false`),
				))
			},
			expectedQuery:  "strict filename:tsconfig.json",
			expectedOutput: "strict",
			wantErr:        false,
		},
		{
			name: "search with repository filter (migrated from ghx)",
			args: []string{"useState"},
			setupFlags: func() {
				searchRepo = []string{"facebook/react"}
				searchLanguage = "typescript"
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("useState language:typescript repo:facebook/react", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("facebook/react", "packages/react/src/ReactHooks.ts", "function useState"),
				))
			},
			expectedQuery:  "useState language:typescript repo:facebook/react",
			expectedOutput: "useState",
			wantErr:        false,
		},
		{
			name: "search with multiple filters (comprehensive ghx test)",
			args: []string{"hooks"},
			setupFlags: func() {
				searchLanguage = "typescript"
				searchExtension = "tsx"
				searchRepo = []string{"facebook/react", "vercel/next.js"}
				minStars = 1000
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("hooks language:typescript extension:tsx stars:>=1000 repo:facebook/react repo:vercel/next.js",
					github.CreateTestSearchResults(2,
						github.CreateTestSearchItem("facebook/react", "packages/react/src/Component.tsx", "useEffect hook"),
						github.CreateTestSearchItem("vercel/next.js", "examples/with-hooks/pages/index.tsx", "useState hook"),
					))
			},
			expectedQuery:  "hooks language:typescript extension:tsx stars:>=1000 repo:facebook/react repo:vercel/next.js",
			expectedOutput: "useEffect",
			wantErr:        false,
		},
		{
			name: "search with owner filter (migrated from ghx)",
			args: []string{"interface"},
			setupFlags: func() {
				searchOwner = []string{"microsoft"}
				searchLanguage = "typescript"
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("interface language:typescript user:microsoft", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("microsoft/vscode", "src/vs/base/common/types.ts", "interface IDisposable"),
				))
			},
			expectedQuery:  "interface language:typescript user:microsoft",
			expectedOutput: "microsoft/vscode",
			wantErr:        false,
		},
		{
			name: "search with size filter (migrated from ghx)",
			args: []string{"class"},
			setupFlags: func() {
				searchSize = ">1000"
				searchLanguage = "typescript"
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("class language:typescript size:>1000", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("microsoft/TypeScript", "src/compiler/checker.ts", "class TypeChecker"),
				))
			},
			expectedQuery:  "class language:typescript size:>1000",
			expectedOutput: "TypeChecker",
			wantErr:        false,
		},
		{
			name: "search with path filter (migrated from ghx)",
			args: []string{"Component"},
			setupFlags: func() {
				searchPath = "src/components"
				searchLanguage = "typescript"
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("Component language:typescript path:src/components", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("facebook/react", "src/components/Button.tsx", "export default Component"),
				))
			},
			expectedQuery:  "Component language:typescript path:src/components",
			expectedOutput: "Button.tsx",
			wantErr:        false,
		},
		{
			name: "pipe output format (migrated from ghx)",
			args: []string{"config"},
			setupFlags: func() {
				pipe = true
				searchLimit = 1
			},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("config", github.CreateTestSearchResults(1,
					github.CreateTestSearchItem("example/repo", "config.json", "{}"),
				))
			},
			expectedQuery:  "config",
			expectedOutput: "example/repo:config.json:https://github.com/example/repo/blob/main/config.json",
			wantErr:        false,
		},
		{
			name: "rate limit error with helpful message",
			args: []string{"popular-query"},
			setupMock: func(mock *github.MockClient) {
				mock.SetError("SearchCode", &github.RateLimitError{
					Message:   "rate limit exceeded",
					ResetTime: 15 * time.Minute,
					Limit:     30,
					Remaining: 0,
				})
			},
			wantErr:        true,
			expectedOutput: "rate limit exceeded",
		},
		{
			name: "authentication error with guidance",
			args: []string{"test-query"},
			setupMock: func(mock *github.MockClient) {
				mock.SetError("SearchCode", &github.AuthenticationError{
					Message: "authentication required",
				})
			},
			wantErr:        true,
			expectedOutput: "authentication required",
		},
		{
			name: "validation error with syntax help",
			args: []string{"invalid AND OR query"},
			setupMock: func(mock *github.MockClient) {
				mock.SetError("SearchCode", &github.ValidationError{
					Message: "invalid query syntax",
					Errors:  []string{"unexpected token 'OR' after 'AND'"},
				})
			},
			wantErr:        true,
			expectedOutput: "invalid query syntax",
		},
		{
			name: "no results found",
			args: []string{"extremely-rare-search-term-12345"},
			setupMock: func(mock *github.MockClient) {
				mock.SetSearchResults("extremely-rare-search-term-12345", github.CreateTestSearchResults(0))
			},
			expectedOutput: "No results found",
			wantErr:        false,
		},
		{
			name: "dry run mode shows query without executing",
			args: []string{"test-query"},
			setupFlags: func() {
				dryRun = true
			},
			setupMock: func(mock *github.MockClient) {
				// Should not be called in dry-run mode
			},
			expectedQuery:  "test-query",
			expectedOutput: "Would search GitHub with query: test-query",
			wantErr:        false,
			checkCalls: func(t *testing.T, mock *github.MockClient) {
				assert.Equal(t, 0, mock.GetCallCount("SearchCode"), "SearchCode should not be called in dry-run mode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			resetSearchFlags()

			// Set up mock client
			originalClient := searchClient
			mockClient := github.NewMockClient()
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}
			searchClient = mockClient
			defer func() {
				searchClient = originalClient
				resetSearchFlags()
			}()

			// Apply flag setup
			if tt.setupFlags != nil {
				tt.setupFlags()
			}

			// Create and execute command
			cmd := searchCmd
			cmd.SetArgs(tt.args)

			// Capture output
			output := captureOutput(func() error {
				// Set context on command
				cmd.SetContext(context.Background())
				return runSearch(cmd, tt.args)
			})

			// Verify error expectation
			if tt.wantErr {
				assert.Error(t, output.err, "Expected error but got none")
				if tt.expectedOutput != "" {
					assert.Contains(t, output.err.Error(), tt.expectedOutput, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, output.err, "Unexpected error: %v", output.err)
				if tt.expectedOutput != "" {
					assert.Contains(t, output.stdout, tt.expectedOutput, "Output should contain expected text")
				}
			}

			// Verify API calls if not in error cases
			if !tt.wantErr && !dryRun {
				assert.Equal(t, 1, mockClient.GetCallCount("SearchCode"), "Should make exactly one search call")

				// Verify query construction
				if tt.expectedQuery != "" {
					lastCall := mockClient.GetLastCall()
					require.NotNil(t, lastCall, "Should have recorded API call")
					assert.Equal(t, "SearchCode", lastCall.Method)
					if len(lastCall.Args) > 0 {
						actualQuery, ok := lastCall.Args[0].(string)
						require.True(t, ok, "First argument should be query string")
						assert.Equal(t, tt.expectedQuery, actualQuery, "Query should match expected")
					}
				}
			}

			// Custom call verification
			if tt.checkCalls != nil {
				tt.checkCalls(t, mockClient)
			}
		})
	}
}

// TestBuildSearchQuery tests the query building logic (migrated from ghx)
func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		terms    []string
		setup    func()
		expected string
	}{
		{
			name:     "simple terms only",
			terms:    []string{"react", "hooks"},
			expected: "react hooks",
		},
		{
			name:  "with language filter",
			terms: []string{"config"},
			setup: func() {
				searchLanguage = "json"
			},
			expected: "config language:json",
		},
		{
			name:  "with filename filter",
			terms: []string{"strict"},
			setup: func() {
				searchFilename = "tsconfig.json"
			},
			expected: "strict filename:tsconfig.json",
		},
		{
			name:  "with multiple repo filters",
			terms: []string{"hooks"},
			setup: func() {
				searchRepo = []string{"facebook/react", "vercel/next.js"}
			},
			expected: "hooks repo:facebook/react repo:vercel/next.js",
		},
		{
			name:  "with all filters (comprehensive)",
			terms: []string{"component"},
			setup: func() {
				searchLanguage = "typescript"
				searchFilename = "*.tsx"
				searchExtension = "tsx"
				searchRepo = []string{"facebook/react"}
				searchPath = "src/components"
				searchOwner = []string{"facebook"}
				searchSize = ">100"
				minStars = 1000
			},
			expected: "component language:typescript filename:*.tsx extension:tsx path:src/components size:>100 stars:>=1000 repo:facebook/react user:facebook",
		},
		{
			name:  "empty terms with filters only",
			terms: []string{},
			setup: func() {
				searchLanguage = "go"
				searchFilename = "main.go"
			},
			expected: "language:go filename:main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetSearchFlags()
			defer resetSearchFlags()

			// Apply setup
			if tt.setup != nil {
				tt.setup()
			}

			// Build query
			result := buildSearchQuery(tt.terms)

			// Verify result
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestOutputFormats tests different output formatting options
func TestOutputFormats(t *testing.T) {
	mockResults := github.CreateTestSearchResults(2,
		github.CreateTestSearchItem("facebook/react", "src/ReactHooks.ts", "function useState()"),
		github.CreateTestSearchItem("vercel/next.js", "packages/next/package.json", `"name": "next"`),
	)

	tests := []struct {
		name           string
		setupFlags     func()
		expectedOutput string
	}{
		{
			name: "default output format",
			setupFlags: func() {
				pipe = false
			},
			expectedOutput: "📁 [facebook/react]",
		},
		{
			name: "pipe output format",
			setupFlags: func() {
				pipe = true
			},
			expectedOutput: "facebook/react:src/ReactHooks.ts:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSearchFlags()
			defer resetSearchFlags()

			if tt.setupFlags != nil {
				tt.setupFlags()
			}

			output := captureOutput(func() error {
				return outputResults(mockResults)
			})

			assert.NoError(t, output.err)
			assert.Contains(t, output.stdout, tt.expectedOutput)
		})
	}
}

// TestDetectLanguage tests language detection from file paths
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"src/main.go", "go"},
		{"components/Button.tsx", "typescript"},
		{"utils/helper.js", "javascript"},
		{"config/settings.json", "json"},
		{"docker/Dockerfile", "dockerfile"},
		{"scripts/build.sh", ""},
		{"README.md", "markdown"},
		{"styles/main.css", ""},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectLanguage(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestErrorHandling tests various error scenarios with helpful messages
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		query          string
		expectedOutput string
	}{
		{
			name: "rate limit error provides actionable guidance",
			error: &github.RateLimitError{
				Message:   "rate limit exceeded",
				ResetTime: 10 * time.Minute,
				Limit:     30,
				Remaining: 0,
			},
			query:          "popular query",
			expectedOutput: "💡 **Solutions**",
		},
		{
			name: "authentication error provides setup guidance",
			error: &github.AuthenticationError{
				Message: "authentication required",
			},
			query:          "test query",
			expectedOutput: "gh auth login",
		},
		{
			name: "validation error provides syntax help",
			error: &github.ValidationError{
				Message: "invalid query",
				Errors:  []string{"unexpected token"},
			},
			query:          "invalid AND OR query",
			expectedOutput: "💡 **GitHub Search Syntax**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleSearchError(tt.error, tt.query)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedOutput)
		})
	}
}

// Helper functions for testing

type capturedOutput struct {
	stdout string
	stderr string
	err    error
}

// captureOutput captures stdout/stderr during function execution
func captureOutput(fn func() error) capturedOutput {
	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes for stdout and stderr
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()

	// Redirect stdout and stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Execute function
	err := fn()

	// Close write ends
	stdoutW.Close()
	stderrW.Close()

	// Restore original stdout and stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = io.Copy(&stdoutBuf, stdoutR)
	_, _ = io.Copy(&stderrBuf, stderrR)

	return capturedOutput{
		stdout: stdoutBuf.String(),
		stderr: stderrBuf.String(),
		err:    err,
	}
}

// resetSearchFlags resets all search-related global flags to their defaults
func resetSearchFlags() {
	searchLanguage = ""
	searchRepo = nil
	searchFilename = ""
	searchExtension = ""
	searchPath = ""
	searchOwner = nil
	searchSize = ""
	searchLimit = 50
	contextLines = 20
	outputFormat = "default"
	pipe = false
	minStars = 0
	sort = "relevance"
	order = "desc"

	// Reset global flags
	dryRun = false
	verbose = false
}

// Additional integration-style tests following gh-comment patterns

// TestSearchCommandIntegration tests end-to-end command execution
func TestSearchCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with actual command construction (but still mocked client)
	mockClient := github.NewMockClient()
	mockClient.SetSearchResults("test language:go", github.CreateTestSearchResults(1,
		github.CreateTestSearchItem("test/repo", "test.go", "package main"),
	))

	originalClient := searchClient
	searchClient = mockClient
	defer func() { searchClient = originalClient }()

	// Reset flags before test
	resetSearchFlags()
	defer resetSearchFlags()

	// Set flags manually for integration test
	searchLanguage = "go"
	searchLimit = 1

	// Execute search command directly
	err := runSearch(searchCmd, []string{"test"})
	assert.NoError(t, err)

	// Verify expected API interaction
	assert.Equal(t, 1, mockClient.GetCallCount("SearchCode"))
	assert.True(t, mockClient.VerifyCall("SearchCode", "test language:go"))
}

// TestLimitFunctionality tests limit functionality including pagination (Issue #2 from ghx)
func TestLimitFunctionality(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		mockSetup      func(*github.MockClient)
		expectedCalls  int
		expectedItems  int
		description    string
	}{
		{
			name:  "limit under 100 - single API call",
			limit: 50,
			mockSetup: func(mock *github.MockClient) {
				mock.SetSearchResults("test", github.CreateTestSearchResults(50,
					createMultipleTestItems(50)...,
				))
			},
			expectedCalls: 1,
			expectedItems: 50,
			description:   "Should make single API call for limits ≤ 100",
		},
		{
			name:  "limit exactly 100 - single API call",
			limit: 100,
			mockSetup: func(mock *github.MockClient) {
				mock.SetSearchResults("test", github.CreateTestSearchResults(100,
					createMultipleTestItems(100)...,
				))
			},
			expectedCalls: 1,
			expectedItems: 100,
			description:   "Should make single API call for limit of exactly 100",
		},
		{
			name:  "limit over 100 - multiple API calls (pagination)",
			limit: 150,
			mockSetup: func(mock *github.MockClient) {
				// First call returns 100 items
				mock.SetPaginatedSearchResults("test", map[int]*github.SearchResults{
					1: github.CreateTestSearchResults(100, createMultipleTestItems(100)...),
					2: github.CreateTestSearchResults(50, createMultipleTestItems(50)...),
				})
			},
			expectedCalls: 2,
			expectedItems: 150,
			description:   "Should paginate for limits > 100",
		},
		{
			name:  "limit over 200 - three API calls",
			limit: 250,
			mockSetup: func(mock *github.MockClient) {
				mock.SetPaginatedSearchResults("test", map[int]*github.SearchResults{
					1: github.CreateTestSearchResults(100, createMultipleTestItems(100)...),
					2: github.CreateTestSearchResults(100, createMultipleTestItems(100)...),
					3: github.CreateTestSearchResults(50, createMultipleTestItems(50)...),
				})
			},
			expectedCalls: 3,
			expectedItems: 250,
			description:   "Should handle multiple pages for large limits",
		},
		{
			name:  "partial results - stops when fewer returned",
			limit: 200,
			mockSetup: func(mock *github.MockClient) {
				// First call returns 100, second returns only 25 (indicating end of results)
				mock.SetPaginatedSearchResults("test", map[int]*github.SearchResults{
					1: github.CreateTestSearchResults(100, createMultipleTestItems(100)...),
					2: github.CreateTestSearchResults(25, createMultipleTestItems(25)...),
				})
			},
			expectedCalls: 2,
			expectedItems: 125, // Only 125 total items available
			description:   "Should stop pagination when fewer results returned than requested",
		},
		{
			name:  "maximum limit 1000 - ten API calls",
			limit: 1000,
			mockSetup: func(mock *github.MockClient) {
				pages := make(map[int]*github.SearchResults)
				for i := 1; i <= 10; i++ {
					pages[i] = github.CreateTestSearchResults(100, createMultipleTestItems(100)...)
				}
				mock.SetPaginatedSearchResults("test", pages)
			},
			expectedCalls: 10,
			expectedItems: 1000,
			description:   "Should handle maximum limit of 1000 results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetSearchFlags()
			defer resetSearchFlags()

			// Set limit
			searchLimit = tt.limit

			// Set up mock client
			mockClient := github.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			originalClient := searchClient
			searchClient = mockClient
			defer func() { searchClient = originalClient }()

			// Execute search
			results, err := executeSearch(context.Background(), "test")

			// Verify no error
			assert.NoError(t, err, tt.description)

			// Verify correct number of API calls
			assert.Equal(t, tt.expectedCalls, mockClient.GetCallCount("SearchCode"),
				"Expected %d API calls for limit %d (%s)", tt.expectedCalls, tt.limit, tt.description)

			// Verify correct number of items returned
			if results != nil {
				assert.Equal(t, tt.expectedItems, len(results.Items),
					"Expected %d items for limit %d (%s)", tt.expectedItems, tt.limit, tt.description)
			}

			// Verify pagination parameters were correct
			calls := mockClient.GetAllCalls()
			for i, call := range calls {
				if call.Method == "SearchCode" && len(call.Args) >= 2 {
					if opts, ok := call.Args[1].(*github.SearchOptions); ok {
						expectedPage := i + 1
						assert.Equal(t, expectedPage, opts.ListOptions.Page,
							"Call %d should be for page %d", i+1, expectedPage)

						// PerPage should be min(remaining, 100)
						remaining := tt.limit - (i * 100)
						expectedPerPage := remaining
						if expectedPerPage > 100 {
							expectedPerPage = 100
						}
						assert.Equal(t, expectedPerPage, opts.ListOptions.PerPage,
							"Call %d should request %d items per page", i+1, expectedPerPage)
					}
				}
			}
		})
	}
}

// createMultipleTestItems creates multiple test search items for pagination testing
func createMultipleTestItems(count int) []github.SearchItem {
	items := make([]github.SearchItem, count)
	for i := 0; i < count; i++ {
		items[i] = github.CreateTestSearchItem(
			fmt.Sprintf("repo%d/test", i+1),
			fmt.Sprintf("file%d.go", i+1),
			fmt.Sprintf("content for item %d", i+1),
		)
	}
	return items
}

// TestTextMatchesInResults tests that we properly request and display code fragments
func TestTextMatchesInResults(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		mockSetup      func(*github.MockClient)
		expectedOutput string
		description    string
	}{
		{
			name: "search results include code fragments",
			args: []string{"useState"},
			mockSetup: func(mock *github.MockClient) {
				// Create search item with text matches (code fragments)
				item := github.CreateTestSearchItem("facebook/react", "src/hooks.tsx", "function useState")
				item.TextMatches = []github.TextMatch{
					{
						Fragment: github.StringPtr("export function useState<T>(initialValue: T): [T, SetState<T>] {\n  const [state, setState] = React.useState(initialValue);\n  return [state, setState];\n}"),
						Property: github.StringPtr("content"),
					},
				}
				mock.SetSearchResults("useState", github.CreateTestSearchResults(1, item))
			},
			expectedOutput: "export function useState",
			description:    "Should display code fragments from TextMatches",
		},
		{
			name: "multiple text matches per file",
			args: []string{"console.log"},
			mockSetup: func(mock *github.MockClient) {
				item := github.CreateTestSearchItem("example/repo", "debug.js", "console logging")
				item.TextMatches = []github.TextMatch{
					{
						Fragment: github.StringPtr("console.log('Starting application');"),
						Property: github.StringPtr("content"),
					},
					{
						Fragment: github.StringPtr("console.log('Error occurred:', error);"),
						Property: github.StringPtr("content"),
					},
				}
				mock.SetSearchResults("console.log", github.CreateTestSearchResults(1, item))
			},
			expectedOutput: "console.log('Starting application')",
			description:    "Should display multiple code fragments when present",
		},
		{
			name: "language detection for syntax highlighting",
			args: []string{"interface"},
			mockSetup: func(mock *github.MockClient) {
				item := github.CreateTestSearchItem("microsoft/types", "src/api.ts", "TypeScript interface")
				item.TextMatches = []github.TextMatch{
					{
						Fragment: github.StringPtr("interface ApiResponse {\n  status: number;\n  data: any;\n}"),
						Property: github.StringPtr("content"),
					},
				}
				mock.SetSearchResults("interface", github.CreateTestSearchResults(1, item))
			},
			expectedOutput: "```typescript",
			description:    "Should detect TypeScript language and add syntax highlighting",
		},
		{
			name: "no text matches fallback",
			args: []string{"config"},
			mockSetup: func(mock *github.MockClient) {
				// Create item without text matches
				item := github.CreateTestSearchItem("example/repo", "config.json", "config file")
				item.TextMatches = []github.TextMatch{} // Empty - no fragments
				mock.SetSearchResults("config", github.CreateTestSearchResults(1, item))
			},
			expectedOutput: "config.json",
			description:    "Should still show file info even without text matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetSearchFlags()
			defer resetSearchFlags()

			// Set up mock client
			mockClient := github.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			originalClient := searchClient
			searchClient = mockClient
			defer func() { searchClient = originalClient }()

			// Capture output
			output := captureOutput(func() error {
				return runSearch(searchCmd, tt.args)
			})

			// Verify no error
			assert.NoError(t, output.err, tt.description)

			// Verify expected content is in output
			if tt.expectedOutput != "" {
				assert.Contains(t, output.stdout, tt.expectedOutput,
					"Output should contain expected text fragment (%s)", tt.description)
			}

			// Verify API was called
			assert.Equal(t, 1, mockClient.GetCallCount("SearchCode"),
				"Should make exactly one search API call (%s)", tt.description)
		})
	}
}

// TestGhxCompatibility tests compatibility with original ghx command patterns
func TestGhxCompatibility(t *testing.T) {
	// These tests ensure we maintain compatibility with ghx usage patterns
	compatibilityTests := []struct {
		name        string
		ghxCommand  string // What the ghx command would look like
		args        []string
		setup       func()
		description string
	}{
		{
			name:        "typescript config search",
			ghxCommand:  "ghx --filename tsconfig.json strict --limit 2",
			args:        []string{"strict"},
			setup:       func() { searchFilename = "tsconfig.json"; searchLimit = 2 },
			description: "Search for 'strict' in tsconfig.json files",
		},
		{
			name:        "react components with hooks",
			ghxCommand:  "ghx --language typescript --extension tsx useState",
			args:        []string{"useState"},
			setup:       func() { searchLanguage = "typescript"; searchExtension = "tsx" },
			description: "Find React components using useState hook",
		},
		{
			name:        "repository-specific search",
			ghxCommand:  "ghx --repo facebook/react hooks --limit 1",
			args:        []string{"hooks"},
			setup:       func() { searchRepo = []string{"facebook/react"}; searchLimit = 1 },
			description: "Search for hooks in facebook/react repository",
		},
	}

	for _, tt := range compatibilityTests {
		t.Run(tt.name, func(t *testing.T) {
			resetSearchFlags()
			defer resetSearchFlags()

			// Set up mock
			mockClient := github.NewMockClient()
			mockClient.SetSearchResults("test-query", github.CreateTestSearchResults(1,
				github.CreateTestSearchItem("test/repo", "test.ts", "test content"),
			))

			originalClient := searchClient
			searchClient = mockClient
			defer func() { searchClient = originalClient }()

			// Apply setup
			if tt.setup != nil {
				tt.setup()
			}

			// Execute search
			err := runSearch(searchCmd, tt.args)
			assert.NoError(t, err, "ghx compatibility test failed for: %s", tt.ghxCommand)

			// Verify API was called
			assert.Equal(t, 1, mockClient.GetCallCount("SearchCode"), "Should execute search for ghx command: %s", tt.ghxCommand)
		})
	}
}
