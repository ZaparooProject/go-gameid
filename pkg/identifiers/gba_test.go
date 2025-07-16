package identifiers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

// Test with real game files
func TestGBAIdentify_RealGames(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("..", "..", "test_data", "reference", "gba_test_data.json")
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

	identifier := &GBAIdentifier{db: db}

	// Test each game
	gbaTests := testData["GBA"]
	for gameName, test := range gbaTests {
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

			checkField("ID")
			checkField("internal_title")
			checkField("maker_code")
			checkField("main_unit_code")
			checkField("device_type")
			checkField("software_version")
			checkField("title")
		})
	}
}

// Test with synthetic data (for CI/CD without game files)
func TestGBAIdentify_Synthetic(t *testing.T) {
	// Create a minimal test database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GBA": {
				"AGSE": database.GameMetadata{
					"ID":    "AGSE",
					"title": "Golden Sun",
				},
			},
		},
	}

	identifier := &GBAIdentifier{db: testDB}

	// Create test ROM header
	header := make([]byte, 192)

	// Nintendo logo at 0x04-0x9F
	copy(header[0x04:], gbaLogo)

	// Game title at 0xA0-0xAB
	copy(header[0xA0:], []byte("Golden_Sun_A"))

	// Game code at 0xAC-0xAF
	copy(header[0xAC:], []byte("AGSE"))

	// Maker code at 0xB0-0xB1
	copy(header[0xB0:], []byte("01"))

	// Other fields
	header[0xB3] = 0x00 // Main unit code
	header[0xB4] = 0x00 // Device type
	header[0xBC] = 0x00 // Software version

	// Write test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gba")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"ID":               "AGSE",
		"internal_title":   "Golden_Sun_A",
		"maker_code":       "01",
		"main_unit_code":   "0x00",
		"device_type":      "0x00",
		"software_version": "0",
		"title":            "Golden Sun",
	}

	for field, want := range expected {
		if result[field] != want {
			t.Errorf("%s mismatch: got %q, want %q", field, result[field], want)
		}
	}
}

func TestGBAIdentify_InvalidLogo(t *testing.T) {
	identifier := &GBAIdentifier{}

	// Create header with invalid logo
	header := make([]byte, 192)
	// Don't copy the logo - leave it as zeros

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.gba")
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

func TestGBAIdentify_TruncatedFile(t *testing.T) {
	identifier := &GBAIdentifier{}

	// Create truncated file
	header := make([]byte, 100) // Too short

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "truncated.gba")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for truncated file")
	}
}

func TestGBAIdentify_GameNotInDB(t *testing.T) {
	// Empty database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GBA": {},
		},
	}

	identifier := &GBAIdentifier{db: testDB}

	// Create header for unknown game
	header := make([]byte, 192)
	copy(header[0x04:], gbaLogo)
	copy(header[0xA0:], []byte("UNKNOWN GAME"))
	copy(header[0xAC:], []byte("XXXX"))
	copy(header[0xB0:], []byte("99"))

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unknown.gba")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Should return internal title as title when not found in DB
	if result["title"] != result["internal_title"] {
		t.Error("Expected title to match internal_title for unknown game")
	}

	if result["ID"] != "XXXX" {
		t.Errorf("Expected ID 'XXXX', got %q", result["ID"])
	}
}
