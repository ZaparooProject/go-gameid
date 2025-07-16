# GameID Go Port Implementation Plan

## Overview
This document outlines the comprehensive plan for porting GameID.py to Go, maintaining full CLI compatibility and identical game identification output.

## Critical Requirements

### Test-Driven Development (TDD)
This project is an ideal candidate for TDD due to:
- **Deterministic outputs** - Each game file should produce exactly the same ID every time
- **Binary compatibility** - We must match Python's output byte-for-byte
- **Complex binary parsing** - Tests prevent regressions in bit-level operations
- **Multiple formats** - Each system has unique header structures to validate
- **Edge cases** - Malformed headers, unusual encodings, missing data

**Every feature MUST be implemented test-first:**
1. Write failing test with expected output from Python
2. Implement minimal code to pass
3. Refactor while keeping tests green

## Architecture Overview

```
┌─────────────────┐     ┌──────────────┐     ┌───────────────┐
│   CLI/Main      │────▶│  Identifier  │────▶│   Database    │
│  (cmd/gameid)   │     │  Interface   │     │  (JSON/TSV)   │
└─────────────────┘     └──────────────┘     └───────────────┘
         │                      │
         ▼                      ▼
┌─────────────────┐     ┌──────────────┐
│   File I/O      │     │ System Impls │
│  (gzip, zip)    │     │ (GBA, PSX...)│
└─────────────────┘     └──────────────┘
```

## Implementation Phases

### Phase 1: Core Infrastructure (Test-First)
- [ ] Write tests for database loading
- [ ] Database conversion from Python pickle to JSON
- [ ] Write tests for core data structures
- [ ] Core data structures and interfaces
- [ ] Write tests for file I/O operations
- [ ] File I/O utilities (gzip, regular files, basic zip)
- [ ] Write tests for binary parsing
- [ ] Binary parsing helpers
- [ ] Write tests for error scenarios
- [ ] Error handling framework

### Phase 2: Simple Systems (Cartridge-based)
- [ ] Write GBA identifier tests with Python reference data
- [ ] Complete GBA identifier (enhance existing)
- [ ] Write GB/GBC tests including edge cases
- [ ] GB/GBC identifier (similar header structure)
- [ ] Write N64 tests for both endianness formats
- [ ] N64 identifier (endianness handling)
- [ ] Write SNES tests for LoROM/HiROM detection
- [ ] SNES identifier (header location detection)
- [ ] Write Genesis tests with various magic words
- [ ] Genesis identifier (magic word search)

### Phase 3: ISO 9660 Foundation
- [ ] Write tests for ISO parsing
- [ ] ISO 9660 disc image parser
- [ ] Write tests for CUE parsing
- [ ] CUE/BIN file handling
- [ ] Write tests for sector size detection
- [ ] Sector size detection (2048/2352)
- [ ] Write tests for file extraction
- [ ] File extraction from ISOs
- [ ] Write tests for mounted discs
- [ ] Mounted disc directory support

### Phase 4: Complex Systems (Disc-based)
- [ ] Write PSX/PS2 tests with various serial formats
- [ ] PSX/PS2 identifiers (shared code)
- [ ] Write GameCube tests
- [ ] GameCube identifier
- [ ] Write Saturn tests
- [ ] Saturn identifier
- [ ] Write SegaCD tests
- [ ] SegaCD identifier
- [ ] Write PSP tests
- [ ] PSP identifier (UMD_DATA.BIN)

### Phase 5: CLI & Integration
- [ ] Write integration tests for CLI
- [ ] Full CLI argument parsing
- [ ] Write tests for interactive mode
- [ ] Interactive mode
- [ ] Write tests for output formatting
- [ ] Output formatting
- [ ] Create comprehensive test suite
- [ ] End-to-end validation against Python version

## Detailed Implementation Tasks

### 1. Database Management

#### Database Conversion Script
```python
# scripts/convert_db.py
# Convert pickle database to JSON format
# Structure: {"system": {"game_id": {metadata}}}
```

#### Go Database Structures
```go
// pkg/database/types.go
type GameMetadata map[string]string
type SystemDatabase map[string]GameMetadata
type GameDatabase struct {
    Systems map[string]SystemDatabase
}
```

### 2. Core Packages

#### pkg/fileio
- `OpenFile()` - Handle regular, gzip, stdin/stdout
- `GetSize()` - Support regular files and /dev/ volumes
- `GetExtension()` - Strip .gz and return actual extension
- `BinsFromCue()` - Parse CUE files for BIN references

#### pkg/binary
- `ReadUint16BE/LE()` - Endian-aware readers
- `ReadUint32BE/LE()` - 32-bit variants
- `ExtractString()` - Clean string extraction
- `CalculateChecksum()` - Various checksum algorithms

#### pkg/identifiers
Enhanced interface:
```go
type Identifier interface {
    Identify(path string) (map[string]string, error)
    Console() string
}
```

### 3. System-Specific Implementations

#### Cartridge Systems

**GBA (pkg/identifiers/gba.go)**
- Nintendo logo validation
- Header fields: title, game_code, maker_code
- Software version extraction

**GB/GBC (pkg/identifiers/gb.go)**
- Shared implementation for GB and GBC
- CGB flag detection
- Header/global checksum calculation
- Licensee code parsing

**N64 (pkg/identifiers/n64.go)**
- Endianness detection from first word
- Cartridge ID extraction
- Country code parsing

**SNES (pkg/identifiers/snes.go)**
- LoROM/HiROM detection
- Checksum validation
- Hardware type detection

**Genesis (pkg/identifiers/genesis.go)**
- Magic word search in header
- Region/device support parsing
- Software type detection

#### Disc Systems

**ISO 9660 Base (pkg/iso9660)**
- Primary Volume Descriptor parsing
- File listing from root directory
- UUID extraction
- Support for 2048/2352 byte sectors

**PSX/PS2 (pkg/identifiers/psx.go)**
- Shared implementation
- Serial from SXXX_XXX.XX files
- Volume ID fallback
- Root file listing

**Saturn (pkg/identifiers/saturn.go)**
- Magic word "SEGA SEGASATURN"
- Device support parsing
- Target area decoding

**GameCube (pkg/identifiers/gc.go)**
- Fixed header format
- Direct field extraction

**PSP (pkg/identifiers/psp.go)**
- UMD_DATA.BIN extraction
- Serial parsing

### 4. CLI Implementation

#### Command Line Arguments
```
--input, -i      Input game file (required)
--console, -c    Console type (required)
--database, -d   Database file path
--output, -o     Output file (default: stdout)
--disc_uuid      Pre-known disc UUID
--disc_label     Pre-known disc label
--delimiter      Output delimiter (default: \t)
--prefer_gamedb  Prefer database over file metadata
--version        Show version
```

#### Interactive Mode
- Triggered when no arguments provided
- Prompt for file and console
- Validate inputs before processing

### 5. Testing Strategy (TDD Approach)

#### Test Structure
```
test_data/
├── reference/      # Python output for comparison
├── samples/        # Small test ROM/ISO files
├── fixtures/       # Binary test data for unit tests
└── scripts/        # Comparison scripts

pkg/
├── binary/
│   └── binary_test.go    # Test binary parsing before implementation
├── fileio/
│   └── fileio_test.go    # Test file operations first
├── identifiers/
│   ├── gba_test.go       # Test each system thoroughly
│   ├── gb_test.go
│   └── ...
└── database/
    └── database_test.go  # Test DB loading/lookup
```

#### TDD Implementation Flow

**For each component:**

1. **Create Test Fixtures**
   ```go
   // gba_test.go
   func TestGBAIdentify_ValidROM(t *testing.T) {
       // Expected output from Python
       expected := map[string]string{
           "ID": "AGBE",
           "internal_title": "GOLDEN SUN",
           "maker_code": "01",
           "title": "Golden Sun",
       }
       
       // Test will fail initially
       result, err := identifyGBA("testdata/golden_sun.gba")
       assert.NoError(t, err)
       assert.Equal(t, expected, result)
   }
   ```

2. **Test Edge Cases First**
   ```go
   func TestGBAIdentify_InvalidNintendoLogo(t *testing.T) {
       // Should handle gracefully
   }
   
   func TestGBAIdentify_TruncatedFile(t *testing.T) {
       // Should return specific error
   }
   ```

3. **Table-Driven Tests**
   ```go
   var gbaTestCases = []struct {
       name     string
       file     string
       expected map[string]string
       wantErr  bool
   }{
       {"Valid ROM", "golden_sun.gba", map[string]string{...}, false},
       {"Bad Logo", "invalid_logo.gba", nil, true},
       {"Empty File", "empty.gba", nil, true},
   }
   ```

#### Critical Test Categories

1. **Binary Parsing Tests**
   - Endianness handling
   - String extraction with null termination
   - Checksum calculations
   - Bit field parsing

2. **File I/O Tests**
   - Regular files
   - Gzip compression
   - Stdin/stdout handling
   - /dev/ volume support
   - File size calculation

3. **System-Specific Tests**
   - Header validation
   - Logo checks
   - Checksum verification
   - Field extraction
   - Edge cases per system

4. **Integration Tests**
   - Full CLI execution
   - Output format validation
   - Cross-reference with Python

5. **Regression Tests**
   - Known problematic games
   - Format variations
   - Encoding issues

#### Test Data Generation

```python
# scripts/generate_test_data.py
# For each test ROM:
# 1. Run Python GameID
# 2. Save output as JSON
# 3. Create minimal test ROM if needed
# 4. Document expected behavior
```

#### Continuous Validation

```bash
# Run after each change
go test ./...

# Compare with Python
./scripts/compare_outputs.sh

# Coverage report
go test -cover ./...
```

## Current Status

### Completed
- [x] Basic project structure
- [x] GBA identifier skeleton
- [x] Database loading placeholder

### In Progress
- [ ] Database conversion script
- [ ] Enhanced identifier interface
- [ ] Binary parsing utilities

### Next Steps (TDD Workflow)
1. Set up test framework and structure
2. Generate test data from Python GameID
3. Write failing tests for database loading
4. Create database conversion script
5. Write failing tests for GBA identifier
6. Implement GameMetadata return type
7. Complete GBA identifier to pass tests
8. Continue with test-first approach for each component

## Success Criteria

1. **Functional Compatibility**
   - All CLI arguments work identically
   - Interactive mode matches Python
   - File format support complete

2. **Output Compatibility**
   - Identical game IDs produced
   - Same metadata fields
   - Matching output format

3. **Code Quality**
   - Idiomatic Go code
   - Comprehensive tests (>90% coverage target)
   - Clear documentation
   - All tests written before implementation
   - No code without corresponding tests

## Notes

- Database format change from pickle to JSON is necessary for Go
- ISO 9660 can use existing Go library or custom implementation
- Edge cases must be tested thoroughly for binary compatibility
- Performance should match or exceed Python version
- **TDD is mandatory** - No production code without failing test first
- Test data should be minimal but representative
- Use table-driven tests for multiple test cases
- Mock file systems for testing edge cases without real ROMs