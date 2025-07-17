package identifiers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

// Test with real game files
func TestSNESIdentify_RealGames(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("..", "..", "test_data", "reference", "snes_test_data.json")
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

	identifier := &SNESIdentifier{db: db}

	// Test each game
	snesTests := testData["SNES"]
	for gameName, test := range snesTests {
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
			checkField("fast_slow_rom")
			checkField("rom_type")
			checkField("developer_ID")
			checkField("rom_version")
			checkField("checksum")
			checkField("hardware")
			checkField("title")
		})
	}
}

// Test with synthetic data (for CI/CD without game files)
func TestSNESIdentify_Synthetic(t *testing.T) {
	// Create a minimal test database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"SNES": {
				"1,0x54455354205346432047414d452020202020202020,0,255": database.GameMetadata{
					"internal_title": "0x54455354205346432047414d452020202020202020",
					"title":          "Test SNES Game",
					"developer":      "Test Developer",
					"publisher":      "Test Publisher",
					"rating":         "E / 3+",
					"region":         "NTSC-U",
					"release_date":   "1990-11-21",
				},
			},
		},
	}

	identifier := &SNESIdentifier{db: testDB}

	// Create test ROM header
	header := make([]byte, 0x8000) // 32KB minimum

	// SNES header starts at different locations based on ROM type
	// For LoROM, header is at 0x7FC0-0x7FFF
	// For HiROM, header is at 0xFFC0-0xFFFF
	// We'll test LoROM format

	headerOffset := 0x7FC0

	// Internal title at header+0x00 to header+0x14 (21 bytes)
	copy(header[headerOffset:], []byte("TEST SFC GAME        ")) // 21 bytes exactly

	// ROM makeup at header+0x15
	header[headerOffset+0x15] = 0x20 // LoROM, slow ROM

	// Cartridge type at header+0x16
	header[headerOffset+0x16] = 0x00 // ROM only

	// ROM size at header+0x17
	header[headerOffset+0x17] = 0x08 // 256KB

	// RAM size at header+0x18
	header[headerOffset+0x18] = 0x00 // No RAM

	// Country code at header+0x19
	header[headerOffset+0x19] = 0x01 // USA

	// Developer ID at header+0x1A
	header[headerOffset+0x1A] = 0x01 // Nintendo

	// Version at header+0x1B
	header[headerOffset+0x1B] = 0x00

	// Checksum complement at header+0x1C-0x1D
	header[headerOffset+0x1C] = 0x00
	header[headerOffset+0x1D] = 0xFF

	// Checksum at header+0x1E-0x1F
	header[headerOffset+0x1E] = 0xFF
	header[headerOffset+0x1F] = 0x00

	// Write test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.sfc")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"internal_title": "0x54455354205346432047414d452020202020202020",
		"fast_slow_rom":  "SlowROM",
		"rom_type":       "LoROM",
		"developer_ID":   "0x01",
		"rom_version":    "0",
		"checksum":       "0xff00",
		"hardware":       "ROM",
		"title":          "Test SNES Game",
	}

	for field, want := range expected {
		if result[field] != want {
			t.Errorf("%s mismatch: got %q, want %q", field, result[field], want)
		}
	}
}

func TestSNESIdentify_TruncatedFile(t *testing.T) {
	identifier := &SNESIdentifier{}

	// Create truncated file
	header := make([]byte, 0x100) // Too short for SNES ROM

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "truncated.sfc")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for truncated file")
	}
}

func TestSNESIdentify_HiROM(t *testing.T) {
	identifier := &SNESIdentifier{}

	// Create test HiROM file
	header := make([]byte, 0x10000) // 64KB minimum for HiROM

	// HiROM header is at 0xFFC0-0xFFFF
	headerOffset := 0xFFC0

	// Internal title
	copy(header[headerOffset:], []byte("HIROM TEST GAME     "))

	// ROM makeup - HiROM, fast ROM
	header[headerOffset+0x15] = 0x31 // HiROM (bit 0x01), fast ROM (bit 0x10)

	// Cartridge type
	header[headerOffset+0x16] = 0x00 // ROM only

	// ROM size
	header[headerOffset+0x17] = 0x09 // 512KB

	// RAM size
	header[headerOffset+0x18] = 0x00 // No RAM

	// Country code
	header[headerOffset+0x19] = 0x01 // USA

	// Developer ID
	header[headerOffset+0x1A] = 0x01 // Nintendo

	// Version
	header[headerOffset+0x1B] = 0x00

	// Checksum complement
	header[headerOffset+0x1C] = 0x00
	header[headerOffset+0x1D] = 0xFF

	// Checksum
	header[headerOffset+0x1E] = 0xFF
	header[headerOffset+0x1F] = 0x00

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "hirom.sfc")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify HiROM game: %v", err)
	}

	if result["rom_type"] != "HiROM" {
		t.Errorf("Expected HiROM, got %q", result["rom_type"])
	}

	if result["fast_slow_rom"] != "FastROM" {
		t.Errorf("Expected FastROM, got %q", result["fast_slow_rom"])
	}
}

func TestSNESIdentify_GameNotInDB(t *testing.T) {
	// Empty database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"SNES": {},
		},
	}

	identifier := &SNESIdentifier{db: testDB}

	// Create header for unknown game
	header := make([]byte, 0x8000)
	headerOffset := 0x7FC0

	copy(header[headerOffset:], []byte("UNKNOWN GAME        "))
	header[headerOffset+0x15] = 0x20 // LoROM, slow ROM
	header[headerOffset+0x16] = 0x00 // ROM only

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unknown.sfc")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify unknown game: %v", err)
	}

	// Should still return basic metadata even if not in database
	if result["internal_title"] == "" {
		t.Error("Expected internal_title even for unknown game")
	}

	if result["rom_type"] != "LoROM" {
		t.Errorf("Expected LoROM, got %q", result["rom_type"])
	}
}
