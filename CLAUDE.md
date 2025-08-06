# CLAUDE.md - Development Context

## Project Overview
gh-search is a professional GitHub CLI extension for code search and configuration discovery, migrating from the ghx TypeScript monolith to a production-grade Go implementation.

## Key Reference Materials
- **Gold Standard**: ~/repos/gh-comment - Use as the reference for architecture patterns, testing strategies, and CLI UX excellence
- **Documentation**: Complete specification in DEVELOPMENT_GUIDE.md, ARCHITECTURE.md, MIGRATION_GUIDE.md, etc.
- **Migration Source**: ~/repos/ghx - Original TypeScript implementation to migrate

## Architecture Principles (from gh-comment)
1. **Interface-based design** with dependency injection for testability
2. **85%+ test coverage** with table-driven tests
3. **Intelligent error handling** with actionable user guidance
4. **Professional CLI UX** with comprehensive help and examples
5. **Modular structure** separating concerns across packages

## Development Standards
- **Commits**: Use conventional commit format (feat:, fix:, docs:, test:, refactor:)
- **Testing**: Mock-first testing with comprehensive scenario coverage
- **Documentation**: Working examples users can copy/paste
- **Quality Gates**: Match gh-comment's production standards

## Current Status - Production Ready ✅
All critical architecture tasks complete. Codebase is production-ready with excellent test coverage, clean architecture, and professional UX.

## Implementation Highlights
- **Interface-based architecture** with dependency injection for testability
- **Comprehensive error handling** with actionable user guidance
- **Full ghx feature parity** plus enhancements (min-stars, multiple formats, etc.)
- **Professional documentation** following gh-comment's clean style
- **Automated testing & releases** via GitHub Actions
- **Respectful ghx acknowledgment** without competitive language




## NEXT PHASE - BATCH OPERATIONS SYSTEM (Primary Focus)

### **Batch Operations - The Definitive Next Feature**
Batch operations will provide massive value for configuration discovery and comparative analysis workflows.


## NEXT: BATCH OPERATIONS SYSTEM 🚀
Primary focus: Multi-repo configuration analysis and comparison workflows.

## DEVELOPMENT REFERENCE GUIDE 📚

### **Architecture Study Commands**
```bash
# Study gh-comment's excellence (GOLD STANDARD)
ls ~/repos/gh-comment/cmd/                    # Command structure
ls ~/repos/gh-comment/internal/               # Package organization
find ~/repos/gh-comment -name "*_test.go" | head -5  # Test patterns
grep -r "formatActionableError" ~/repos/gh-comment/   # Error handling

# Current codebase analysis
tree -I 'vendor|.git' -L 3                   # Project structure
go test ./... -cover                          # Test coverage
go build && ./gh-search --version            # Verify build
```

### **Testing & Verification Commands**  
```bash
# Verify current functionality
./gh-search search "test" --format json | head -10
./gh-search search "test" --format markdown | head -10
./gh-search search "test" --format compact | head -3
go test ./... -cover  # Check test coverage
```


### **Code Quality Verification**
```bash
# Check for issues mentioned in code review  
gofmt -d .                                    # Code formatting
go vet ./...                                  # Static analysis
golangci-lint run                             # Comprehensive linting
go mod tidy                                   # Dependency cleanup

# Architecture verification
grep -r "interface{}" --include="*.go" .     # Avoid empty interfaces
grep -r "panic\|os.Exit" --include="*.go" .  # Avoid panics in libraries
find . -name "*.go" -exec wc -l {} + | sort -n | tail -10  # Find large files
```

### **Performance & Security Checks**
```bash
# Rate limiting and API usage
./gh-search search "test" --limit 100 --verbose  # Check API calls
./gh-search rate-limit                           # Check remaining quota

# Security validation  
grep -r "token\|secret\|key" --include="*.go" . | grep -v test  # Token handling
find . -name "*.go" -exec grep -l "os.Create\|WriteFile" {} \;  # File permissions
```

## IMPLEMENTATION ROADMAP 📋

### **PRIMARY FEATURE: Batch Operations System**
1. **Phase 1**: YAML-driven batch searches (2-3 hours)
2. **Phase 2**: Multi-repository comparative search (2-3 hours)  
3. **Phase 3**: Advanced batch features and comparison outputs (3-4 hours)

### **ENHANCEMENTS**
- Add rate limiting between paginated calls
- Performance benchmarking and optimization
- Pattern analysis and template extraction

## DEVELOPER CONTEXT 👋

**Current State**: Production-ready codebase with excellent architecture, 85%+ test coverage, and professional UX. All critical foundation work is complete.

**Ready For**: Advanced feature development, specifically batch operations system for multi-repo configuration analysis.

**Architecture**: Interface-based design with dependency injection, comprehensive error handling, and modular structure following gh-comment patterns.

---

## 🚀 **MAJOR ENHANCEMENT: BATCH OPERATIONS SYSTEM**

Based on comprehensive analysis of gh-comment's excellent batch functionality, gh-search is **perfectly positioned** to implement powerful batch operations that would provide massive value for configuration discovery and comparative analysis workflows.

### **Why Batch Operations Are Critical for gh-search**

**Current Pain Points**:
- Users need to run multiple searches manually for tech stack analysis
- No way to compare configurations across different repositories/organizations
- Repetitive searches for similar patterns across tech ecosystems
- No aggregation of results for pattern analysis

**Batch Operations Would Enable**:
```bash
# Tech stack comparative analysis
gh search batch react-typescript-configs.yaml --format comparison

# Multi-organization configuration discovery  
gh search "vite.config" --repos microsoft/*,facebook/*,vercel/* --aggregate

# Pattern analysis across ecosystems
gh search "webpack OR vite" --repos facebook/*,vercel/* --compare

# Pattern analysis across ecosystems
gh search batch ecosystem-analysis.yaml --output analysis/ --compare
```

### **IMPLEMENTATION FOUNDATION - ALREADY EXCELLENT** ✅

**gh-search's Existing Architecture is Perfect for Batch Operations**:
- ✅ **Interface-based design** with dependency injection (easy batch client management)
- ✅ **Comprehensive configuration system** (`internal/config/config.go`)  
- ✅ **Multiple output formats** (JSON, Markdown, Compact)
- ✅ **Professional error handling** with actionable guidance
- ✅ **Rate limiting awareness** built into client

### **DETAILED IMPLEMENTATION PLAN** 📋

#### **Phase 1: YAML-Driven Batch Searches (2-3 hours)**

**Create `cmd/batch.go`** following gh-comment's exact patterns:
```go
// Based on gh-comment's batch.go structure
type BatchConfig struct {
    Name        string              `yaml:"name,omitempty"`
    Description string              `yaml:"description,omitempty"`
    Output      BatchOutputConfig   `yaml:"output,omitempty"`
    Searches    []BatchSearchConfig `yaml:"searches"`
}

type BatchSearchConfig struct {
    Name        string               `yaml:"name"`
    Query       string               `yaml:"query"`
    Filters     search.SearchFilters `yaml:"filters"`  // Use existing search package
    MaxResults  int                  `yaml:"max_results,omitempty"`
    Tags        []string             `yaml:"tags,omitempty"`
}

type BatchOutputConfig struct {
    Format      string `yaml:"format"`        // "combined", "separate", "comparison"
    Directory   string `yaml:"directory"`     // Output directory
    Compare     bool   `yaml:"compare"`       // Enable comparison mode
    Aggregate   bool   `yaml:"aggregate"`     // Combine results
}
```

**Example batch configuration** (`examples/tech-stack-analysis.yaml`):
```yaml
name: "React + TypeScript Configuration Analysis"
description: "Compare configuration patterns across different tech stacks"
output:
  format: "comparison"
  directory: "analysis-results"
  compare: true

searches:
  - name: "vite-react-ts"
    query: "vite.config"
    filters:
      language: "typescript"
      filename: "vite.config.ts"
      min_stars: 100
    max_results: 50
    tags: ["vite", "react", "typescript"]
    
  - name: "webpack-react-ts"
    query: "webpack.config"
    filters:
      language: "javascript"
      filename: "webpack.config.js"
      min_stars: 100
    max_results: 50
    tags: ["webpack", "react", "typescript"]
```

**📍 Reference Implementation**: Study `~/repos/gh-comment/cmd/batch.go` for:
- YAML parsing patterns
- Validation strategies  
- Error handling approaches
- Dry-run implementation
- Test structure in `batch_test.go`

#### **Phase 2: Multi-Repository Comparative Search (2-3 hours)**

**Command Structure**:
```bash
# Multi-repo search with aggregation
gh search "docker-compose.yml" --repos microsoft/vscode,facebook/react,vercel/next.js --aggregate

# Organization-wide configuration discovery
gh search "tsconfig.json" --orgs microsoft,google,facebook --min-stars 500 --compare

# Ecosystem analysis
gh search batch ecosystem-configs.yaml --format comparison --output results/
```

**Implementation in `cmd/search.go`**:
```go
// Add to existing search flags
var (
    batchRepos    []string  // --repos flag for multiple repositories
    batchOrgs     []string  // --orgs flag for multiple organizations  
    aggregateMode bool      // --aggregate flag
    compareMode   bool      // --compare flag
)

// Extend executeSearch to handle batch scenarios
func executeBatchSearch(ctx context.Context, query string, repos []string) (*BatchResults, error) {
    var allResults []*github.SearchResults
    
    for _, repo := range repos {
        repoQuery := fmt.Sprintf("%s repo:%s", query, repo)
        results, err := searchClient.SearchCode(ctx, repoQuery, &opts)
        if err != nil {
            return nil, handleSearchError(err, repoQuery)
        }
        allResults = append(allResults, results)
    }
    
    return aggregateResults(allResults), nil
}
```

#### **Phase 3: Advanced Batch Features (3-4 hours)**

**Comparison Output Format**:
```markdown
# Configuration Analysis: Vite vs Webpack (Generated by gh-search batch)

## Summary
- **Vite Configs Found**: 150 across 45 repositories
- **Webpack Configs Found**: 200 across 60 repositories  
- **Common Patterns**: 12 identified
- **Key Differences**: TypeScript integration, dev server config

## Pattern Analysis
### Vite Configurations
| Repository | Stars | Config Pattern | TypeScript |
|------------|-------|----------------|------------|
| vitejs/vite | 50k | ESM, plugins | ✅ |
| ...

### Webpack Configurations  
| Repository | Stars | Config Pattern | TypeScript |
|------------|-------|----------------|------------|
| webpack/webpack | 60k | CommonJS, loaders | ❌ |
| ...

## Recommendations
Based on analysis of 350 configurations:
1. **TypeScript**: Vite configs show 85% TypeScript adoption vs 45% for Webpack
2. **Plugin Patterns**: Vite uses simpler plugin syntax
3. **Performance**: Vite configs average 20% fewer lines
```

### **KEY ARCHITECTURAL ADVANTAGES** 🏆

**gh-search's Existing Foundation Provides**:
1. **Zero Breaking Changes**: Batch operations extend existing architecture
2. **Reuse Everything**: Existing search logic, output formats, error handling
3. **Dependency Injection**: Easy to test batch operations with mock clients
4. **Configuration System**: YAML-driven batch configurations
5. **Professional UX**: Same high-quality CLI experience

### **IMPLEMENTATION TIMELINE** ⏱️

**Phase 1 - Basic YAML Batch (2-3 hours)**:
- Create `cmd/batch.go` following gh-comment patterns
- YAML parsing with validation
- Dry-run support  
- Basic sequential execution

**Phase 2 - Multi-Repo Search (2-3 hours)**:  
- Add `--repos` and `--orgs` flags to existing search command
- Implement `executeBatchSearch` function
- Add aggregation and comparison logic
- Enhanced output formatting

**Phase 3 - Advanced Features (3-4 hours)**:
- Pattern analysis and extraction
- Template generation from common patterns  
- Rich comparison outputs
- Performance optimizations

### **SUCCESS SCENARIOS** 🎯

**Configuration Discovery Workflow**:
```bash
# 1. Analyze React ecosystem configurations
gh search batch react-ecosystem.yaml --output analysis/

# 2. Compare Next.js vs Remix patterns  
gh search "app router" --repos vercel/next.js,remix-run/remix --compare

# 3. Generate configuration templates
gh search batch docker-patterns.yaml --extract-templates

# 4. Organization-wide audit
gh search "security config" --orgs myorg --min-stars 10 --aggregate
```

**Expected User Value**:
- **10x faster** configuration discovery workflows
- **Systematic comparison** across tech stacks
- **Pattern identification** for best practices
- **Template generation** for new projects
- **Audit capabilities** for organizations

### **COMPETITIVE ADVANTAGE** 🚀

This would make gh-search the **only CLI tool** that provides:
- Batch configuration discovery across repositories
- Systematic pattern analysis and comparison  
- YAML-driven batch search workflows
- Professional-grade aggregation and reporting

Following gh-comment's proven batch patterns ensures this will be:
- **Reliable**: Tested patterns from production tool
- **Intuitive**: Familiar UX for gh-comment users  
- **Maintainable**: Clean architecture and comprehensive tests
- **Extensible**: Foundation for advanced features

**This enhancement would position gh-search as the definitive tool for configuration discovery and analysis in the GitHub ecosystem.**