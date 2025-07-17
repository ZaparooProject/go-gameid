# go-gameid Project Status

## Overview
The go-gameid project is a Go port of Python tools (GameID and ConsoleID) for video game file identification. The port is now **functionally complete** for basic usage and achieves **80% test compatibility** with the Python version.

## Current Status (✅ COMPLETE)

### Phase 1: Core Infrastructure ✅
- Database conversion from Python pickle to JSON
- Core data structures and interfaces
- File I/O utilities (gzip, regular files, basic zip)
- Binary parsing helpers
- Error handling framework
- Comprehensive test framework

### Phase 2: Cartridge Systems ✅
- **GBA** - Game Boy Advance
- **GB/GBC** - Game Boy / Game Boy Color
- **N64** - Nintendo 64 (with endianness handling)
- **SNES** - Super Nintendo (LoROM/HiROM detection)
- **Genesis** - Sega Genesis/Mega Drive

### Phase 3: ISO 9660 Foundation ✅
- ISO 9660 disc image parser
- CUE/BIN file handling
- Sector size detection (2048/2352)
- File extraction from ISOs

### Phase 4: Disc Systems ✅
- **PSX/PS2** - PlayStation 1 & 2 (shared implementation)
- **GameCube** - Nintendo GameCube
- **Saturn** - Sega Saturn
- **SegaCD** - Sega CD
- **PSP** - PlayStation Portable

### Phase 5: CLI & Integration ✅ (Basic)
- Full command-line argument parsing
- Interactive mode when no arguments provided
- Output formatting with customizable delimiter
- Help and version flags
- Error handling for edge cases
- Test framework for CLI validation
- Comparison script for Python compatibility testing

### Phase 6: Compatibility Improvements ✅ (2025-07-17)
- **80% test compatibility achieved** (8/10 systems passing)
- Fixed GB licensee field mapping and title case handling
- Fixed GameCube string padding preservation
- Fixed SegaCD string handling and device support spacing
- Fixed PSX UUID formatting with dashes
- Fixed SNES field compatibility
- Added N64/PSP fallback logic for missing database entries
- Go implementation now more robust than Python for N64

## Remaining Tasks (Advanced Features)

### High Priority
1. **disc_uuid and disc_label parameters** - Pass these to disc-based identifiers for mounted disc support
2. **prefer_gamedb functionality** - Implement logic to prefer database metadata over file metadata

### Medium Priority
3. **Mounted disc directory support** - Add ISO 9660 support for reading from mounted directories
4. **Real game validation** - Test with production database and actual game files
5. **Performance optimization** - Benchmark against Python version

### Low Priority
6. **Documentation updates** - Complete README with usage examples
7. **CI/CD setup** - Automated testing and releases
8. **Binary releases** - Pre-compiled binaries for different platforms

## Usage

### Build
```bash
go build ./cmd/gameid
```

### Basic Usage
```bash
# Identify a game
./gameid -i game.gba -c GBA

# With database
./gameid -i game.iso -c PSX -d dbs/gameid_db.json

# Interactive mode
./gameid

# Custom output
./gameid -i game.rom -c SNES -o output.txt --delimiter "|"
```

### Testing
```bash
# Run all tests
go test ./...

# Run CLI tests
go test ./cmd/gameid/...

# Compare with Python version
python scripts/compare_outputs.py -i game.gba -c GBA
```

## Architecture

The project follows a modular architecture:
- `cmd/gameid/` - CLI implementation
- `pkg/identifiers/` - System-specific identifiers
- `pkg/database/` - Database loading and management
- `pkg/fileio/` - File I/O utilities
- `pkg/binary/` - Binary parsing helpers
- `pkg/iso9660/` - ISO 9660 file system support

## Known Limitations

1. **Advanced features not implemented**: disc_uuid, disc_label, prefer_gamedb
2. **Mounted disc support**: Not yet implemented for ISO 9660
3. **Database format**: Uses JSON instead of Python pickle (by design)

## Compatibility

The Go implementation aims for 100% output compatibility with the Python version. All game IDs and metadata fields should match exactly.

### Current Compatibility Status
- **8/10 systems**: Full byte-for-byte compatibility
- **N64**: Go provides fallback (Python errors without database)  
- **PSP**: Test data issue (both implementations fail on invalid ISO)

## Contributing

The project uses Test-Driven Development (TDD). All new features must have tests written first. The codebase follows idiomatic Go patterns and standard formatting.

## License

Same as the original Python GameID project.