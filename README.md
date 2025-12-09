# go-gameid

A Go library for identifying video game ROM and disc images. Detects console types from file extensions and headers, then extracts game metadata (IDs, titles, regions) from various retro gaming formats. Supports Game Boy, GBA, NES, SNES, N64, Genesis, GameCube, PlayStation, PS2, PSP, Saturn, Sega CD, and Neo Geo CD.

## Installation

```bash
go get github.com/ZaparooProject/go-gameid
```

## Usage

```go
import "github.com/ZaparooProject/go-gameid"

// Auto-detect console and identify
result, err := gameid.Identify("game.gba", nil)

// With database for title lookups
db, _ := gameid.LoadDatabase("games.gob.gz")
result, err := gameid.Identify("game.iso", db)

// Specify console explicitly
result, err := gameid.IdentifyWithConsole("game.bin", gameid.ConsolePSX, db)
```

## CLI

```bash
# Build
make gameid

# Run
./cmd/gameid/gameid -i game.gba -json
./cmd/gameid/gameid -i game.iso -c PSX -db games.gob.gz
```

## Acknowledgements

This project is a Go port of [GameID](https://github.com/niemasd/GameID) by [Niema Moshiri](https://github.com/niemasd). The original Python implementation and game database are the foundation of this work.

Test data files in `testdata/` are from the [240p Test Suite](https://github.com/ArtemioUrbina/240pTestSuite) by Artemio Urbina, licensed under GPL-3.0.

## License

GPL-3.0-or-later. See [LICENSE](LICENSE) for details.
