package identifiers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

// TestIdentifier provides common test functionality for all identifier types
type TestIdentifier interface {
	Identifier
	GenerateTestROM(t *testing.T, gameID string) []byte
}

// CreateTestDatabase creates a minimal test database for testing
func CreateTestDatabase() *database.GameDatabase {
	return &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GBA": {
				"AGSE": database.GameMetadata{
					"ID":    "AGSE",
					"title": "Golden Sun",
				},
				"BPRE": database.GameMetadata{
					"ID":    "BPRE",
					"title": "Pokemon Ruby",
				},
			},
			"GB_GBC": {
				"POKEMON_RED,0x91": database.GameMetadata{
					"ID":    "POKEMON_RED",
					"title": "Pokemon Red",
				},
			},
			"N64": {
				"NSMJ": database.GameMetadata{
					"ID":    "NSMJ",
					"title": "Super Mario 64",
				},
			},
			"SNES": {
				"SUPER_MARIO_WORLD": database.GameMetadata{
					"ID":    "SUPER_MARIO_WORLD",
					"title": "Super Mario World",
				},
			},
			"Genesis": {
				"MK4402": database.GameMetadata{
					"ID":    "MK-4402",
					"title": "Sonic the Hedgehog",
				},
			},
			"PSX": {
				"SCUS_94163": database.GameMetadata{
					"ID":    "SCUS-94163",
					"title": "Final Fantasy VII",
				},
			},
			"PS2": {
				"SLUS_20062": database.GameMetadata{
					"ID":    "SLUS-20062",
					"title": "Grand Theft Auto: Vice City",
				},
			},
			"GC": {
				"GALE01": database.GameMetadata{
					"ID":    "GALE01",
					"title": "Super Smash Bros. Melee",
				},
			},
			"Saturn": {
				"MK81005": database.GameMetadata{
					"ID":    "MK-81005",
					"title": "Virtua Fighter",
				},
			},
			"SegaCD": {
				"MK4407": database.GameMetadata{
					"ID":    "MK-4407",
					"title": "Sonic CD",
				},
			},
			"PSP": {
				"UCUS98701": database.GameMetadata{
					"ID":    "UCUS-98701",
					"title": "Grand Theft Auto: Liberty City Stories",
				},
			},
		},
	}
}

// CreateTestFile creates a temporary test file with the given content
func CreateTestFile(t *testing.T, content []byte, filename string) string {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, filename)

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	return testFile
}

// TestIdentifierBasics tests basic identifier functionality
func TestIdentifierBasics(t *testing.T, identifier Identifier, testFile string) {
	// test console name
	console := identifier.Console()
	if console == "" {
		t.Error("Console() returned empty string")
	}

	// test identification
	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result == nil {
		t.Fatal("Identify() returned nil result")
	}

	// all identifiers should return at least an ID
	if result["ID"] == "" {
		t.Error("Identify() did not return an ID")
	}
}

// TestIdentifierErrors tests error handling for identifiers
func TestIdentifierErrors(t *testing.T, identifier Identifier) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Non-existent file",
			path:    "/nonexistent/file.rom",
			wantErr: true,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Directory instead of file",
			path:    t.TempDir(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := identifier.Identify(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Identify() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Error("Identify() returned nil result for valid case")
			}
		})
	}
}

// TestIdentifierWithDatabase tests identifier behavior with database
func TestIdentifierWithDatabase(t *testing.T, identifier Identifier, testFile string, expectedID string) {
	db := CreateTestDatabase()

	// create new identifier with database
	var identifierWithDB Identifier

	switch id := identifier.(type) {
	case *GBAIdentifier:
		identifierWithDB = NewGBAIdentifier(db)
	case *GenesisIdentifier:
		identifierWithDB = NewGenesisIdentifier(db)
	case *GameCubeIdentifier:
		identifierWithDB = NewGameCubeIdentifier(db)
	case *PSXIdentifier:
		identifierWithDB = NewPSXIdentifier(db)
	case *PS2Identifier:
		identifierWithDB = NewPS2Identifier(db)
	case *PSPIdentifier:
		identifierWithDB = NewPSPIdentifier(db)
	case *SaturnIdentifier:
		identifierWithDB = NewSaturnIdentifier(db)
	case *SegaCDIdentifier:
		identifierWithDB = NewSegaCDIdentifier(db)
	default:
		t.Fatalf("Unknown identifier type: %T", id)
	}

	result, err := identifierWithDB.Identify(testFile)
	if err != nil {
		t.Fatalf("Identify() with database error = %v", err)
	}

	if result["ID"] != expectedID {
		t.Errorf("Expected ID %s, got %s", expectedID, result["ID"])
	}

	// with database, should have title
	if result["title"] == "" {
		t.Error("Expected title from database")
	}
}

// TestIdentifierInvalidFile tests identifier behavior with invalid files
func TestIdentifierInvalidFile(t *testing.T, identifier Identifier) {
	// create invalid file (too small)
	invalidFile := CreateTestFile(t, []byte("too small"), "invalid.rom")

	result, err := identifier.Identify(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid file")
	}
	if result != nil {
		t.Error("Expected nil result for invalid file")
	}
}

// TestIdentifierConcurrency tests concurrent access to identifier
func TestIdentifierConcurrency(t *testing.T, identifier Identifier, testFile string) {
	// create multiple goroutines calling Identify
	const numGoroutines = 10
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := identifier.Identify(testFile)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}()
	}

	// collect results
	var firstResult map[string]string
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if firstResult == nil {
				firstResult = result
			} else {
				// all results should be identical
				if result["ID"] != firstResult["ID"] {
					t.Errorf("Concurrent results differ: got %s, want %s", result["ID"], firstResult["ID"])
				}
			}
		case err := <-errors:
			t.Errorf("Concurrent error: %v", err)
		}
	}
}

// AssertFieldEquals asserts that a field in the result equals the expected value
func AssertFieldEquals(t *testing.T, result map[string]string, field, expected string) {
	if actual := result[field]; actual != expected {
		t.Errorf("Field %s: got %q, want %q", field, actual, expected)
	}
}

// AssertFieldNotEmpty asserts that a field in the result is not empty
func AssertFieldNotEmpty(t *testing.T, result map[string]string, field string) {
	if actual := result[field]; actual == "" {
		t.Errorf("Field %s should not be empty", field)
	}
}

// AssertFieldExists asserts that a field exists in the result
func AssertFieldExists(t *testing.T, result map[string]string, field string) {
	if _, exists := result[field]; !exists {
		t.Errorf("Field %s should exist in result", field)
	}
}

// GenerateGBATestROM generates a minimal valid GBA ROM for testing
func GenerateGBATestROM(gameID string) []byte {
	header := make([]byte, 192)

	// Nintendo logo (required for valid GBA ROM)
	copy(header[0x04:], gbaLogo)

	// Game title
	copy(header[0xA0:], []byte(gameID+"_TITLE"))

	// Game code
	copy(header[0xAC:], []byte(gameID))

	// Maker code
	copy(header[0xB0:], []byte("01"))

	// Other required fields
	header[0xB3] = 0x00 // Main unit code
	header[0xB4] = 0x00 // Device type
	header[0xBC] = 0x00 // Software version

	return header
}

// GenerateGenesisTestROM generates a minimal valid Genesis ROM for testing
func GenerateGenesisTestROM(gameID string) []byte {
	header := make([]byte, 0x200)

	// Magic word at 0x100
	copy(header[0x100:], []byte("SEGA GENESIS    "))

	// Publisher
	copy(header[0x113:], []byte("SEGA"))

	// Game ID
	copy(header[0x182:], []byte(gameID))

	// Title
	copy(header[0x150:], []byte(gameID+" TEST GAME"))

	return header
}

// GenerateGameCubeTestROM generates a minimal valid GameCube ROM for testing
func GenerateGameCubeTestROM(gameID string) []byte {
	header := make([]byte, 0x440)

	// Game ID
	copy(header[0x000:], []byte(gameID))

	// Maker code
	copy(header[0x004:], []byte("01"))

	// Internal title
	copy(header[0x020:], []byte(gameID+" TEST"))

	return header
}

// GenerateSaturnTestROM generates a minimal valid Saturn ROM for testing
func GenerateSaturnTestROM(gameID string) []byte {
	header := make([]byte, 0x100)

	// Magic word
	copy(header[0x000:], []byte("SEGA SEGASATURN"))

	// Product ID
	copy(header[0x020:], []byte(gameID))

	// Title
	copy(header[0x060:], []byte(gameID+" TEST GAME"))

	return header
}

// GenerateSegaCDTestROM generates a minimal valid SegaCD ROM for testing
func GenerateSegaCDTestROM(gameID string) []byte {
	header := make([]byte, 0x300)

	// Magic word
	copy(header[0x000:], []byte("SEGADISCSYSTEM"))

	// ID field
	copy(header[0x180:], []byte("GM "+gameID+" -00"))

	// Title
	copy(header[0x150:], []byte(gameID+" TEST GAME"))

	return header
}
