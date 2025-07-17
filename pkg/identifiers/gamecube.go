package identifiers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

// GameCubeIdentifier implements game identification for GameCube
type GameCubeIdentifier struct {
	db *database.GameDatabase
}

// NewGameCubeIdentifier creates a new GameCube identifier
func NewGameCubeIdentifier(db *database.GameDatabase) *GameCubeIdentifier {
	return &GameCubeIdentifier{db: db}
}

// Console returns the console name
func (g *GameCubeIdentifier) Console() string {
	return "GC"
}

// Identify identifies a GameCube game and returns its metadata
func (g *GameCubeIdentifier) Identify(path string) (map[string]string, error) {
	return g.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a GameCube game with additional parameters
func (g *GameCubeIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open disc image (ISO, CUE/BIN, or mounted directory)
	disc, err := iso9660.OpenImage(path, discUUID, discLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to open disc: %w", err)
	}
	defer disc.Close()

	// Read GameCube header (0x440 bytes) from the disc image
	// https://hitmen.c02.at/files/yagcd/yagcd/chap13.html#sec13
	header, err := disc.ReadFile(0, 0x440) // Read from LBA 0, size 0x440
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	if len(header) < 0x440 {
		return nil, fmt.Errorf("file too small: expected at least 0x440 bytes, got %d", len(header))
	}

	// Extract fields from header
	result := make(map[string]string)

	// ID at 0x0000 (4 bytes)
	id := strings.TrimRight(string(header[0x0000:0x0004]), "\x00")
	result["ID"] = id

	// Maker code at 0x0004 (2 bytes)
	makerCode := strings.TrimRight(string(header[0x0004:0x0006]), "\x00")
	result["maker_code"] = makerCode

	// Disk ID at 0x0006 (1 byte)
	diskID := header[0x0006]
	result["disk_ID"] = strconv.Itoa(int(diskID))

	// Version at 0x0007 (1 byte)
	version := header[0x0007]
	result["version"] = strconv.Itoa(int(version))

	// Internal title at 0x0020 (up to 0x3E0 bytes, ends at 0x400)
	titleBytes := header[0x0020:0x400]
	// Python preserves raw bytes including nulls
	internalTitle := string(titleBytes)
	result["internal_title"] = internalTitle

	// Try to look up game in database
	if g.db != nil && id != "" {
		if gameData, found := g.db.LookupGame("GC", id); found {
			// Add database fields to result
			for key, value := range gameData {
				// Override existing data if preferDB is set, otherwise only add new
				_, exists := result[key]
				if preferDB || !exists {
					result[key] = value
				}
			}
		}
	}

	// If no title from database, use internal title (raw bytes)
	if result["title"] == "" {
		result["title"] = internalTitle
	}

	return result, nil
}
