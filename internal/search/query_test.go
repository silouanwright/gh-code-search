package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQueryBuilder(t *testing.T) {
	terms := []string{"useState", "React hooks"}
	qb := NewQueryBuilder(terms)

	assert.NotNil(t, qb)
	assert.Equal(t, terms, qb.terms)
	assert.NotNil(t, qb.filters)
	assert.NotNil(t, qb.qualifiers)
	assert.NotNil(t, qb.constraints)
}

func TestQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name     string
		setupQB  func() *QueryBuilder
		expected string
	}{
		{
			name: "simple terms only",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{"useState", "React"})
			},
			expected: "useState React",
		},
		{
			name: "terms with language filter",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{"useState"}).WithLanguage("typescript")
			},
			expected: "useState language:typescript",
		},
		{
			name: "comprehensive query",
			setupQB: func() *QueryBuilder {
				qb := NewQueryBuilder([]string{"hooks"})
				qb.WithLanguage("typescript")
				qb.WithFilename("*.tsx")
				qb.WithRepositories([]string{"facebook/react", "vercel/next.js"})
				qb.WithMinStars(100)
				return qb
			},
			expected: "hooks language:typescript filename:*.tsx stars:>=100 repo:facebook/react repo:vercel/next.js",
		},
		{
			name: "no terms with filters only",
			setupQB: func() *QueryBuilder {
				qb := NewQueryBuilder([]string{})
				qb.WithLanguage("go")
				qb.WithFilename("main.go")
				return qb
			},
			expected: "language:go filename:main.go",
		},
		{
			name: "multiple repositories and owners",
			setupQB: func() *QueryBuilder {
				qb := NewQueryBuilder([]string{"config"})
				qb.WithRepositories([]string{"repo1", "repo2"})
				qb.WithOwners([]string{"microsoft", "google"})
				return qb
			},
			expected: "config repo:repo1 repo:repo2 user:microsoft user:google",
		},
		{
			name: "with size and path filters",
			setupQB: func() *QueryBuilder {
				qb := NewQueryBuilder([]string{"large file"})
				qb.WithSize(">1000")
				qb.WithPath("src/")
				qb.WithExtension("js")
				return qb
			},
			expected: "large file extension:js path:src/ size:>1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setupQB()
			result := qb.Build()

			// Split both strings into parts for comparison (order may vary)
			expectedParts := strings.Fields(tt.expected)
			resultParts := strings.Fields(result)

			assert.ElementsMatch(t, expectedParts, resultParts, 
				"Query parts should match regardless of order. Got: %s, Expected: %s", result, tt.expected)
		})
	}
}

func TestQueryBuilder_ChainableMethods(t *testing.T) {
	// Test that all methods return the QueryBuilder for chaining
	qb := NewQueryBuilder([]string{"test"}).
		WithLanguage("go").
		WithFilename("main.go").
		WithExtension("go").
		WithPath("cmd/").
		WithSize(">100").
		WithMinStars(50).
		WithMaxAge("2023-01-01").
		WithFork("false").
		WithRepositories([]string{"owner/repo"}).
		WithOwners([]string{"owner"}).
		WithMatch([]string{"file"})

	assert.NotNil(t, qb)
	
	query := qb.Build()
	assert.Contains(t, query, "test")
	assert.Contains(t, query, "language:go")
	assert.Contains(t, query, "filename:main.go")
	assert.Contains(t, query, "extension:go")
	assert.Contains(t, query, "path:cmd/")
	assert.Contains(t, query, "size:>100")
	assert.Contains(t, query, "stars:>=50")
	assert.Contains(t, query, "pushed:>2023-01-01")
	assert.Contains(t, query, "fork:false")
	assert.Contains(t, query, "repo:owner/repo")
	assert.Contains(t, query, "user:owner")
	assert.Contains(t, query, "in:file")
}

func TestNewQueryBuilderFromFilters(t *testing.T) {
	filters := SearchFilters{
		Language:   "typescript",
		Filename:   "*.tsx",
		Extension:  "tsx", 
		Repository: []string{"facebook/react"},
		Path:       "src/",
		Owner:      []string{"facebook"},
		Size:       ">1000",
		MinStars:   100,
		MaxAge:     "2023-01-01",
		Fork:       "false",
		Match:      []string{"file", "path"},
	}

	qb := NewQueryBuilderFromFilters([]string{"useState"}, filters)
	query := qb.Build()

	assert.Contains(t, query, "useState")
	assert.Contains(t, query, "language:typescript")
	assert.Contains(t, query, "filename:*.tsx")
	assert.Contains(t, query, "extension:tsx")
	assert.Contains(t, query, "repo:facebook/react")
	assert.Contains(t, query, "path:src/")
	assert.Contains(t, query, "user:facebook")
	assert.Contains(t, query, "size:>1000")
	assert.Contains(t, query, "stars:>=100")
	assert.Contains(t, query, "pushed:>2023-01-01")
	assert.Contains(t, query, "fork:false")
	assert.Contains(t, query, "in:file")
	assert.Contains(t, query, "in:path")
}

func TestQueryBuilder_GetFilters(t *testing.T) {
	qb := NewQueryBuilder([]string{"test"}).
		WithLanguage("go").
		WithFilename("main.go").
		WithExtension("go").
		WithRepositories([]string{"owner/repo1", "owner/repo2"}).
		WithOwners([]string{"microsoft", "google"}).
		WithPath("cmd/").
		WithSize(">100").
		WithMinStars(50).
		WithMaxAge("2023-01-01").
		WithFork("true").
		WithMatch([]string{"file"})

	filters := qb.GetFilters()

	assert.Equal(t, "go", filters.Language)
	assert.Equal(t, "main.go", filters.Filename)
	assert.Equal(t, "go", filters.Extension)
	assert.Equal(t, []string{"owner/repo1", "owner/repo2"}, filters.Repository)
	assert.Equal(t, []string{"microsoft", "google"}, filters.Owner)
	assert.Equal(t, "cmd/", filters.Path)
	assert.Equal(t, ">100", filters.Size)
	assert.Equal(t, 50, filters.MinStars)
	assert.Equal(t, "2023-01-01", filters.MaxAge)
	assert.Equal(t, "true", filters.Fork)
	assert.Equal(t, []string{"file"}, filters.Match)
}

func TestQueryBuilder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setupQB func() *QueryBuilder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid query with terms",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{"useState"})
			},
			wantErr: false,
		},
		{
			name: "valid query with filters only",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithLanguage("go")
			},
			wantErr: false,
		},
		{
			name: "empty query",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{})
			},
			wantErr: true,
			errMsg:  "query must contain search terms or filters",
		},
		{
			name: "invalid language",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithLanguage("invalidlang")
			},
			wantErr: true,
			errMsg:  "invalid language",
		},
		{
			name: "valid language",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithLanguage("javascript")
			},
			wantErr: false,
		},
		{
			name: "invalid size format",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithSize("invalid").WithLanguage("go")
			},
			wantErr: true,
			errMsg:  "invalid size format",
		},
		{
			name: "valid size formats",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithSize(">1000").WithLanguage("go")
			},
			wantErr: false,
		},
		{
			name: "invalid fork value",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithFork("invalid").WithLanguage("go")
			},
			wantErr: true,
			errMsg:  "invalid fork value",
		},
		{
			name: "valid fork values",
			setupQB: func() *QueryBuilder {
				return NewQueryBuilder([]string{}).WithFork("only").WithLanguage("go")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := tt.setupQB()
			err := qb.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationHelpers(t *testing.T) {
	t.Run("isValidLanguage", func(t *testing.T) {
		validLanguages := []string{
			"javascript", "typescript", "python", "go", "java", "c", "cpp",
			"html", "css", "json", "yaml", "shell", "dockerfile",
		}

		for _, lang := range validLanguages {
			assert.True(t, isValidLanguage(lang), "Language %s should be valid", lang)
			// Test case insensitivity
			assert.True(t, isValidLanguage(strings.ToUpper(lang)), "Language %s (uppercase) should be valid", lang)
		}

		invalidLanguages := []string{"invalidlang", "xyz", "notreal"}
		for _, lang := range invalidLanguages {
			assert.False(t, isValidLanguage(lang), "Language %s should be invalid", lang)
		}
	})

	t.Run("isValidSizeFormat", func(t *testing.T) {
		validSizes := []string{
			">1000", "<500", ">=100", "<=200", "=42", "100..200", "1000",
		}

		for _, size := range validSizes {
			assert.True(t, isValidSizeFormat(size), "Size format %s should be valid", size)
		}

		invalidSizes := []string{"abc", ">abc", "invalid", ">>100"}
		for _, size := range invalidSizes {
			assert.False(t, isValidSizeFormat(size), "Size format %s should be invalid", size)
		}
	})

	t.Run("isValidForkValue", func(t *testing.T) {
		validValues := []string{"true", "false", "only", "TRUE", "FALSE", "ONLY"}
		for _, value := range validValues {
			assert.True(t, isValidForkValue(value), "Fork value %s should be valid", value)
		}

		invalidValues := []string{"yes", "no", "1", "0", "invalid"}
		for _, value := range invalidValues {
			assert.False(t, isValidForkValue(value), "Fork value %s should be invalid", value)
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("BuildConfigQuery", func(t *testing.T) {
		tests := []struct {
			name       string
			configType string
			filters    SearchFilters
			contains   []string
		}{
			{
				name:       "typescript config",
				configType: "typescript",
				filters:    SearchFilters{Repository: []string{"facebook/react"}},
				contains:   []string{"tsconfig.json", "language:json", "repo:facebook/react"},
			},
			{
				name:       "docker config",
				configType: "dockerfile",
				filters:    SearchFilters{MinStars: 100},
				contains:   []string{"dockerfile", "stars:>=100"},
			},
			{
				name:       "package.json",
				configType: "package",
				filters:    SearchFilters{Path: "frontend/"},
				contains:   []string{"package.json", "language:json", "path:frontend/"},
			},
			{
				name:       "custom config type",
				configType: "custom.conf",
				filters:    SearchFilters{Language: "shell"},
				contains:   []string{"custom.conf", "language:shell"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				query := BuildConfigQuery(tt.configType, tt.filters)

				for _, expected := range tt.contains {
					assert.Contains(t, query, expected, "Query should contain %s", expected)
				}
			})
		}
	})

	t.Run("BuildPatternQuery", func(t *testing.T) {
		query := BuildPatternQuery("useState", "typescript", SearchFilters{
			Repository: []string{"facebook/react"},
			MinStars:   50,
		})

		assert.Contains(t, query, "useState")
		assert.Contains(t, query, "language:typescript")
		assert.Contains(t, query, "repo:facebook/react")
		assert.Contains(t, query, "stars:>=50")
	})

	t.Run("BuildRepoQuery", func(t *testing.T) {
		query := BuildRepoQuery(
			[]string{"config", "setup"},
			[]string{"microsoft/vscode", "github/hub"},
			SearchFilters{Language: "json"},
		)

		assert.Contains(t, query, "config setup")
		assert.Contains(t, query, "repo:microsoft/vscode")
		assert.Contains(t, query, "repo:github/hub")
		assert.Contains(t, query, "language:json")
	})
}

func TestSearchFilters(t *testing.T) {
	filters := SearchFilters{
		Language:   "typescript",
		Filename:   "config.ts",
		Extension:  "ts",
		Repository: []string{"org/repo1", "org/repo2"},
		Path:       "src/",
		Owner:      []string{"microsoft", "google"},
		Size:       ">1000",
		MinStars:   100,
		MaxAge:     "2023-01-01",
		Fork:       "false",
		Match:      []string{"file", "path"},
	}

	// Test that all fields are properly accessible
	assert.Equal(t, "typescript", filters.Language)
	assert.Equal(t, "config.ts", filters.Filename)
	assert.Equal(t, "ts", filters.Extension)
	assert.Len(t, filters.Repository, 2)
	assert.Contains(t, filters.Repository, "org/repo1")
	assert.Equal(t, "src/", filters.Path)
	assert.Len(t, filters.Owner, 2)
	assert.Contains(t, filters.Owner, "microsoft")
	assert.Equal(t, ">1000", filters.Size)
	assert.Equal(t, 100, filters.MinStars)
	assert.Equal(t, "2023-01-01", filters.MaxAge)
	assert.Equal(t, "false", filters.Fork)
	assert.Len(t, filters.Match, 2)
	assert.Contains(t, filters.Match, "file")
}

func TestEmptyFilters(t *testing.T) {
	qb := NewQueryBuilder([]string{"test"})

	// Test that empty values are ignored
	qb.WithLanguage("")
	qb.WithFilename("")
	qb.WithRepositories([]string{})
	qb.WithOwners(nil)
	qb.WithMinStars(0)

	query := qb.Build()
	assert.Equal(t, "test", query) // Should only contain the search terms
}

func TestComplexQuery(t *testing.T) {
	// Test a realistic complex query that might be used in practice
	qb := NewQueryBuilder([]string{"React", "TypeScript", "components"}).
		WithLanguage("typescript").
		WithExtension("tsx").
		WithRepositories([]string{"facebook/react", "microsoft/fluentui", "ant-design/ant-design"}).
		WithPath("src/components/").
		WithMinStars(1000).
		WithFork("false").
		WithMatch([]string{"file"})

	query := qb.Build()

	// Verify all parts are present
	assert.Contains(t, query, "React TypeScript components")
	assert.Contains(t, query, "language:typescript")
	assert.Contains(t, query, "extension:tsx")
	assert.Contains(t, query, "repo:facebook/react")
	assert.Contains(t, query, "repo:microsoft/fluentui")
	assert.Contains(t, query, "repo:ant-design/ant-design")
	assert.Contains(t, query, "path:src/components/")
	assert.Contains(t, query, "stars:>=1000")
	assert.Contains(t, query, "fork:false")
	assert.Contains(t, query, "in:file")

	// Verify the query is valid
	err := qb.Validate()
	assert.NoError(t, err)
}