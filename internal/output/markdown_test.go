package output

import (
	"testing"

	"github.com/silouanwright/gh-code-search/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMarkdownFormatter(t *testing.T) {
	formatter := NewMarkdownFormatter()

	assert.NotNil(t, formatter)
	assert.False(t, formatter.ShowLineNumbers)
	assert.Equal(t, 20, formatter.ContextLines)
	assert.Equal(t, []string{}, formatter.HighlightTerms)
	assert.True(t, formatter.ShowRepository)
	assert.True(t, formatter.ShowStars)
	assert.True(t, formatter.ShowPatterns)
	assert.Equal(t, 50, formatter.MaxContentLines)
	assert.Equal(t, "auto", formatter.ColorMode)
}

func TestMarkdownFormatter_Format(t *testing.T) {
	formatter := NewMarkdownFormatter()

	// Test with empty results
	emptyResults := &github.SearchResults{
		Total:             github.IntPtr(0),
		IncompleteResults: github.BoolPtr(false),
		Items:             []github.SearchItem{},
	}

	output, err := formatter.Format(emptyResults, "test query")
	require.NoError(t, err)
	assert.Contains(t, output, "# GitHub Code Search Results")
	assert.Contains(t, output, "Query**: `test query`")
	assert.Contains(t, output, "No results found")
	assert.Contains(t, output, "Suggestions:")

	// Test with real results
	results := createTestSearchResults()
	output, err = formatter.Format(results, "useState")
	require.NoError(t, err)
	
	assert.Contains(t, output, "# GitHub Code Search Results")
	assert.Contains(t, output, "Query**: `useState`")
	assert.Contains(t, output, "Found 2 results")
	assert.Contains(t, output, "📁 [facebook/react]")
	assert.Contains(t, output, "ReactHooks.ts")
	assert.Contains(t, output, "```typescript")
	assert.Contains(t, output, "function useState")
	assert.Contains(t, output, "View on GitHub")
}

func TestMarkdownFormatter_FormatWithCustomSettings(t *testing.T) {
	formatter := NewMarkdownFormatter()
	formatter.ShowLineNumbers = true
	formatter.ShowStars = false
	formatter.MaxContentLines = 3

	results := createTestSearchResults()
	output, err := formatter.Format(results, "test")
	require.NoError(t, err)

	// Should have line numbers
	assert.Contains(t, output, "  1: ")
	assert.Contains(t, output, "  2: ")

	// Should not show stars since ShowStars is false
	assert.NotContains(t, output, "⭐")
}

func TestGetStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    github.StringPtr(""),
			expected: "",
		},
		{
			name:     "non-empty string",
			input:    github.StringPtr("test"),
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "zero value",
			input:    github.IntPtr(0),
			expected: 0,
		},
		{
			name:     "positive value",
			input:    github.IntPtr(42),
			expected: 42,
		},
		{
			name:     "negative value", 
			input:    github.IntPtr(-10),
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRepositoryStats(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		name     string
		item     *github.SearchItem
		expected string
	}{
		{
			name: "repository with all stats",
			item: &github.SearchItem{
				Repository: github.Repository{
					StargazersCount: github.IntPtr(1500),
					ForksCount:      github.IntPtr(200),
					Language:        github.StringPtr("TypeScript"),
				},
			},
			expected: "(⭐ 1.5k • 🔀 200 • 📝 TypeScript)",
		},
		{
			name: "repository with stars only",
			item: &github.SearchItem{
				Repository: github.Repository{
					StargazersCount: github.IntPtr(100),
					ForksCount:      github.IntPtr(0),
					Language:        nil,
				},
			},
			expected: "(⭐ 100)",
		},
		{
			name: "repository with no stats",
			item: &github.SearchItem{
				Repository: github.Repository{
					StargazersCount: github.IntPtr(0),
					ForksCount:      github.IntPtr(0),
					Language:        nil,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.getRepositoryStats(tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileIcon(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		path     string
		expected string
	}{
		// Special filenames
		{"Dockerfile", "🐳"},
		{"dockerfile", "🐳"},
		{"Makefile", "🔨"},
		{"README.md", "📖"},
		{"package.json", "📦"},
		{"tsconfig.json", "⚙️"},
		{".gitignore", "🚫"},

		// Extensions
		{"main.go", "🐹"},
		{"script.js", "💛"},
		{"component.ts", "🔷"},
		{"component.tsx", "🔷"},
		{"app.py", "🐍"},
		{"lib.rs", "🦀"},
		{"Main.java", "☕"},
		{"program.cpp", "⚡"},
		{"code.c", "🔧"},
		{"app.cs", "🔷"},
		{"index.php", "🐘"},
		{"gem.rb", "💎"},
		{"app.swift", "🐦"},
		{"main.kt", "🎯"},
		{"App.scala", "🌶️"},
		{"index.html", "🌐"},
		{"style.css", "🎨"},
		{"config.json", "📋"},
		{"docker-compose.yml", "📄"},
		{"data.xml", "📰"},
		{"notes.md", "📝"},
		{"setup.sh", "💻"},
		{"query.sql", "🗃️"},
		
		// Default
		{"unknown.xyz", "📄"},
		{"", "📄"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := formatter.getFileIcon(tt.path)
			assert.Equal(t, tt.expected, result, "File path: %s", tt.path)
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		path     string
		expected string
	}{
		// Special filenames
		{"Dockerfile", "dockerfile"},
		{"dockerfile", "dockerfile"},
		{"Makefile", "makefile"},

		// Extensions
		{"main.go", "go"},
		{"script.js", "javascript"},
		{"module.mjs", "javascript"},
		{"component.ts", "typescript"},
		{"component.tsx", "tsx"},
		{"app.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"program.cpp", "cpp"},
		{"code.c", "c"},
		{"app.cs", "csharp"},
		{"index.php", "php"},
		{"gem.rb", "ruby"},
		{"app.swift", "swift"},
		{"main.kt", "kotlin"},
		{"App.scala", "scala"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"style.scss", "scss"},
		{"style.sass", "sass"},
		{"config.json", "json"},
		{"docker-compose.yml", "yaml"},
		{"data.xml", "xml"},
		{"README.md", "markdown"},
		{"setup.sh", "bash"},
		{"script.bash", "bash"},
		{"query.sql", "sql"},
		{"custom.dockerfile", "dockerfile"},

		// Unknown/default
		{"unknown.xyz", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := formatter.detectLanguage(tt.path)
			assert.Equal(t, tt.expected, result, "File path: %s", tt.path)
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{12345, "12.3k"},
		{999999, "1000.0k"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
		{12345678, "12.3M"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.input)), func(t *testing.T) {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLimitContentLines(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		name     string
		content  string
		maxLines int
		expected string
	}{
		{
			name:     "content within limit",
			content:  "line 1\nline 2\nline 3",
			maxLines: 5,
			expected: "line 1\nline 2\nline 3",
		},
		{
			name:     "content exceeds limit",
			content:  "line 1\nline 2\nline 3\nline 4\nline 5",
			maxLines: 3,
			expected: "line 1\nline 2\nline 3\n... (2 more lines)",
		},
		{
			name:     "single line",
			content:  "single line",
			maxLines: 1,
			expected: "single line",
		},
		{
			name:     "empty content",
			content:  "",
			maxLines: 5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.limitContentLines(tt.content, tt.maxLines)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzeLanguages(t *testing.T) {
	formatter := NewMarkdownFormatter()
	results := createTestSearchResults()

	languages := formatter.analyzeLanguages(results)

	assert.Contains(t, languages, "typescript")
	assert.Contains(t, languages, "json")
	assert.Equal(t, 1, languages["typescript"]) // ReactHooks.ts
	assert.Equal(t, 1, languages["json"])       // package.json
}

func TestAnalyzeRepositories(t *testing.T) {
	formatter := NewMarkdownFormatter()
	results := createTestSearchResults()

	repos := formatter.analyzeRepositories(results)

	assert.Contains(t, repos, "facebook/react")
	assert.Contains(t, repos, "vercel/next.js")
	assert.Equal(t, 2, len(repos))
}

func TestGetTopRepositories(t *testing.T) {
	formatter := NewMarkdownFormatter()
	results := createTestSearchResults()

	topRepos := formatter.getTopRepositories(results, 2)

	assert.Len(t, topRepos, 2)
	// Should be sorted by stars (descending)
	assert.Equal(t, "facebook/react", topRepos[0].name)
	assert.Equal(t, 50000, topRepos[0].stars)
	assert.Equal(t, "vercel/next.js", topRepos[1].name)
	assert.Equal(t, 30000, topRepos[1].stars)
}

func TestGetFileMetadata(t *testing.T) {
	formatter := NewMarkdownFormatter()
	
	item := &github.SearchItem{
		TextMatches: []github.TextMatch{
			{Fragment: github.StringPtr("match1")},
			{Fragment: github.StringPtr("match2")},
		},
		Repository: github.Repository{
			Language: github.StringPtr("TypeScript"),
		},
	}

	metadata := formatter.getFileMetadata(item)
	assert.Contains(t, metadata, "2 match(es)")
	assert.Contains(t, metadata, "Language: TypeScript")
}

// Helper function to create test search results
func createTestSearchResults() *github.SearchResults {
	return &github.SearchResults{
		Total:             github.IntPtr(2),
		IncompleteResults: github.BoolPtr(false),
		Items: []github.SearchItem{
			{
				Name:    github.StringPtr("ReactHooks.ts"),
				Path:    github.StringPtr("packages/react/src/ReactHooks.ts"),
				SHA:     github.StringPtr("abc123"),
				HTMLURL: github.StringPtr("https://github.com/facebook/react/blob/main/packages/react/src/ReactHooks.ts"),
				Repository: github.Repository{
					ID:              (*int64)(nil), // Use nil instead of Int64Ptr
					Name:            github.StringPtr("react"),
					FullName:        github.StringPtr("facebook/react"),
					HTMLURL:         github.StringPtr("https://github.com/facebook/react"),
					Description:     github.StringPtr("A declarative, efficient, and flexible JavaScript library for building user interfaces."),
					StargazersCount: github.IntPtr(50000),
					ForksCount:      github.IntPtr(10000),
					Language:        github.StringPtr("TypeScript"),
					Owner: &github.User{
						Login:   github.StringPtr("facebook"),
						HTMLURL: github.StringPtr("https://github.com/facebook"),
					},
				},
				TextMatches: []github.TextMatch{
					{
						Fragment: github.StringPtr("function useState<S>(initialState: S | (() => S)): [S, Dispatch<SetStateAction<S>>] {\n  return resolveDispatcher().useState(initialState);\n}"),
						Matches: []github.Match{
							{
								Text:    github.StringPtr("useState"),
								Indices: []int{9, 17},
							},
						},
					},
				},
			},
			{
				Name:    github.StringPtr("package.json"),
				Path:    github.StringPtr("packages/next/package.json"),
				SHA:     github.StringPtr("def456"),
				HTMLURL: github.StringPtr("https://github.com/vercel/next.js/blob/main/packages/next/package.json"),
				Repository: github.Repository{
					ID:              (*int64)(nil), // Use nil instead of Int64Ptr
					Name:            github.StringPtr("next.js"),
					FullName:        github.StringPtr("vercel/next.js"),
					HTMLURL:         github.StringPtr("https://github.com/vercel/next.js"),
					Description:     github.StringPtr("The React Framework"),
					StargazersCount: github.IntPtr(30000),
					ForksCount:      github.IntPtr(5000),
					Language:        github.StringPtr("JavaScript"),
					Owner: &github.User{
						Login:   github.StringPtr("vercel"),
						HTMLURL: github.StringPtr("https://github.com/vercel"),
					},
				},
				TextMatches: []github.TextMatch{
					{
						Fragment: github.StringPtr("{\n  \"name\": \"next\",\n  \"version\": \"13.0.0\"\n}"),
						Matches: []github.Match{
							{
								Text:    github.StringPtr("next"),
								Indices: []int{11, 15},
							},
						},
					},
				},
			},
		},
	}
}