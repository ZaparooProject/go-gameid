# Code Review TODO List

Generated from comprehensive code review on 2025-07-16
Last updated: 2025-07-17

## Critical Security Issues (MUST FIX IMMEDIATELY)

- [ ] **Path Traversal Vulnerability** - Add path validation in fileio.go to prevent directory traversal attacks
- [ ] **Memory Exhaustion Risk** - Add size limits to io.ReadAll() in fileio.go:75
- [ ] **Directory Traversal DoS** - Add limits to recursive directory walking in GetSize() (fileio.go:91)
- [ ] **CUE File Path Injection** - Sanitize path joins in BinsFromCue() (fileio.go:150)

## Critical Code Issues

- [ ] **Remove Debug Print** - Remove fmt.Println statement in database.go:57
- [ ] **Fix Hardcoded Path** - Remove hardcoded "dbs/PSX.data.tsv" path in database.go:15

## High Priority Issues

- [ ] **Consolidate Database Layer** - Merge database.go and loader.go into single coherent implementation
- [x] **Add Tests** - ~~Current coverage ~15%~~ Significantly improved with comprehensive test suites for mounted disc functionality, all identifiers have tests
- [ ] **Add Error Types** - Define custom error types for better error handling
- [ ] **Implement Logging** - Replace fmt.Fprintln with proper logging framework (e.g., logrus, zap)
- [ ] **Add Documentation** - Add godoc comments to all exported types and functions

## Medium Priority Issues

- [x] **Complete TODO Features** - ✅ Implemented discUUID, discLabel, preferDB parameters in main.go and all identifiers
- [ ] **ISO9660 Recursive Listing** - Implement recursive directory listing (iso9660.go:281)
- [ ] **CUE Support** - Complete CUE file support for Saturn/SegaCD identifiers
- [ ] **URL Database Loading** - Implement LoadDatabaseFromURL in loader.go:48
- [ ] **Configuration System** - Add configuration management (viper, env vars, etc.)

## Low Priority Issues

- [ ] **Add Caching** - Implement caching layer for database lookups
- [ ] **Optimize String Operations** - Improve string handling in tight loops
- [ ] **Reduce Code Duplication** - Extract common patterns from identifiers
- [ ] **Standardize Error Messages** - Make error messages consistent across codebase

## Performance Improvements

- [ ] **Resource Limits** - Add configurable limits for file operations
- [ ] **Connection Pooling** - Implement connection pooling for database operations
- [ ] **Optimize File Reading** - Avoid reading files multiple times in same code path

## Testing Checklist

- [ ] Unit tests for all identifiers (GB, GBC, GBA, N64, SNES, Genesis, PSX, PS2, PSP, GameCube, Saturn, SegaCD)
- [ ] Integration tests for database operations
- [ ] Security tests for path validation
- [ ] Performance tests for large files
- [ ] Edge case tests (corrupted files, empty files, etc.)

## Documentation Tasks

- [ ] Update README with security considerations
- [ ] Document all identifier formats
- [ ] Add API documentation
- [ ] Create developer guide
- [ ] Document database schema

## Positive Findings (Already Done Well)

- ✓ All console identifiers implemented
- ✓ Good error handling with fmt.Errorf wrapping
- ✓ Clean interface design for identifiers
- ✓ Proper resource cleanup with defer
- ✓ Well-structured project layout
- ✓ Binary parsing well-implemented
- ✓ ISO9660 implementation solid (except recursive listing)
- ✓ Mounted disc directory support added with comprehensive tests
- ✓ Advanced CLI features (disc_uuid, disc_label, prefer_gamedb) fully implemented
- ✓ IdentifierWithOptions interface added for enhanced functionality
- ✓ Unified DiscImage interface for ISO files and directories

## Priority Order for Fixes

1. Security vulnerabilities (path validation, size limits)
2. Remove debug print and hardcoded paths
3. Consolidate database layer
4. Add comprehensive tests
5. Implement logging framework
6. Complete TODO features
7. Performance optimizations
8. Documentation

## Notes

- All identifiers (including Saturn and SegaCD) are actually implemented
- ISO9660 support is good but needs recursive directory listing
- Consider using Go modules for dependency management
- Consider adding CI/CD pipeline with security scanning
- Phase 5 (CLI & Integration) has been completed with all advanced features
- Mounted disc support allows treating directories as ISO images
- Test coverage significantly improved with testgen tool usage
- Ready for production testing with real game files