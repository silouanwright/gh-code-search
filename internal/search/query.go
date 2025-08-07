package search

import (
	"fmt"
	"strings"
)

// QueryBuilder constructs GitHub search queries with filters
type QueryBuilder struct {
	terms       []string
	filters     map[string][]string
	qualifiers  map[string]string
	constraints map[string]string
}

// SearchFilters represents common search filter options
type SearchFilters struct {
	Language    string   `json:"language,omitempty" yaml:"language,omitempty"`
	Filename    string   `json:"filename,omitempty" yaml:"filename,omitempty"`
	Extension   string   `json:"extension,omitempty" yaml:"extension,omitempty"`
	Repository  []string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Path        string   `json:"path,omitempty" yaml:"path,omitempty"`
	Owner       []string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Size        string   `json:"size,omitempty" yaml:"size,omitempty"`
	MinStars    int      `json:"min_stars,omitempty" yaml:"min_stars,omitempty"`
	MaxAge      string   `json:"max_age,omitempty" yaml:"max_age,omitempty"`
	Fork        string   `json:"fork,omitempty" yaml:"fork,omitempty"`
	Match       []string `json:"match,omitempty" yaml:"match,omitempty"`
}

// NewQueryBuilder creates a new QueryBuilder with search terms
func NewQueryBuilder(terms []string) *QueryBuilder {
	return &QueryBuilder{
		terms:       terms,
		filters:     make(map[string][]string),
		qualifiers:  make(map[string]string),
		constraints: make(map[string]string),
	}
}

// NewQueryBuilderFromFilters creates a QueryBuilder from SearchFilters struct
func NewQueryBuilderFromFilters(terms []string, filters SearchFilters) *QueryBuilder {
	qb := NewQueryBuilder(terms)

	// Apply single-value filters
	if filters.Language != "" {
		qb.WithLanguage(filters.Language)
	}
	if filters.Filename != "" {
		qb.WithFilename(filters.Filename)
	}
	if filters.Extension != "" {
		qb.WithExtension(filters.Extension)
	}
	if filters.Path != "" {
		qb.WithPath(filters.Path)
	}
	if filters.Size != "" {
		qb.WithSize(filters.Size)
	}
	if filters.MinStars > 0 {
		qb.WithMinStars(filters.MinStars)
	}
	if filters.MaxAge != "" {
		qb.WithMaxAge(filters.MaxAge)
	}
	if filters.Fork != "" {
		qb.WithFork(filters.Fork)
	}

	// Apply multi-value filters
	if len(filters.Repository) > 0 {
		qb.WithRepositories(filters.Repository)
	}
	if len(filters.Owner) > 0 {
		qb.WithOwners(filters.Owner)
	}
	if len(filters.Match) > 0 {
		qb.WithMatch(filters.Match)
	}

	return qb
}

// WithLanguage adds a language filter
func (qb *QueryBuilder) WithLanguage(lang string) *QueryBuilder {
	if lang != "" {
		qb.qualifiers["language"] = lang
	}
	return qb
}

// WithFilename adds a filename filter
func (qb *QueryBuilder) WithFilename(filename string) *QueryBuilder {
	if filename != "" {
		qb.qualifiers["filename"] = filename
	}
	return qb
}

// WithExtension adds a file extension filter
func (qb *QueryBuilder) WithExtension(ext string) *QueryBuilder {
	if ext != "" {
		qb.qualifiers["extension"] = ext
	}
	return qb
}

// WithPath adds a file path filter
func (qb *QueryBuilder) WithPath(path string) *QueryBuilder {
	if path != "" {
		qb.qualifiers["path"] = path
	}
	return qb
}

// WithSize adds a file size filter
func (qb *QueryBuilder) WithSize(size string) *QueryBuilder {
	if size != "" {
		qb.qualifiers["size"] = size
	}
	return qb
}

// WithMinStars adds a minimum stars filter
func (qb *QueryBuilder) WithMinStars(stars int) *QueryBuilder {
	if stars > 0 {
		qb.constraints["stars"] = fmt.Sprintf(">=%d", stars)
	}
	return qb
}

// WithMaxAge adds a maximum file age filter
func (qb *QueryBuilder) WithMaxAge(maxAge string) *QueryBuilder {
	if maxAge != "" {
		qb.constraints["pushed"] = fmt.Sprintf(">%s", maxAge)
	}
	return qb
}

// WithFork adds a fork filter
func (qb *QueryBuilder) WithFork(fork string) *QueryBuilder {
	if fork != "" {
		qb.qualifiers["fork"] = fork
	}
	return qb
}

// WithRepositories adds repository filters
func (qb *QueryBuilder) WithRepositories(repos []string) *QueryBuilder {
	if len(repos) > 0 {
		qb.filters["repo"] = repos
	}
	return qb
}

// WithOwners adds owner/user filters
func (qb *QueryBuilder) WithOwners(owners []string) *QueryBuilder {
	if len(owners) > 0 {
		qb.filters["user"] = owners
	}
	return qb
}

// WithMatch adds match location filters (file or path)
func (qb *QueryBuilder) WithMatch(matches []string) *QueryBuilder {
	if len(matches) > 0 {
		qb.filters["in"] = matches
	}
	return qb
}

// Build constructs the final GitHub search query string
func (qb *QueryBuilder) Build() string {
	var parts []string

	// Add main search terms
	if len(qb.terms) > 0 {
		// Join terms and handle phrases
		termString := strings.Join(qb.terms, " ")
		parts = append(parts, termString)
	}

	// Add single-value qualifiers in consistent order (language:go, filename:config.json)
	qualifierOrder := []string{"language", "filename", "extension", "path", "size", "fork"}
	for _, key := range qualifierOrder {
		if value, exists := qb.qualifiers[key]; exists {
			parts = append(parts, fmt.Sprintf("%s:%s", key, value))
		}
	}

	// Add constraints in consistent order (stars:>=100, pushed:>2023-01-01)
	constraintOrder := []string{"stars", "pushed"}
	for _, key := range constraintOrder {
		if value, exists := qb.constraints[key]; exists {
			parts = append(parts, fmt.Sprintf("%s:%s", key, value))
		}
	}

	// Add multi-value filters in consistent order (repo:owner/name repo:other/repo)
	filterOrder := []string{"repo", "user", "in"}
	for _, key := range filterOrder {
		if values, exists := qb.filters[key]; exists {
			for _, value := range values {
				parts = append(parts, fmt.Sprintf("%s:%s", key, value))
			}
		}
	}

	return strings.Join(parts, " ")
}

// GetFilters returns the current filters as a SearchFilters struct
func (qb *QueryBuilder) GetFilters() SearchFilters {
	filters := SearchFilters{
		Language:   qb.qualifiers["language"],
		Filename:   qb.qualifiers["filename"],
		Extension:  qb.qualifiers["extension"],
		Path:       qb.qualifiers["path"],
		Size:       qb.qualifiers["size"],
		Fork:       qb.qualifiers["fork"],
		Repository: qb.filters["repo"],
		Owner:      qb.filters["user"],
		Match:      qb.filters["in"],
	}

	// Parse stars constraint
	if starsConstraint, exists := qb.constraints["stars"]; exists {
		// Simple parsing for >=n format
		if strings.HasPrefix(starsConstraint, ">=") {
			var stars int
			_, _ = fmt.Sscanf(starsConstraint, ">=%d", &stars)
			filters.MinStars = stars
		}
	}

	// Parse age constraint
	if ageConstraint, exists := qb.constraints["pushed"]; exists {
		// Simple parsing for >date format
		if strings.HasPrefix(ageConstraint, ">") {
			filters.MaxAge = strings.TrimPrefix(ageConstraint, ">")
		}
	}

	return filters
}

// Validate checks if the query builder has valid configuration
func (qb *QueryBuilder) Validate() error {
	// Check if we have at least search terms or qualifiers
	if len(qb.terms) == 0 && len(qb.qualifiers) == 0 && len(qb.filters) == 0 {
		return fmt.Errorf("query must contain search terms or filters")
	}

	// Validate language values
	if lang, exists := qb.qualifiers["language"]; exists {
		if !isValidLanguage(lang) {
			return fmt.Errorf("invalid language: %s", lang)
		}
	}

	// Validate size format
	if size, exists := qb.qualifiers["size"]; exists {
		if !isValidSizeFormat(size) {
			return fmt.Errorf("invalid size format: %s", size)
		}
	}

	// Validate fork values
	if fork, exists := qb.qualifiers["fork"]; exists {
		if !isValidForkValue(fork) {
			return fmt.Errorf("invalid fork value: %s (must be true, false, or only)", fork)
		}
	}

	return nil
}

// Helper functions for validation

// isValidLanguage checks if the language is recognized by GitHub
func isValidLanguage(lang string) bool {
	// Common languages supported by GitHub search
	validLanguages := map[string]bool{
		"javascript": true, "typescript": true, "python": true, "go": true,
		"java": true, "c": true, "cpp": true, "csharp": true, "php": true,
		"ruby": true, "rust": true, "swift": true, "kotlin": true, "scala": true,
		"html": true, "css": true, "scss": true, "sass": true, "less": true,
		"json": true, "yaml": true, "xml": true, "markdown": true, "sql": true,
		"shell": true, "bash": true, "dockerfile": true, "makefile": true,
	}
	return validLanguages[strings.ToLower(lang)]
}

// isValidSizeFormat checks if the size format is valid
func isValidSizeFormat(size string) bool {
	// Check for comparison operators with numbers
	if strings.HasPrefix(size, ">=") {
		var num int
		_, err := fmt.Sscanf(size[2:], "%d", &num)
		return err == nil
	}
	if strings.HasPrefix(size, "<=") {
		var num int
		_, err := fmt.Sscanf(size[2:], "%d", &num)
		return err == nil
	}
	if strings.HasPrefix(size, ">") {
		var num int
		_, err := fmt.Sscanf(size[1:], "%d", &num)
		return err == nil
	}
	if strings.HasPrefix(size, "<") {
		var num int
		_, err := fmt.Sscanf(size[1:], "%d", &num)
		return err == nil
	}
	if strings.HasPrefix(size, "=") {
		var num int
		_, err := fmt.Sscanf(size[1:], "%d", &num)
		return err == nil
	}

	// Check for range format (e.g., "100..200")
	if strings.Contains(size, "..") {
		parts := strings.Split(size, "..")
		if len(parts) == 2 {
			var num1, num2 int
			_, err1 := fmt.Sscanf(parts[0], "%d", &num1)
			_, err2 := fmt.Sscanf(parts[1], "%d", &num2)
			return err1 == nil && err2 == nil
		}
		return false
	}

	// Check for plain numbers
	var num int
	_, err := fmt.Sscanf(size, "%d", &num)
	return err == nil
}

// isValidForkValue checks if the fork value is valid
func isValidForkValue(fork string) bool {
	validValues := map[string]bool{
		"true": true, "false": true, "only": true,
	}
	return validValues[strings.ToLower(fork)]
}

// Utility functions for common query patterns

// BuildConfigQuery creates a query for finding configuration files
func BuildConfigQuery(configType string, filters SearchFilters) string {
	var terms []string

	switch strings.ToLower(configType) {
	case "typescript", "tsconfig":
		terms = []string{"tsconfig.json"}
		filters.Language = "json"
	case "eslint":
		terms = []string{"eslint.config"}
		filters.Language = "javascript"
	case "docker", "dockerfile":
		terms = []string{"dockerfile"}
	case "package":
		terms = []string{"package.json"}
		filters.Language = "json"
	case "vite":
		terms = []string{"vite.config"}
		filters.Language = "javascript"
	case "webpack":
		terms = []string{"webpack.config"}
		filters.Language = "javascript"
	case "tailwind":
		terms = []string{"tailwind.config"}
		filters.Language = "javascript"
	default:
		terms = []string{configType}
	}

	return NewQueryBuilderFromFilters(terms, filters).Build()
}

// BuildPatternQuery creates a query for finding code patterns
func BuildPatternQuery(pattern string, language string, filters SearchFilters) string {
	terms := []string{pattern}
	if language != "" {
		filters.Language = language
	}

	return NewQueryBuilderFromFilters(terms, filters).Build()
}

// BuildRepoQuery creates a query for searching within specific repositories
func BuildRepoQuery(terms []string, repos []string, filters SearchFilters) string {
	filters.Repository = repos
	return NewQueryBuilderFromFilters(terms, filters).Build()
}
