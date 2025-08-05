# gh-search Development Guide

**Project**: Convert ghx to gh-search (GitHub CLI Extension)  
**Goal**: Professional-grade GitHub code search with configuration discovery workflows  
**Target Quality**: Match gh-comment's 85% test coverage and production standards  

## 🎯 **Project Overview**

gh-search transforms GitHub code search from a basic utility into a powerful configuration discovery and pattern analysis tool. It leverages gh-comment's proven architecture patterns while solving a different developer workflow problem.

### **Core Value Proposition**
- **Configuration Discovery**: Find working configs across millions of GitHub repos
- **Pattern Analysis**: Identify common patterns and best practices  
- **AI Integration**: Feed concrete examples to AI for better assistance
- **Developer Workflow**: Seamless integration into development process

## 🏗️ **Architecture Migration Strategy**

### **From TypeScript Monolith → Go CLI Extension**

**Current ghx Architecture (TypeScript):**
```
src/index.ts         # 400+ lines, all functionality
test/index.test.ts   # Basic integration tests
```

**Target gh-search Architecture (Go, following gh-comment patterns):**
```
main.go              # Entry point
cmd/
├── root.go          # Root command & global flags
├── search.go        # Main search command  
├── patterns.go      # Pattern analysis command
├── saved.go         # Saved searches command
├── compare.go       # Configuration comparison
├── template.go      # Template generation
└── *_test.go        # Comprehensive test files
internal/
├── github/          # GitHub API client (interface-based)
├── search/          # Search logic and filters
├── analysis/        # Pattern analysis engine
├── config/          # Configuration management
└── output/          # Output formatting (markdown, JSON, etc.)
```

## 📋 **Development Phases**

### **Phase 1: Core Migration (2-3 weeks)**
1. **Architecture Setup**: Apply gh-comment patterns to Go CLI
2. **GitHub Integration**: Implement search API with intelligent error handling
3. **Command Structure**: Multiple focused commands vs single monolithic command
4. **Testing Foundation**: 80%+ test coverage from day one

### **Phase 2: Enhanced Features (4-6 weeks)**  
1. **Smart Ranking**: Quality-based result sorting
2. **Pattern Analysis**: Common configuration pattern detection
3. **Saved Searches**: Organization and caching system
4. **Comparison Mode**: Side-by-side configuration analysis

### **Phase 3: AI Integration (6-8 weeks)**
1. **Template Generation**: Create configs from patterns
2. **AI Workflow Integration**: Direct pipeline to AI assistance  
3. **Advanced Analytics**: Deep pattern analysis
4. **Enterprise Features**: Team sharing, organization insights

## 🎨 **Design Patterns from gh-comment**

### **1. Dependency Injection Architecture**
```go
// github/client.go - Interface for testability
type GitHubAPI interface {
    SearchCode(query string, opts SearchOptions) (*SearchResults, error)
    GetFileContent(owner, repo, path, ref string) ([]byte, error)
}

// Real implementation
type RealClient struct {
    restClient *github.Client
}

// Mock for testing  
type MockClient struct {
    searchResults map[string]*SearchResults
    errors        map[string]error
}
```

### **2. Command Structure Pattern**
```go
// cmd/search.go - Main command with dependency injection
var searchClient github.GitHubAPI

var searchCmd = &cobra.Command{
    Use:   "search <query> [flags]",
    Short: "Search GitHub code with intelligent filtering",
    RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
    if searchClient == nil {
        client, err := createGitHubClient()
        if err != nil {
            return fmt.Errorf("failed to create client: %w", err)
        }
        searchClient = client
    }
    // Implementation...
}
```

### **3. Intelligent Error Handling**
```go
// Following gh-comment's formatActionableError pattern
func handleSearchError(err error, query string) error {
    errMsg := err.Error()
    
    if strings.Contains(errMsg, "rate limit") {
        return fmt.Errorf("GitHub search rate limit exceeded: %w\n\n💡 Solutions:\n  • Wait %s for rate limit reset\n  • Use more specific search terms to reduce results\n  • Add --repo filter to search specific repositories", err, getRateLimitResetTime())
    }
    
    if strings.Contains(errMsg, "invalid query") {
        return fmt.Errorf("invalid search query: %w\n\n💡 GitHub Search Tips:\n  • Use quotes for exact phrases: \"exact match\"\n  • Combine terms: config AND typescript\n  • Use qualifiers: language:go filename:main.go", err)
    }
    
    return fmt.Errorf("search failed: %w", err)
}
```

### **4. Table-Driven Testing Pattern**
```go
// cmd/search_test.go
func TestSearchCommand(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        mockResults    *github.SearchResults
        mockError      error
        expectedOutput string
        wantErr        bool
    }{
        {
            name: "successful search with results",
            args: []string{"tsconfig.json", "--language", "json"},
            mockResults: &github.SearchResults{
                Items: []github.SearchItem{
                    {Path: "tsconfig.json", Repository: github.Repository{FullName: "facebook/react"}},
                },
            },
            expectedOutput: "Found 1 results",
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### **5. Configuration Management**
```go
// internal/config/config.go - Following gh-comment's pattern
type Config struct {
    Defaults struct {
        Language   string `yaml:"language" json:"language"`
        MaxResults int    `yaml:"max_results" json:"max_results"`
        Context    int    `yaml:"context_lines" json:"context_lines"`
    } `yaml:"defaults" json:"defaults"`
    
    Saved map[string]SavedSearch `yaml:"saved" json:"saved"`
    
    Output struct {
        Format string `yaml:"format" json:"format"`
        Editor string `yaml:"editor" json:"editor"`
    } `yaml:"output" json:"output"`
}

type SavedSearch struct {
    Query    string            `yaml:"query" json:"query"`
    Filters  map[string]string `yaml:"filters" json:"filters"`
    Created  time.Time         `yaml:"created" json:"created"`
}
```

## 🚀 **Command Design**

### **Root Command (gh search)**
```bash
gh search <query> [flags]                    # Main search interface
gh search --saved <name>                     # Run saved search
gh search --help                             # Comprehensive help
```

### **Subcommands Following gh-comment Pattern**
```bash
gh search patterns <query>                   # Analyze common patterns
gh search save <name> <query> [flags]       # Save search for reuse  
gh search list                               # List saved searches
gh search compare <file1> <file2>           # Compare configurations
gh search template <query> --output <file>  # Generate template from patterns
```

### **Global Flags (Consistent with gh-comment)**
```bash
--repo <owner/repo>          # Filter by repository
--language <lang>            # Filter by language  
--filename <name>            # Filter by filename
--extension <ext>            # Filter by extension
--limit <n>                  # Max results (default: 50)
--context <n>                # Context lines (default: 20)
--format <format>            # Output format: default|json|markdown
--save <name>                # Save search
--dry-run                    # Show what would be searched
--verbose                    # Detailed output
--no-color                   # Disable colors
```

## 📊 **Quality Standards**

### **Following gh-comment's Excellence:**
- **Test Coverage**: 80%+ minimum (aim for 85%+)
- **Error Handling**: Intelligent analysis with actionable suggestions
- **Documentation**: Comprehensive help text with working examples
- **Architecture**: Clean separation of concerns with interfaces
- **User Experience**: Professional CLI with consistent patterns

### **Testing Strategy**
```go
// Unit Tests - Every public function
func TestSearchFilters(t *testing.T) { /* ... */ }
func TestPatternAnalysis(t *testing.T) { /* ... */ }
func TestResultRanking(t *testing.T) { /* ... */ }

// Integration Tests - End-to-end workflows  
func TestSearchWorkflow(t *testing.T) { /* ... */ }
func TestSavedSearches(t *testing.T) { /* ... */ }

// Benchmark Tests - Performance tracking
func BenchmarkLargeSearch(b *testing.B) { /* ... */ }
func BenchmarkPatternAnalysis(b *testing.B) { /* ... */ }
```

### **Documentation Standards**
- **README.md**: User-focused with practical examples
- **ADVANCED_USAGE.md**: Power user features and automation
- **CONTRIBUTING.md**: Development setup and patterns
- **Help Text**: Every command has comprehensive examples

## 🔧 **Implementation Priorities**

### **Week 1-2: Foundation**
1. ✅ Set up Go module structure
2. ✅ Implement GitHub API client with interfaces
3. ✅ Basic search command with core filters
4. ✅ Test framework and initial coverage
5. ✅ Error handling system

### **Week 3-4: Core Features** 
1. ✅ Advanced filtering (repo, language, filename, etc.)
2. ✅ Output formatting (markdown, JSON)
3. ✅ Configuration management
4. ✅ Saved searches functionality
5. ✅ Integration testing

### **Week 5-6: Enhancement**
1. ✅ Pattern analysis engine
2. ✅ Result ranking algorithms  
3. ✅ Comparison functionality
4. ✅ Template generation
5. ✅ Performance optimization

### **Week 7-8: Polish**
1. ✅ Comprehensive testing (85%+ coverage)
2. ✅ Documentation completion
3. ✅ CI/CD pipeline setup
4. ✅ gh extension packaging
5. ✅ Beta testing and refinement

## 📚 **Key Learnings from gh-comment**

### **What Made gh-comment Successful:**
1. **Clear Value Proposition**: Solved real developer pain points
2. **Professional Architecture**: Interface-based, testable, maintainable
3. **Excellent Error Handling**: Users get helpful guidance, not cryptic errors
4. **Comprehensive Testing**: 85% coverage ensures reliability
5. **Great Documentation**: Working examples that users can copy/paste
6. **Consistent UX**: Predictable flag patterns and behavior

### **Apply These Patterns:**
1. **Dependency Injection**: Every command uses interfaces for testing
2. **Table-Driven Tests**: Comprehensive scenario coverage
3. **Intelligent Errors**: Analyze failures and provide solutions
4. **Professional Help**: Rich examples and clear explanations
5. **Mock-First Testing**: Isolated, reliable test suites

## 🎯 **Success Metrics**

### **Technical Excellence**
- **Test Coverage**: 85%+ (match gh-comment)
- **Performance**: <2s for typical searches  
- **Reliability**: Intelligent error handling for all failure modes
- **Maintainability**: Clean, modular, well-documented code

### **User Experience**
- **Discovery**: Users find working configurations 10x faster
- **Integration**: Seamless workflow with existing gh CLI
- **Learning**: Developers understand patterns across codebases
- **Productivity**: Configuration setup time reduced by 80%

---

**Next Steps**: Review this guide, then proceed to the specific implementation documents for architecture, testing strategy, and command specifications.