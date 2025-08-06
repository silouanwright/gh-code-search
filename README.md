# gh-code-search

Search GitHub code from the command line. Inspired by [ghx](https://github.com/johnlindquist/ghx).

**GitHub CLI Extension** - Extends the `gh` command with intelligent code search capabilities. Learn more about [GitHub CLI extensions](https://cli.github.com/manual/gh_extension).

## Installation

```bash
gh extension install silouanwright/gh-code-search
```

Requires [GitHub CLI](https://cli.github.com/) (`gh auth login` to authenticate).

## 📖 Usage

### Basic Search

```bash
# Find TypeScript configuration files
gh code-search "tsconfig.json" --language json --limit 10

# Search for React hooks usage
gh code-search "useState" --language typescript --extension tsx

# Find Docker configurations in popular repositories
gh code-search "dockerfile" --min-stars 1000 --limit 5
```

### Advanced Filtering & Pagination

```bash
# Search specific repositories
gh code-search "vite.config" --repo vercel/next.js --repo facebook/react

# Filter by file location and size
gh code-search "component" --path "src/components" --size ">1000"

# Search by organization/owner (multiple supported)
gh code-search "interface" --owner microsoft --owner google --language typescript

# Page-based search (API efficient for large datasets)
gh code-search "config" --page 1 --limit 100        # Get first 100 results
gh code-search "config" --page 2 --limit 100        # Get next 100 results  
gh code-search "config" --page 3 --limit 50         # Get results 201-250
```

### Output Formats

```bash
# Default rich output with syntax highlighting
gh code-search "config" --language json

# Pipe-friendly output for scripting
gh code-search "hooks" --language typescript --pipe

# Save results to file
gh code-search "dockerfile" --format markdown > docker-examples.md
```

### Save and Reuse Searches

```bash
# Save a search for later use
gh code-search "eslint.config" --language javascript --save eslint-configs

# List saved searches
gh code-search saved list

# Run a saved search
gh code-search saved run eslint-configs
```

## 🎯 Common Use Cases

### Configuration Discovery
Perfect for finding working configuration examples:

```bash
# TypeScript project setup
gh code-search "tsconfig.json" --language json --min-stars 500

# ESLint configurations  
gh code-search "eslint.config" --language javascript --path "**/examples/**"

# Docker best practices
gh code-search "dockerfile" --min-stars 1000 --repo "**/production"

# GitHub Actions workflows
gh code-search "github/workflows" --filename "*.yml" --path ".github/workflows"
```

### Organization-Wide Code Search

Perfect for exploring patterns across all repositories in an organization:

```bash
# All TypeScript configs across Microsoft repos
gh code-search "tsconfig.json" --owner microsoft --language json

# React patterns from top organizations
gh code-search "useEffect" --owner facebook --owner vercel --language typescript

# Go best practices from Google's projects
gh code-search "interface" --owner google --language go --min-stars 1000
```

### Learning from Popular Projects

```bash
# Specific repository patterns
gh code-search "useEffect" --repo facebook/react --repo vercel/next.js

# Go patterns from well-maintained projects
gh code-search "interface" --language go --min-stars 5000 --limit 20

# Python best practices across organizations
gh code-search "class" --language python --owner google --owner microsoft
```

### API and Library Usage

```bash
# Find API usage examples
gh code-search "stripe.charges.create" --language javascript --min-stars 100

# Database patterns
gh code-search "prisma" --language typescript --path "**/models/**"

# Testing patterns
gh code-search "test" --filename "*test*" --language go --min-stars 1000
```

## ⚙️ Configuration

Create `~/.gh-code-search.yaml` for custom defaults:

```yaml
defaults:
  language: "typescript"
  max_results: 25
  context_lines: 30
  output_format: "default"
  min_stars: 100
  sort_by: "stars"

saved_searches:
  react-configs:
    description: "React TypeScript configurations"
    query: "tsconfig.json"
    filters:
      language: "json"
      repo: ["facebook/react", "vercel/next.js"]
      min_stars: 500

output:
  color_mode: "auto"
  show_patterns: true
  show_stars: true

github:
  timeout: "30s"
  retry_count: 3
```

## 🔍 Search Syntax

gh-code-search supports GitHub's powerful search syntax:

```bash
# Exact phrases
gh code-search "exact phrase match"

# Boolean operators
gh code-search "config AND typescript"
gh code-search "docker NOT test"

# Wildcards and patterns
gh code-search "*.config.js" --language javascript

# GitHub qualifiers
gh code-search "language:go filename:main.go stars:>100"
```

## 📊 Command Reference

### Global Flags
- `--verbose, -v`: Detailed output with timing and debug info
- `--dry-run`: Show what would be searched without executing
- `--config`: Custom configuration file path
- `--no-color`: Disable colored output

### Search Flags
- `--language, -l`: Programming language filter
- `--repo, -r`: Repository filter (supports wildcards)
- `--filename, -f`: Exact filename match
- `--extension, -e`: File extension filter
- `--path, -p`: File path filter
- `--owner, -o`: Repository owner/organization filter (multiple supported)
- `--size`: File size filter (e.g., ">1000", "<500")
- `--min-stars`: Minimum repository stars
- `--limit`: Maximum results per page (default: 50, max: 100)
- `--page`: Specific page number (more API efficient than auto-pagination)
- `--context`: Context lines around matches (default: 20)
- `--format`: Output format (default, json, markdown, compact)
- `--pipe`: Pipe-friendly output for scripting
- `--save`: Save search with given name

## 🏗️ Architecture

gh-code-search follows professional CLI development patterns:

- **Interface-Based Design**: Dependency injection for testability
- **Comprehensive Testing**: 85%+ test coverage with table-driven tests
- **Intelligent Error Handling**: Actionable error messages with solutions
- **Modular Architecture**: Clean separation of concerns
- **Performance Optimized**: Efficient Go implementation with caching


## 🐛 Troubleshooting

### Authentication Issues
```bash
# Check GitHub CLI authentication
gh auth status

# Re-authenticate if needed
gh auth login --web
gh auth refresh --scopes repo
```

### Rate Limiting
```bash
# Check current rate limits
gh code-search --rate-limit

# Use more specific filters to reduce API calls
gh code-search "config" --repo specific/repo --language json
```

### No Results Found
- Try broader search terms
- Remove or adjust filters  
- Check spelling and syntax
- Search in popular repositories with `--min-stars`

## 🔬 Development

### Building from Source
```bash
git clone https://github.com/silouanwright/gh-code-search.git
cd gh-code-search
go mod tidy
go build -o gh-code-search
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Project Structure
```
gh-code-search/
├── cmd/                    # CLI commands
├── internal/
│   ├── github/            # GitHub API client
│   ├── search/            # Search logic and query building
│   ├── config/            # Configuration management  
│   └── output/            # Output formatting
├── docs/                  # Documentation
└── examples/              # Configuration examples
```

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the great [ghx](https://github.com/johnlindquist/ghx) by [@johnlindquist](https://github.com/johnlindquist)
- Architecture patterns from [gh-comment](https://github.com/silouanwright/gh-comment)
- Uses [GitHub CLI](https://cli.github.com/) for authentication and follows their extension conventions

## 🔗 Links

- [GitHub Repository](https://github.com/silouanwright/gh-code-search)
- [Issues & Bug Reports](https://github.com/silouanwright/gh-code-search/issues)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

---

**Happy searching! 🔍** Find those configuration examples and level up your development workflow.