# Contributing to gh-code-search

Thank you for your interest in contributing to gh-code-search! This project is inspired by the great [ghx](https://github.com/johnlindquist/ghx), just reimagined as a GitHub CLI extension with enhanced features and professional architecture.

## 🚀 Quick Start

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/gh-code-search.git
   cd gh-code-search
   ```

2. **Set up development environment**
   ```bash
   go mod tidy
   go build -o gh-code-search
   ./gh-code-search --help
   ```

3. **Run tests**
   ```bash
   go test ./...
   go test -race -coverprofile=coverage.out ./...
   ```

## 📋 Development Guidelines

### Code Standards
- Follow Go conventions and `gofmt` formatting
- Maintain 85%+ test coverage (following gh-comment standards)
- Use table-driven tests with comprehensive scenarios
- Follow interface-based architecture with dependency injection
- Write actionable error messages with helpful suggestions

### Commit Messages
Use conventional commit format:
- `feat:` - New features
- `fix:` - Bug fixes  
- `docs:` - Documentation changes
- `test:` - Test additions/changes
- `refactor:` - Code restructuring

### Pull Request Process
1. Create a feature branch from `main`
2. Make your changes with tests
3. Ensure all tests pass and coverage remains high
4. Update documentation if needed
5. Submit PR with clear description

## 🏗️ Architecture

gh-code-search follows professional CLI patterns:

- **`cmd/`** - CLI commands using cobra
- **`internal/github/`** - GitHub API client with interfaces
- **`internal/search/`** - Query building and search logic
- **`internal/config/`** - Configuration management
- **`internal/output/`** - Output formatting

### Key Principles
- Interface-based design for testability
- Comprehensive error handling with user guidance
- Mock-first testing approach
- Clean separation of concerns

## 🧪 Testing

- Write table-driven tests for all new functionality
- Use mock clients for GitHub API interactions
- Test both success and error scenarios
- Include integration tests for command workflows
- Maintain compatibility with original ghx patterns

## 📖 Documentation

- Update README.md for new features
- Add examples for new functionality
- Document configuration options
- Keep help text comprehensive and accurate

## 🤝 Community

- Be respectful and inclusive
- Help others learn and contribute
- Share knowledge and best practices
- Report issues with clear reproduction steps

## 📞 Getting Help

- Check existing [issues](https://github.com/silouanwright/gh-code-search/issues)
- Review the [documentation](docs/)
- Join discussions in GitHub Discussions

---

Thank you for helping make gh-code-search better! 🙏