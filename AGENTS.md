# go-gameid

A Go library for identifying video game ROM and disc images. Extracts game IDs, titles, and metadata from files for various retro gaming consoles.

## Project Purpose

go-gameid detects console types from file extensions/headers and extracts game metadata (IDs, titles, regions) from ROM/disc images. Used by [Zaparoo](https://github.com/ZaparooProject) for NFC-based game launching.

## Quick Reference

```bash
# Build
make build              # Build all packages
make gameid             # Build CLI binary to cmd/gameid/

# Test
make test               # Run unit tests with race detection
make coverage           # Generate HTML coverage report

# Lint
make lint               # Run golangci-lint
make lint-fix           # Auto-fix lint issues

# Pre-commit check
make check              # Lint + test

# Run CLI
./cmd/gameid/gameid -i game.gba -json
```

## Architecture

```
go-gameid/
├── gameid.go           # Main API: Identify(), IdentifyWithConsole(), DetectConsole()
├── console.go          # Console detection from file extensions/headers
├── database.go         # GameDatabase for metadata lookup (gob.gz format)
├── identifier/         # Console-specific identification logic
│   ├── identifier.go   # Identifier interface, Result type, Console constants
│   ├── gb.go           # Game Boy / Game Boy Color
│   ├── gba.go          # Game Boy Advance
│   ├── gc.go           # GameCube
│   ├── genesis.go      # Sega Genesis / Mega Drive
│   ├── n64.go          # Nintendo 64
│   ├── nes.go          # NES / Famicom
│   ├── snes.go         # SNES / Super Famicom
│   ├── psx.go          # PlayStation
│   ├── ps2.go          # PlayStation 2
│   ├── psp.go          # PlayStation Portable
│   ├── saturn.go       # Sega Saturn
│   ├── segacd.go       # Sega CD / Mega CD
│   └── neogeocd.go     # Neo Geo CD
├── iso9660/            # ISO9660 filesystem parsing (disc images)
│   ├── iso9660.go      # ISO reader implementation
│   ├── cue.go          # CUE sheet parsing
│   └── mounted.go      # Mounted disc support
├── internal/binary/    # Binary reading utilities
└── cmd/
    ├── gameid/         # CLI tool
    └── dbgen/          # Database generator
```

## Supported Consoles

| Console | Extensions | Media Type |
|---------|------------|------------|
| GB/GBC | .gb, .gbc | Cartridge |
| GBA | .gba, .srl | Cartridge |
| NES | .nes, .fds, .unf, .nez | Cartridge |
| SNES | .sfc, .smc, .swc | Cartridge |
| N64 | .n64, .z64, .v64, .ndd | Cartridge |
| Genesis | .gen, .md, .smd | Cartridge |
| GameCube | .gcm, .gcz, .rvz | Disc |
| PSX | .bin, .iso, .cue | Disc |
| PS2 | .bin, .iso, .cue | Disc |
| PSP | .iso, .cso | Disc |
| Saturn | .bin, .iso, .cue | Disc |
| Sega CD | .bin, .iso, .cue | Disc |
| Neo Geo CD | .bin, .iso, .cue | Disc |

## Code Patterns

### Adding a New Console Identifier

1. Create `identifier/<console>.go` implementing the `Identifier` interface:

```go
type Identifier interface {
    Identify(r io.ReaderAt, size int64, db Database) (*Result, error)
    Console() Console
}
```

2. For disc-based consoles, also implement `pathIdentifier`:

```go
type pathIdentifier interface {
    IdentifyFromPath(path string, db Database) (*Result, error)
}
```

3. Register in `gameid.go`:

```go
var identifiers = map[identifier.Console]identifier.Identifier{
    identifier.ConsoleXXX: identifier.NewXXXIdentifier(),
}
```

4. Add extension mappings in `console.go`

### Result Structure

```go
result := identifier.NewResult(console)
result.ID = "GAME-ID"
result.SetMetadata("ID", result.ID)
result.SetMetadata("internal_title", title)
result.SetMetadata("region", region)
```

### Database Lookup Keys

- **GB/GBC**: `(internal_title, global_checksum)` tuple
- **SNES**: `(developer_id, internal_name_hex, rom_version, checksum)` tuple
- **NES**: CRC32 hash (int)
- **GBA/GC/N64/Genesis**: Game code string
- **Disc consoles**: Serial number string
- **NeoGeoCD**: `(uuid, volume_id)` tuple

## Code Style

### License Header (Required)

Every `.go` file must have this header:

```go
// Copyright (c) 2025 Niema Moshiri and The Zaparoo Project.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This file is part of go-gameid.
//
// go-gameid is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gameid is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-gameid.  If not, see <https://www.gnu.org/licenses/>.
```

### Go Conventions

- **Go version**: 1.24+
- **Line length**: 120 characters max
- **Function length**: 50 statements / 80 lines max
- **Cyclomatic complexity**: 15 max
- **Arguments**: 5 max per function
- **Return values**: 3 max per function
- **Nesting depth**: 3 levels max

### Naming

- Use 2+ character variable names (exceptions: `err`, `id`, `i`, `j`, `k`, `db`, `tx`, `ctx`, `wg`)
- JSON tags use snake_case
- Error types: `ErrXxx` or `XxxError`

### Error Handling

- Wrap errors with `fmt.Errorf("context: %w", err)`
- Use custom error types:
  - `identifier.ErrNotSupported{Format: "xxx"}` for unsupported formats
  - `identifier.ErrInvalidFormat{Console: c, Reason: "xxx"}` for invalid data

### Test Patterns

```go
func TestXxx(t *testing.T) {
    t.Parallel()  // Always use parallel tests

    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"case name", "input", "expected", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Subtests also parallel
            // test logic
        })
    }
}
```

## Linting

The project uses golangci-lint with a strict configuration. Key enabled linters:

- **Security**: gosec, noctx
- **Errors**: errcheck, errorlint, wrapcheck, nilerr
- **Style**: revive, gocritic, gofumpt, gci
- **Complexity**: gocyclo, gocognit, cyclop, funlen, nestif
- **Performance**: prealloc, bodyclose
- **Testing**: paralleltest, tparallel, testifylint

When lint violations occur, either fix them or add a `//nolint:linter // reason` comment explaining why the exception is necessary.

## CI Pipeline

Tests run on Ubuntu, macOS, and Windows with:
- Unit tests with race detection
- golangci-lint
- govulncheck for security vulnerabilities
- Cross-compilation checks (linux/darwin/windows, amd64/arm64)
- Coverage reporting to Codecov

## Common Tasks

### Identify a game file

```go
import "github.com/ZaparooProject/go-gameid"

result, err := gameid.Identify("game.gba", nil)
// or with database
db, _ := gameid.LoadDatabase("games.gob.gz")
result, err := gameid.Identify("game.iso", db)
```

### Detect console from file

```go
console, err := gameid.DetectConsole("game.bin")
```

### Parse console name

```go
console, err := gameid.ParseConsole("psx")  // Returns ConsolePSX
```

### Check media type

```go
if gameid.IsDiscBased(console) {
    // Handle disc image
}
```

## Platform-Specific Code

Block device detection has platform-specific implementations:
- `blockdevice_unix.go` - Linux/macOS: checks `/dev/` prefix and `syscall.Stat_t` mode
- `blockdevice_windows.go` - Windows: checks `\\.\` prefix

## Dependencies

The project has zero external dependencies (stdlib only).

## Debugging Tips

1. **Console not detected**: Check extension mappings in `console.go` and magic bytes in `detectConsoleFromHeader()`

2. **Wrong metadata extracted**: Check the identifier's `Identify()` method and byte offsets for the ROM header format

3. **Database lookup fails**: Verify key format matches what the identifier generates (see Database Lookup Keys above)

4. **Disc not reading**: Check ISO9660 parsing in `iso9660/` - may need to handle different sector sizes (2048 vs 2352)

## Important Notes

- Disc-based identifiers need path (not reader) due to ISO filesystem parsing
- GBC uses the same identifier as GB (header format is identical)
- Some disc formats (.bin, .iso, .cue) are ambiguous - detection relies on header magic and filesystem analysis
- Block device support allows reading directly from physical disc drives
