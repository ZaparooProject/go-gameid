package identifiers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

// Test with real game files
func TestGBIdentify_RealGames(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("..", "..", "test_data", "reference", "gb_test_data.json")
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Skipf("Test data not found: %v", err)
	}

	var testData map[string]map[string]struct {
		Path     string            `json:"path"`
		Expected map[string]string `json:"expected"`
	}
	if err := json.Unmarshal(data, &testData); err != nil {
		t.Fatalf("Failed to parse test data: %v", err)
	}

	// Load database
	dbPath := filepath.Join("..", "..", "dbs", "gameid_db.json")
	db, err := database.LoadDatabase(dbPath)
	if err != nil {
		t.Skipf("Database not found: %v", err)
	}

	identifier := &GBIdentifier{db: db}

	// Test each game
	gbTests := testData["GB"]
	for gameName, test := range gbTests {
		t.Run(gameName, func(t *testing.T) {
			// Check if game file exists
			if _, err := os.Stat(test.Path); err != nil {
				t.Skipf("Game file not found: %s", test.Path)
			}

			result, err := identifier.Identify(test.Path)
			if err != nil {
				t.Fatalf("Failed to identify game: %v", err)
			}

			// Check key fields
			checkField := func(field string) {
				if result[field] != test.Expected[field] {
					t.Errorf("%s mismatch: got %q, want %q", field, result[field], test.Expected[field])
				}
			}

			checkField("internal_title")
			checkField("cgb_mode")
			checkField("sgb_support")
			checkField("cartridge_type")
			checkField("rom_size")
			checkField("rom_banks")
			checkField("ram_size")
			checkField("ram_banks")
			checkField("licensee")
			checkField("rom_version")
			checkField("header_checksum_expected")
			checkField("header_checksum_actual")
			checkField("global_checksum_expected")
			checkField("global_checksum_actual")
			checkField("title")
			checkField("language")
			checkField("manufacturer_code")
			checkField("region")
		})
	}
}

// Test with synthetic data (for CI/CD without game files)
func TestGBIdentify_Synthetic(t *testing.T) {
	// Create a minimal test database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GB": {
				"TETRIS": database.GameMetadata{
					"ID":    "TETRIS",
					"title": "Tetris",
				},
			},
		},
	}

	identifier := &GBIdentifier{db: testDB}

	// Create test ROM header
	header := make([]byte, 0x150)

	// Nintendo logo at 0x104-0x133
	copy(header[0x104:], gbLogo)

	// Game title at 0x134-0x143
	copy(header[0x134:], []byte("TETRIS"))

	// CGB flag at 0x143 (GB mode)
	header[0x143] = 0x00

	// SGB flag at 0x146 (no SGB support)
	header[0x146] = 0x00

	// Cartridge type at 0x147 (MBC1)
	header[0x147] = 0x01

	// ROM size at 0x148 (64KB, 4 banks)
	header[0x148] = 0x01

	// RAM size at 0x149 (no RAM)
	header[0x149] = 0x00

	// Old licensee at 0x14B
	header[0x14B] = 0x01 // Nintendo

	// ROM version at 0x14C
	header[0x14C] = 0x00

	// Header checksum at 0x14D (will be calculated)
	expectedChecksum := calculateHeaderChecksum(header[0x134:0x14D])
	header[0x14D] = expectedChecksum

	// Global checksum at 0x14E-0x14F
	globalChecksum := calculateGlobalChecksum(header)
	header[0x14E] = byte(globalChecksum >> 8)
	header[0x14F] = byte(globalChecksum & 0xFF)

	// Write test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gb")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"internal_title":           "TETRIS",
		"cgb_mode":                 "GB",
		"sgb_support":              "False",
		"cartridge_type":           "MBC1",
		"rom_size":                 "65536",
		"rom_banks":                "4",
		"ram_size":                 "0",
		"ram_banks":                "0",
		"licensee":                 "Nintendo",
		"rom_version":              "0",
		"header_checksum_expected": "0x09",
		"header_checksum_actual":   "0x09",
		"title":                    "Tetris",
		"language":                 "N/A",
		"manufacturer_code":        "N/A",
		"region":                   "NTSC-U",
	}

	for field, want := range expected {
		if result[field] != want {
			t.Errorf("%s mismatch: got %q, want %q", field, result[field], want)
		}
	}
}

func TestGBIdentify_InvalidLogo(t *testing.T) {
	identifier := &GBIdentifier{}

	// Create header with invalid logo
	header := make([]byte, 0x150)
	// Don't copy the logo - leave it as zeros

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.gb")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should still work but might log a warning
	_, err := identifier.Identify(testFile)
	if err != nil {
		// It's okay if it returns an error for invalid logo
		t.Logf("Got expected error for invalid logo: %v", err)
	}
}

func TestGBIdentify_TruncatedFile(t *testing.T) {
	identifier := &GBIdentifier{}

	// Create truncated file
	header := make([]byte, 0x100) // Too short

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "truncated.gb")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for truncated file")
	}
}

func TestGBIdentify_GameNotInDB(t *testing.T) {
	// Empty database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GB": {},
		},
	}

	identifier := &GBIdentifier{db: testDB}

	// Create header for unknown game
	header := make([]byte, 0x150)
	copy(header[0x104:], gbLogo)
	copy(header[0x134:], []byte("UNKNOWN GAME"))

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unknown.gb")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Should return cleaned internal title as title when not found in DB
	if result["title"] == "" {
		t.Error("Expected title to be set for unknown game")
	}

	if result["internal_title"] != "UNKNOWN GAME" {
		t.Errorf("Expected internal_title 'UNKNOWN GAME', got %q", result["internal_title"])
	}
}
