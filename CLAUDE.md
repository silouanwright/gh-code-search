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

## Current Status
- ✅ Go module initialized with dependencies
- ✅ Directory structure created following architecture spec
- 🚧 Implementing GitHub API client interface (following gh-comment patterns)
- 📋 15 todos queued for foundational work

## Reference Commands
When implementing features, refer to gh-comment's patterns:
```bash
# Check gh-comment architecture
ls ~/repos/gh-comment/cmd/
ls ~/repos/gh-comment/internal/

# Review testing patterns
find ~/repos/gh-comment -name "*_test.go" | head -5

# Study error handling
grep -r "formatActionableError" ~/repos/gh-comment/
```

Always maintain the same quality standards that made gh-comment successful.