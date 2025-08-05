# gh-search Command Specification

**Based on**: gh-comment's command patterns and UX excellence  
**Goal**: Intuitive, powerful GitHub code search with professional workflows  
**Style**: Consistent with GitHub CLI conventions and gh-comment patterns  

## 🎯 **Command Philosophy**

### **Core Principles (From gh-comment)**
1. **Predictable Patterns**: Consistent flag naming and behavior across commands
2. **Professional Help**: Rich examples that users can copy/paste and use immediately
3. **Intelligent Defaults**: Sensible defaults that work for common use cases
4. **Error Guidance**: When things fail, provide actionable solutions
5. **Workflow Integration**: Commands compose naturally together

### **Enhanced for Configuration Discovery**
- **Pattern Recognition**: Identify common configuration patterns across results
- **Quality Ranking**: Surface high-quality examples first (stars, activity, recency)
- **Context Preservation**: Always show enough context to understand usage
- **Workflow Optimization**: Designed for the "find working config → adapt → implement" cycle

## 🏗️ **Command Structure**

### **Root Command: `gh search`**
```bash
gh search <query> [flags]                    # Main search interface
```

**Philosophy**: Single entry point with powerful filtering, following `gh pr`, `gh issue` patterns

### **Subcommands for Advanced Features**
```bash
gh search patterns <query> [flags]          # Analyze configuration patterns
gh search saved <command> [args]            # Manage saved searches  
gh search template <query> [flags]          # Generate templates from patterns
gh search compare <file1> <file2>           # Compare configurations side-by-side
```

## 📋 **Detailed Command Specifications**

### **1. Main Search Command: `gh search`**

```bash
Usage: gh search <query> [flags]

Description:
  Search GitHub's codebase to discover working examples and configurations.
  
  Perfect for finding real-world usage patterns, configuration examples,
  and best practices across millions of repositories.

Examples:
  # Configuration discovery workflows
  gh search "tsconfig.json" --language json --limit 10
  gh search "vite.config" --language javascript --context 30
  gh search "dockerfile" --filename dockerfile --repo "**/react"
  
  # Pattern research with AI integration  
  gh search "eslint.config.js" --language javascript --save eslint-research
  gh search "next.config.js" --repo vercel/next.js --context 50
  
  # Advanced filtering for quality results
  gh search "tailwind.config" --min-stars 1000 --language javascript
  gh search "package.json" --path "examples/" --limit 20

Arguments:
  query                 Search terms (required)

Flags:
  # Core Filtering
  --language, -l string         Programming language filter
  --filename, -f string         Exact filename match
  --extension, -e string        File extension filter  
  --repo, -r strings            Repository filter (supports wildcards)
  --path, -p string             File path filter
  --owner, -o strings           Repository owner filter
  
  # Quality & Ranking
  --min-stars int               Minimum repository stars (default: 0)
  --max-age string              Maximum file age (e.g., "6m", "1y")
  --sort string                 Sort by: relevance, stars, updated, created
  
  # Output Control
  --limit int                   Maximum results (default: 50, max: 1000)
  --context int                 Context lines around matches (default: 20)
  --format string               Output format: default, json, markdown, compact
  --no-content                  Show only metadata, skip file contents
  
  # Workflow Integration
  --save string                 Save search with given name
  --open                        Open results in editor after search
  --pipe                        Output to stdout (for piping to other tools)
  
  # Global Flags (consistent with gh-comment)
  --dry-run                     Show query without executing
  --verbose                     Detailed output and timing
  --no-color                    Disable colored output
  --config string               Config file path

Exit Codes:
  0    Success (results found)
  1    Error (authentication, network, invalid query)  
  2    No results (empty result set)
```

#### **Search Query Intelligence**
```bash
# Smart query building (like gh-comment's intelligent parsing)
gh search "config typescript"              # → "config typescript"
gh search config --language typescript     # → "config language:typescript"  
gh search --filename "*.config.js"         # → "filename:*.config.js"
gh search react --repo facebook/react      # → "react repo:facebook/react"

# Advanced GitHub search syntax support
gh search "exact phrase match"             # → "exact phrase match"
gh search "config AND typescript"          # → "config AND typescript"  
gh search "config NOT test"                # → "config NOT test"
gh search "extension:js OR extension:ts"   # → "extension:js OR extension:ts"
```

### **2. Pattern Analysis: `gh search patterns`**

```bash
Usage: gh search patterns <query> [flags]

Description:
  Analyze common configuration patterns across search results.
  
  Identifies frequently used patterns, property combinations, and best
  practices from real-world configurations.

Examples:
  # Analyze TypeScript config patterns
  gh search patterns "tsconfig.json" --language json --min-pattern-count 3
  
  # Find common ESLint configurations
  gh search patterns "eslint.config" --language javascript --group-by property
  
  # Analyze Docker patterns with ranking
  gh search patterns "dockerfile" --rank-by frequency --show-examples 5

Flags:
  # Pattern Detection
  --min-pattern-count int       Minimum occurrences to consider a pattern (default: 2)
  --group-by string             Group patterns by: property, structure, value
  --show-examples int           Number of examples per pattern (default: 3)
  --rank-by string              Rank by: frequency, stars, recency
  
  # Analysis Depth  
  --deep-analysis               Enable deep structural analysis (slower)
  --exclude-tests               Exclude test files from analysis
  --focus-config                Focus on configuration-like files only
  
  # Inherit from main search
  --language, -l string         Programming language filter
  --repo, -r strings            Repository filter
  --limit int                   Results to analyze (default: 100)
```

### **3. Saved Searches: `gh search saved`**

```bash
Usage: gh search saved <command> [args]

Description:
  Manage saved searches for repeated configuration research workflows.

Commands:
  list                          List all saved searches
  run <name>                    Execute a saved search  
  save <name> <query> [flags]   Save a new search
  edit <name>                   Edit saved search (opens editor)
  delete <name>                 Delete saved search
  export [file]                 Export searches to YAML file
  import <file>                 Import searches from YAML file

Examples:
  # Save common research queries
  gh search saved save "react-configs" "tsconfig.json" --repo "*react*" --language json
  gh search saved save "docker-examples" "dockerfile" --min-stars 100
  
  # Use saved searches  
  gh search saved run react-configs
  gh search saved run docker-examples --limit 20  # Override saved limit
  
  # Manage saved searches
  gh search saved list
  gh search saved edit react-configs  
  gh search saved delete old-search
  
  # Team sharing
  gh search saved export team-searches.yaml
  gh search saved import team-searches.yaml

Saved Search Format:
  name: react-configs
  description: TypeScript configs from React projects  
  query: tsconfig.json
  filters:
    language: json
    repo: ["*react*", "*next*"]
    min_stars: 50
  created: 2024-01-15T10:30:00Z
  last_used: 2024-01-20T14:15:00Z
  use_count: 15
```

### **4. Template Generation: `gh search template`**

```bash
Usage: gh search template <query> [flags]

Description:
  Generate configuration templates by analyzing common patterns across
  multiple real-world examples.

Examples:
  # Generate TypeScript config template
  gh search template "tsconfig.json" --language json --output tsconfig.template.json
  
  # Create Docker template from popular examples
  gh search template "dockerfile" --min-stars 1000 --output Dockerfile.template
  
  # Generate with pattern analysis
  gh search template "vite.config" --analyze-patterns --merge-common --output vite.config.template.js

Flags:
  # Template Generation
  --output, -o string           Output file path (required)
  --merge-common                Merge commonly used properties
  --include-comments            Add explanatory comments  
  --pattern-threshold float     Pattern inclusion threshold 0.0-1.0 (default: 0.3)
  
  # Source Filtering (inherit from main search)
  --language, -l string         Programming language filter
  --min-stars int               Minimum repository stars for template sources
  --limit int                   Examples to analyze (default: 50)
  
  # Template Style
  --style string                Template style: minimal, comprehensive, commented
  --format string               Force output format: json, yaml, javascript, etc.
```

### **5. Configuration Comparison: `gh search compare`**

```bash
Usage: gh search compare <file1> <file2> [flags]

Description:
  Compare two configuration files side-by-side, highlighting differences
  and showing common patterns from similar files on GitHub.

Examples:
  # Compare local configs  
  gh search compare tsconfig.json ../other-project/tsconfig.json
  
  # Compare with GitHub examples
  gh search compare package.json --github facebook/react:package.json
  
  # Find similar configs for comparison context
  gh search compare vite.config.js --find-similar --language javascript

Flags:
  # Comparison Mode
  --github string               Compare with GitHub file (repo:path format)
  --find-similar                Find similar configs for context
  --unified                     Show unified diff format
  --side-by-side                Show side-by-side comparison
  
  # Analysis
  --highlight-patterns          Highlight common patterns from GitHub
  --suggest-improvements        Suggest improvements based on popular patterns
  --show-context int            Lines of context around differences (default: 3)
  
  # Output
  --output, -o string           Save comparison to file
  --format string               Output format: default, json, markdown
```

## 🎨 **Output Formats & UX**

### **Default Output Format (GitHub CLI style)**
```
🔍 Searching GitHub for: tsconfig.json language:typescript

✅ Found 87 results (showing top 10)

📁 facebook/react ⭐ 220k
   📄 tsconfig.json
   
   ```json
   {
     "compilerOptions": {
       "strict": true,
       "target": "es2019",
       "lib": ["dom", "dom.iterable", "es6"]
     }
   }
   ```
   🔗 https://github.com/facebook/react/blob/main/tsconfig.json

📁 vercel/next.js ⭐ 118k
   📄 packages/next/tsconfig.json
   
   ```json
   {
     "compilerOptions": {
       "strict": false,
       "target": "es5",  
       "lib": ["dom", "dom.iterable", "es2017"]
     }
   }
   ```
   🔗 https://github.com/vercel/next.js/blob/canary/packages/next/tsconfig.json

💡 Common patterns found:
   • "strict": true (73% of configs)
   • "target": "es2019" (45% of configs)  
   • "lib" includes "dom" (89% of configs)

⚡ Search completed in 1.2s
💾 Use --save <name> to save this search for reuse
```

### **JSON Output Format (for automation)**
```json
{
  "query": "tsconfig.json language:typescript",
  "total_count": 87,
  "results": [
    {
      "repository": {
        "full_name": "facebook/react",
        "html_url": "https://github.com/facebook/react",
        "stargazers_count": 220000,
        "updated_at": "2024-01-20T10:30:00Z"
      },
      "file": {
        "name": "tsconfig.json",
        "path": "tsconfig.json",
        "html_url": "https://github.com/facebook/react/blob/main/tsconfig.json",
        "content": "{\n  \"compilerOptions\": {\n    \"strict\": true\n  }\n}",
        "context_lines": 20
      },
      "matches": [
        {
          "text": "strict",
          "line_number": 3,
          "surrounding_context": "..."
        }
      ]
    }
  ],
  "patterns": {
    "common_properties": {
      "strict": {"frequency": 0.73, "values": {"true": 0.9, "false": 0.1}},
      "target": {"frequency": 0.45, "values": {"es2019": 0.6, "es2020": 0.4}}
    }
  },
  "metadata": {
    "search_time_ms": 1200,
    "rate_limit_remaining": 29,
    "cached": false
  }
}
```

### **Compact Output (for scripting)**
```bash
# --format compact
facebook/react:tsconfig.json:https://github.com/facebook/react/blob/main/tsconfig.json
vercel/next.js:packages/next/tsconfig.json:https://github.com/vercel/next.js/blob/canary/packages/next/tsconfig.json
microsoft/vscode:tsconfig.json:https://github.com/microsoft/vscode/blob/main/tsconfig.json
```

## 🚨 **Error Handling & User Guidance** 

### **Rate Limiting (Following gh-comment's intelligent approach)**
```bash
❌ GitHub search rate limit exceeded

🔍 Rate Limit Status:
   • Limit: 30 searches per minute
   • Remaining: 0
   • Reset: in 12m 34s

💡 Solutions:
   • Wait 12m 34s for automatic reset
   • Use more specific filters: --language, --repo, --filename
   • Search specific repositories: --repo facebook/react
   • Use saved searches: gh search saved list

📊 Check current status: gh search --rate-limit
⚡ Try this instead: gh search "config" --repo facebook/react --language json
```

### **Invalid Query Syntax**
```bash
❌ Invalid search query: unexpected token 'AND' at position 15

🔍 Your query: config AND OR typescript
                    ^^^^
                    Problem here

💡 GitHub Search Syntax:
   • Exact phrases: "exact match"
   • Boolean: config AND typescript (not OR after AND)
   • Exclusions: config NOT test  
   • Wildcards: *.config.js
   • Qualifiers: language:typescript filename:config.json

📖 Corrected examples:
   gh search "config typescript"                    # phrase search
   gh search config --language typescript          # using flags  
   gh search "config AND typescript"               # boolean AND
   gh search "typescript OR javascript"            # boolean OR
   
🚀 Try: gh search "config typescript" --language typescript
```

### **No Results Found**
```bash
❌ No results found for: extremely-rare-config-name-12345

💡 Try These Approaches:
   • Broaden search terms: "config" instead of "extremely-rare-config-name-12345"
   • Check spelling and remove typos
   • Search popular repositories: --repo facebook/react --repo microsoft/vscode
   • Use broader filters: --language javascript (instead of typescript)
   • Try related terms: "configuration", "setup", "options"

🔍 Alternative searches:
   gh search "config" --language javascript --limit 10
   gh search "package.json" --path examples/ --limit 5
   gh search --saved popular-configs  # if you have saved searches

📖 Browse common patterns: gh search patterns --help
```

### **Authentication Issues**
```bash
❌ GitHub authentication required

🔍 Current status: Not authenticated
   • Search rate limit: 10/minute (very limited)
   • Authenticated limit: 30/minute

💡 Fix Authentication:
   1. Check current status: gh auth status
   2. Login to GitHub: gh auth login
   3. Refresh if needed: gh auth refresh

📈 Benefits of authentication:
   • 3x higher rate limits (30 vs 10 per minute)
   • Access to private repositories (if permissions allow)
   • Detailed repository metadata

🚀 After authentication: gh search "your query here"
```

## ⚙️ **Configuration System**

### **Config File Format (.gh-search.yaml)**
```yaml
# Global defaults for all searches
defaults:
  language: ""              # Default language filter
  max_results: 50           # Default result limit  
  context_lines: 20         # Default context around matches
  output_format: "default"  # default, json, markdown, compact
  min_stars: 0              # Minimum repository stars
  sort_by: "relevance"      # relevance, stars, updated, created

# Saved searches for reuse
saved_searches:
  react-configs:
    description: "TypeScript configs from React projects"
    query: "tsconfig.json"
    filters:
      language: "json"
      repo: ["*react*", "*next*"]
      min_stars: 100
    created: "2024-01-15T10:30:00Z"
    last_used: "2024-01-20T14:15:00Z"
    use_count: 15
    
  docker-examples:
    description: "Dockerfile examples from popular projects"
    query: "dockerfile"
    filters:
      filename: "dockerfile"
      min_stars: 500
    created: "2024-01-10T09:00:00Z"
    use_count: 8

# Pattern analysis settings
analysis:
  min_pattern_count: 2      # Minimum occurrences to detect pattern
  enable_deep_analysis: false  # Enable slower but more thorough analysis
  exclude_test_files: true  # Skip test files in pattern analysis
  pattern_threshold: 0.3    # Threshold for including patterns in templates

# Output customization
output:
  color_mode: "auto"        # auto, always, never
  editor_command: "cursor"  # Command to open results
  save_path: "~/gh-search-results"  # Where to save result files
  show_patterns: true       # Show pattern analysis in results
  max_content_lines: 50     # Maximum lines of file content to show

# GitHub API settings
github:
  timeout: "30s"            # API request timeout
  retry_count: 3            # Number of retries on failure
  rate_limit_buffer: 5      # Keep this many requests in buffer
```

---

**Next**: See MIGRATION_GUIDE.md for converting from TypeScript to Go and IMPLEMENTATION_CHECKLIST.md for development tracking.