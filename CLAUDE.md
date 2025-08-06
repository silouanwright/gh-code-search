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

## Current Status - PHASE 1 COMPLETE ✅
- ✅ **Foundation Complete**: All 15 initial todos finished
- ✅ **Working Executable**: Builds and runs successfully
- ✅ **Professional Repository**: Clean documentation, CI/CD, MIT license
- ✅ **ghx Compatibility**: All original functionality preserved and enhanced
- ✅ **GitHub CLI Extension**: Ready for `gh extension install`

## Implementation Highlights
- **Interface-based architecture** with dependency injection for testability
- **Comprehensive error handling** with actionable user guidance
- **Full ghx feature parity** plus enhancements (min-stars, multiple formats, etc.)
- **Professional documentation** following gh-comment's clean style
- **Automated testing & releases** via GitHub Actions
- **Respectful ghx acknowledgment** without competitive language

## Next Phase Options
- **Enhanced Features**: Pattern analysis, saved searches, template generation
- **Performance**: Benchmarking, caching, optimization
- **Community**: Issue templates, contributor onboarding, feature requests
- **Advanced**: AI integration, team collaboration, enterprise features

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