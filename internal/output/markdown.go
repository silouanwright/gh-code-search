package output

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/silouanwright/gh-search/internal/github"
)

// MarkdownFormatter formats search results as markdown
type MarkdownFormatter struct {
	ShowLineNumbers bool
	ContextLines    int
	HighlightTerms  []string
	ShowRepository  bool
	ShowStars       bool
	ShowPatterns    bool
	MaxContentLines int
	ColorMode       string
}

// NewMarkdownFormatter creates a new markdown formatter with default settings
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{
		ShowLineNumbers: false,
		ContextLines:    20,
		HighlightTerms:  []string{},
		ShowRepository:  true,
		ShowStars:       true,
		ShowPatterns:    true,
		MaxContentLines: 50,
		ColorMode:       "auto",
	}
}

// Format formats search results as markdown
func (f *MarkdownFormatter) Format(results *github.SearchResults, query string) (string, error) {
	var buf strings.Builder
	
	// Header with search summary
	f.writeHeader(&buf, results, query)
	
	// Format each result
	for i, item := range results.Items {
		if i > 0 {
			buf.WriteString("\n---\n\n")
		}
		
		f.formatSearchItem(&buf, &item, i+1)
	}
	
	// Footer with additional info
	f.writeFooter(&buf, results)
	
	return buf.String(), nil
}

// writeHeader writes the markdown header with search summary
func (f *MarkdownFormatter) writeHeader(buf *strings.Builder, results *github.SearchResults, query string) {
	buf.WriteString("# GitHub Code Search Results\n\n")
	
	// Search query info
	buf.WriteString(fmt.Sprintf("**Query**: `%s`  \n", query))
	buf.WriteString(fmt.Sprintf("**Date**: %s  \n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Results summary
	totalCount := 0
	if results.Total != nil {
		totalCount = *results.Total
	}
	
	resultCount := len(results.Items)
	if totalCount > resultCount {
		buf.WriteString(fmt.Sprintf("**Results**: Showing %d of %d total results\n\n", resultCount, totalCount))
	} else {
		buf.WriteString(fmt.Sprintf("**Results**: Found %d results\n\n", totalCount))
	}
	
	// Results overview
	if resultCount > 0 {
		buf.WriteString("🔍 **Search Results:**\n\n")
	} else {
		buf.WriteString("❌ **No results found** for this query.\n\n")
		buf.WriteString("💡 **Suggestions:**\n")
		buf.WriteString("- Try broader search terms\n")
		buf.WriteString("- Check spelling and syntax\n")
		buf.WriteString("- Use different language or repository filters\n")
		buf.WriteString("- Search in popular repositories\n\n")
	}
}

// formatSearchItem formats a single search result item
func (f *MarkdownFormatter) formatSearchItem(buf *strings.Builder, item *github.SearchItem, index int) {
	// Repository header with emoji and stats
	f.formatRepositoryHeader(buf, item, index)
	
	// File information
	f.formatFileInfo(buf, item)
	
	// Code content with syntax highlighting
	f.formatCodeContent(buf, item)
	
	// File link and metadata
	f.formatFileFooter(buf, item)
}

// formatRepositoryHeader formats the repository header section
func (f *MarkdownFormatter) formatRepositoryHeader(buf *strings.Builder, item *github.SearchItem, index int) {
	repoName := getStringValue(item.Repository.FullName)
	repoURL := getStringValue(item.Repository.HTMLURL)
	
	if repoName == "" {
		buf.WriteString(fmt.Sprintf("## %d. Unknown Repository\n\n", index))
		return
	}
	
	// Repository name with link
	if repoURL != "" {
		buf.WriteString(fmt.Sprintf("## %d. 📁 [%s](%s)", index, repoName, repoURL))
	} else {
		buf.WriteString(fmt.Sprintf("## %d. 📁 %s", index, repoName))
	}
	
	// Repository statistics
	if f.ShowStars {
		stats := f.getRepositoryStats(item)
		if stats != "" {
			buf.WriteString(fmt.Sprintf(" %s", stats))
		}
	}
	
	buf.WriteString("\n\n")
	
	// Repository description
	if desc := getStringValue(item.Repository.Description); desc != "" {
		buf.WriteString(fmt.Sprintf("*%s*\n\n", desc))
	}
}

// formatFileInfo formats file path and metadata
func (f *MarkdownFormatter) formatFileInfo(buf *strings.Builder, item *github.SearchItem) {
	filePath := getStringValue(item.Path)
	if filePath == "" {
		return
	}
	
	// File icon based on extension
	icon := f.getFileIcon(filePath)
	
	// File path with icon
	buf.WriteString(fmt.Sprintf("**%s %s**\n\n", icon, filePath))
	
	// File metadata
	if f.ShowLineNumbers || f.ShowPatterns {
		metadata := f.getFileMetadata(item)
		if metadata != "" {
			buf.WriteString(fmt.Sprintf("*%s*\n\n", metadata))
		}
	}
}

// formatCodeContent formats the code content with syntax highlighting
func (f *MarkdownFormatter) formatCodeContent(buf *strings.Builder, item *github.SearchItem) {
	if len(item.TextMatches) == 0 {
		buf.WriteString("*No code preview available*\n\n")
		return
	}
	
	language := f.detectLanguage(getStringValue(item.Path))
	
	for i, match := range item.TextMatches {
		if i > 0 {
			buf.WriteString("\n")
		}
		
		f.formatTextMatch(buf, &match, language, i+1)
	}
}

// formatTextMatch formats a single text match with context
func (f *MarkdownFormatter) formatTextMatch(buf *strings.Builder, match *github.TextMatch, language string, matchIndex int) {
	fragment := getStringValue(match.Fragment)
	if fragment == "" {
		return
	}
	
	// Multiple matches in same file get numbered
	if matchIndex > 1 {
		buf.WriteString(fmt.Sprintf("**Match %d:**\n\n", matchIndex))
	}
	
	// Code block with syntax highlighting
	buf.WriteString(fmt.Sprintf("```%s\n", language))
	
	// Format the fragment with line numbers if requested
	if f.ShowLineNumbers {
		lines := strings.Split(fragment, "\n")
		for i, line := range lines {
			buf.WriteString(fmt.Sprintf("%3d: %s\n", i+1, line))
		}
	} else {
		// Apply content length limit
		if f.MaxContentLines > 0 {
			fragment = f.limitContentLines(fragment, f.MaxContentLines)
		}
		buf.WriteString(fragment)
		if !strings.HasSuffix(fragment, "\n") {
			buf.WriteString("\n")
		}
	}
	
	buf.WriteString("```\n\n")
	
	// Show match highlights if available
	f.formatMatchHighlights(buf, match)
}

// formatMatchHighlights shows specific match locations within fragments
func (f *MarkdownFormatter) formatMatchHighlights(buf *strings.Builder, match *github.TextMatch) {
	if len(match.Matches) == 0 {
		return
	}
	
	buf.WriteString("**Matches:**\n")
	for _, m := range match.Matches {
		if text := getStringValue(m.Text); text != "" {
			buf.WriteString(fmt.Sprintf("- `%s`", text))
			if len(m.Indices) >= 2 {
				buf.WriteString(fmt.Sprintf(" (positions %d-%d)", m.Indices[0], m.Indices[1]))
			}
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n")
}

// formatFileFooter formats links and additional file information
func (f *MarkdownFormatter) formatFileFooter(buf *strings.Builder, item *github.SearchItem) {
	fileURL := getStringValue(item.HTMLURL)
	if fileURL == "" {
		return
	}
	
	// File link
	buf.WriteString(fmt.Sprintf("🔗 [View on GitHub](%s)\n", fileURL))
	
	// Additional links
	if gitURL := getStringValue(item.GitURL); gitURL != "" {
		buf.WriteString(fmt.Sprintf("📄 [Raw file](%s)\n", strings.Replace(gitURL, "git://", "https://raw.githubusercontent.com/", 1)))
	}
	
	buf.WriteString("\n")
}

// writeFooter writes the markdown footer with summary information
func (f *MarkdownFormatter) writeFooter(buf *strings.Builder, results *github.SearchResults) {
	if len(results.Items) == 0 {
		return
	}
	
	buf.WriteString("---\n\n")
	
	// Summary statistics
	f.writeStatistics(buf, results)
	
	// Usage tips
	buf.WriteString("💡 **Tips:**\n")
	buf.WriteString("- Use `--save <name>` to save this search for reuse\n")
	buf.WriteString("- Add `--format json` for machine-readable output\n")
	buf.WriteString("- Try `--min-stars N` to filter by repository popularity\n")
	buf.WriteString("- Use `--context N` to adjust code context lines\n\n")
	
	// Timestamp
	buf.WriteString(fmt.Sprintf("*Generated by gh-search on %s*\n", time.Now().Format("2006-01-02 15:04:05")))
}

// writeStatistics writes summary statistics about the results
func (f *MarkdownFormatter) writeStatistics(buf *strings.Builder, results *github.SearchResults) {
	buf.WriteString("📊 **Summary:**\n")
	
	// Language distribution
	languages := f.analyzeLanguages(results)
	if len(languages) > 0 {
		buf.WriteString("- **Languages**: ")
		langParts := make([]string, 0, len(languages))
		for lang, count := range languages {
			langParts = append(langParts, fmt.Sprintf("%s (%d)", lang, count))
		}
		buf.WriteString(strings.Join(langParts, ", "))
		buf.WriteString("\n")
	}
	
	// Repository distribution
	repos := f.analyzeRepositories(results)
	if len(repos) > 1 {
		buf.WriteString(fmt.Sprintf("- **Repositories**: %d unique repositories\n", len(repos)))
	}
	
	// Top repositories by stars
	if f.ShowStars {
		topRepos := f.getTopRepositories(results, 3)
		if len(topRepos) > 0 {
			buf.WriteString("- **Popular repositories**: ")
			repoParts := make([]string, len(topRepos))
			for i, repo := range topRepos {
				repoParts[i] = fmt.Sprintf("%s (%d⭐)", repo.name, repo.stars)
			}
			buf.WriteString(strings.Join(repoParts, ", "))
			buf.WriteString("\n")
		}
	}
	
	buf.WriteString("\n")
}

// Helper functions

// getStringValue safely gets string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getIntValue safely gets int value from pointer
func getIntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// getRepositoryStats formats repository statistics
func (f *MarkdownFormatter) getRepositoryStats(item *github.SearchItem) string {
	var stats []string
	
	if stars := getIntValue(item.Repository.StargazersCount); stars > 0 {
		stats = append(stats, fmt.Sprintf("⭐ %s", formatNumber(stars)))
	}
	
	if forks := getIntValue(item.Repository.ForksCount); forks > 0 {
		stats = append(stats, fmt.Sprintf("🔀 %s", formatNumber(forks)))
	}
	
	if lang := getStringValue(item.Repository.Language); lang != "" {
		stats = append(stats, fmt.Sprintf("📝 %s", lang))
	}
	
	if len(stats) == 0 {
		return ""
	}
	
	return "(" + strings.Join(stats, " • ") + ")"
}

// getFileIcon returns an appropriate emoji for the file type
func (f *MarkdownFormatter) getFileIcon(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := strings.ToLower(filepath.Base(filePath))
	
	// Special filenames
	switch filename {
	case "dockerfile", "dockerfile.dev", "dockerfile.prod":
		return "🐳"
	case "makefile", "makefile.am":
		return "🔨"
	case "readme.md", "readme.txt":
		return "📖"
	case "license", "licence":
		return "⚖️"
	case "package.json":
		return "📦"
	case "tsconfig.json":
		return "⚙️"
	case ".gitignore":
		return "🚫"
	}
	
	// Extensions
	switch ext {
	case ".go":
		return "🐹"
	case ".js", ".mjs":
		return "💛"
	case ".ts", ".tsx":
		return "🔷"
	case ".py":
		return "🐍"
	case ".rs":
		return "🦀"
	case ".java":
		return "☕"
	case ".cpp", ".cc", ".cxx":
		return "⚡"
	case ".c":
		return "🔧"
	case ".cs":
		return "🔷"
	case ".php":
		return "🐘"
	case ".rb":
		return "💎"
	case ".swift":
		return "🐦"
	case ".kt":
		return "🎯"
	case ".scala":
		return "🌶️"
	case ".html", ".htm":
		return "🌐"
	case ".css", ".scss", ".sass":
		return "🎨"
	case ".json":
		return "📋"
	case ".yaml", ".yml":
		return "📄"
	case ".xml":
		return "📰"
	case ".md":
		return "📝"
	case ".sh", ".bash":
		return "💻"
	case ".sql":
		return "🗃️"
	case ".dockerfile":
		return "🐳"
	default:
		return "📄"
	}
}

// detectLanguage detects programming language from file path
func (f *MarkdownFormatter) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	filename := strings.ToLower(filepath.Base(path))
	
	// Special filenames
	switch filename {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	}
	
	// Extensions
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss":
		return "scss"
	case ".sass":
		return "sass"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "bash"
	case ".sql":
		return "sql"
	case ".dockerfile":
		return "dockerfile"
	default:
		return ""
	}
}

// getFileMetadata returns metadata about the file
func (f *MarkdownFormatter) getFileMetadata(item *github.SearchItem) string {
	var metadata []string
	
	// File size info from matches
	if len(item.TextMatches) > 0 {
		metadata = append(metadata, fmt.Sprintf("%d match(es)", len(item.TextMatches)))
	}
	
	// Language from repository
	if lang := getStringValue(item.Repository.Language); lang != "" {
		metadata = append(metadata, fmt.Sprintf("Language: %s", lang))
	}
	
	return strings.Join(metadata, " • ")
}

// limitContentLines limits content to specified number of lines
func (f *MarkdownFormatter) limitContentLines(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	
	limited := strings.Join(lines[:maxLines], "\n")
	remaining := len(lines) - maxLines
	limited += fmt.Sprintf("\n... (%d more lines)", remaining)
	return limited
}

// formatNumber formats large numbers with appropriate suffixes
func formatNumber(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// Analysis helper functions

// analyzeLanguages returns language distribution in results
func (f *MarkdownFormatter) analyzeLanguages(results *github.SearchResults) map[string]int {
	languages := make(map[string]int)
	
	for _, item := range results.Items {
		if path := getStringValue(item.Path); path != "" {
			lang := f.detectLanguage(path)
			if lang != "" {
				languages[lang]++
			}
		}
	}
	
	return languages
}

// analyzeRepositories returns unique repository names
func (f *MarkdownFormatter) analyzeRepositories(results *github.SearchResults) map[string]bool {
	repos := make(map[string]bool)
	
	for _, item := range results.Items {
		if repoName := getStringValue(item.Repository.FullName); repoName != "" {
			repos[repoName] = true
		}
	}
	
	return repos
}

type repoInfo struct {
	name  string
	stars int
}

// getTopRepositories returns top repositories by star count
func (f *MarkdownFormatter) getTopRepositories(results *github.SearchResults, limit int) []repoInfo {
	repoMap := make(map[string]int)
	
	for _, item := range results.Items {
		if repoName := getStringValue(item.Repository.FullName); repoName != "" {
			stars := getIntValue(item.Repository.StargazersCount)
			if existing, exists := repoMap[repoName]; !exists || stars > existing {
				repoMap[repoName] = stars
			}
		}
	}
	
	// Convert to slice and sort
	repos := make([]repoInfo, 0, len(repoMap))
	for name, stars := range repoMap {
		repos = append(repos, repoInfo{name: name, stars: stars})
	}
	
	// Simple sort by stars (descending)
	for i := 0; i < len(repos); i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[j].stars > repos[i].stars {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
	
	// Limit results
	if len(repos) > limit {
		repos = repos[:limit]
	}
	
	return repos
}