# gh-search

Search GitHub code from the command line.

**GitHub CLI Extension** - Extends the `gh` command with intelligent code search capabilities. Inspired by the great [ghx](https://github.com/johnlindquist/ghx). Learn more about [GitHub CLI extensions](https://cli.github.com/manual/gh_extension).

## Installation

```bash
gh extension install silouanwright/gh-search
```

Requires [GitHub CLI](https://cli.github.com/) (`gh auth login` to authenticate).

## 📖 Usage

### Basic Search

```bash
# Find TypeScript configuration files
gh search "tsconfig.json" --language json --limit 10

# Search for React hooks usage
gh search "useState" --language typescript --extension tsx

# Find Docker configurations in popular repositories
gh search "dockerfile" --min-stars 1000 --limit 5
```

### Advanced Filtering

```bash
# Search specific repositories
gh search "vite.config" --repo vercel/next.js --repo facebook/react

# Filter by file location and size
gh search "component" --path "src/components" --size ">1000"

# Search by repository owner
gh search "interface" --owner microsoft --language typescript
```

### Output Formats

```bash
# Default rich output with syntax highlighting
gh search "config" --language json

# Pipe-friendly output for scripting
gh search "hooks" --language typescript --pipe

# Save results to file
gh search "dockerfile" --format markdown > docker-examples.md
```

### Save and Reuse Searches

```bash
# Save a search for later use
gh search "eslint.config" --language javascript --save eslint-configs

# List saved searches
gh search saved list

# Run a saved search
gh search saved run eslint-configs
```

## 🎯 Common Use Cases

### Configuration Discovery
Perfect for finding working configuration examples:

```bash
# TypeScript project setup
gh search "tsconfig.json" --language json --min-stars 500

# ESLint configurations  
gh search "eslint.config" --language javascript --path "**/examples/**"

# Docker best practices
gh search "dockerfile" --min-stars 1000 --repo "**/production"

# GitHub Actions workflows
gh search "github/workflows" --filename "*.yml" --path ".github/workflows"
```

### Learning from Popular Projects

```bash
# React patterns from top repositories
gh search "useEffect" --repo facebook/react --repo vercel/next.js

# Go patterns from well-maintained projects
gh search "interface" --language go --min-stars 5000 --limit 20

# Python best practices
gh search "class" --language python --owner google --owner microsoft
```

### API and Library Usage

```bash
# Find API usage examples
gh search "stripe.charges.create" --language javascript --min-stars 100

# Database patterns
gh search "prisma" --language typescript --path "**/models/**"

# Testing patterns
gh search "test" --filename "*test*" --language go --min-stars 1000
```

## ⚙️ Configuration

Create `~/.gh-search.yaml` for custom defaults:

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

gh-search supports GitHub's powerful search syntax:

```bash
# Exact phrases
gh search "exact phrase match"

# Boolean operators
gh search "config AND typescript"
gh search "docker NOT test"

# Wildcards and patterns
gh search "*.config.js" --language javascript

# GitHub qualifiers
gh search "language:go filename:main.go stars:>100"
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
- `--owner, -o`: Repository owner filter
- `--size`: File size filter (e.g., ">1000", "<500")
- `--min-stars`: Minimum repository stars
- `--limit`: Maximum results (default: 50, max: 1000)
- `--context`: Context lines around matches (default: 20)
- `--format`: Output format (default, json, markdown, compact)
- `--pipe`: Pipe-friendly output for scripting
- `--save`: Save search with given name

## 🏗️ Architecture

gh-search follows professional CLI development patterns:

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
gh search --rate-limit

# Use more specific filters to reduce API calls
gh search "config" --repo specific/repo --language json
```

### No Results Found
- Try broader search terms
- Remove or adjust filters  
- Check spelling and syntax
- Search in popular repositories with `--min-stars`

## 🔬 Development

### Building from Source
```bash
git clone https://github.com/silouanwright/gh-search.git
cd gh-search
go mod tidy
go build -o gh-search
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
gh-search/
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

- [GitHub Repository](https://github.com/silouanwright/gh-search)
- [Issues & Bug Reports](https://github.com/silouanwright/gh-search/issues)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

---

**Happy searching! 🔍** Find those configuration examples and level up your development workflow.