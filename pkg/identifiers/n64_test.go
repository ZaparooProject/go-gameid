package identifiers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

// Test with real game files
func TestN64Identify_RealGames(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("..", "..", "test_data", "reference", "n64_test_data.json")
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

	identifier := &N64Identifier{db: db}

	// Test each game
	n64Tests := testData["N64"]
	for gameName, test := range n64Tests {
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
			checkField("title")
			checkField("developer")
			checkField("publisher")
			checkField("rating")
			checkField("region")
			checkField("release_date")
		})
	}
}

// Test with synthetic data (for CI/CD without game files)
func TestN64Identify_Synthetic(t *testing.T) {
	// Create a minimal test database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"N64": {
				"NTE": database.GameMetadata{
					"ID":           "NTE",
					"title":        "Test N64 Game",
					"developer":    "Test Developer",
					"publisher":    "Test Publisher",
					"rating":       "E / 3+",
					"region":       "NTSC-U",
					"release_date": "1996-01-01",
				},
			},
		},
	}

	identifier := &N64Identifier{db: testDB}

	// Create test ROM header (64 bytes)
	header := make([]byte, 0x40)

	// N64 magic number (big-endian)
	header[0] = 0x80
	header[1] = 0x37
	header[2] = 0x12
	header[3] = 0x40

	// Internal title at 0x20-0x33
	copy(header[0x20:], []byte("Test N64 Game   "))

	// Cartridge ID at 0x3C-0x3D
	header[0x3C] = 'N' // Game ID char 1
	header[0x3D] = 'T' // Game ID char 2

	// Country code at 0x3E
	header[0x3E] = 'E' // Country code (USA)

	// Version at 0x3F
	header[0x3F] = 0x00

	// Write test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.z64")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"ID":           "NTE",
		"title":        "Test N64 Game",
		"developer":    "Test Developer",
		"publisher":    "Test Publisher",
		"rating":       "E / 3+",
		"region":       "NTSC-U",
		"release_date": "1996-01-01",
	}

	for field, want := range expected {
		if result[field] != want {
			t.Errorf("%s mismatch: got %q, want %q", field, result[field], want)
		}
	}
}

func TestN64Identify_InvalidMagic(t *testing.T) {
	identifier := &N64Identifier{}

	// Create header with invalid magic
	header := make([]byte, 0x40)
	// Don't set the magic number - leave as zeros

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.z64")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for invalid magic number")
	}
}

func TestN64Identify_TruncatedFile(t *testing.T) {
	identifier := &N64Identifier{}

	// Create truncated file
	header := make([]byte, 0x20) // Too short

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "truncated.z64")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for truncated file")
	}
}

func TestN64Identify_GameNotInDB(t *testing.T) {
	// Empty database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"N64": {},
		},
	}

	identifier := &N64Identifier{db: testDB}

	// Create header for unknown game
	header := make([]byte, 0x40)
	header[0] = 0x80
	header[1] = 0x37
	header[2] = 0x12
	header[3] = 0x40
	copy(header[0x20:], []byte("Unknown Game    "))
	header[0x3C] = 'X'
	header[0x3D] = 'X'
	header[0x3E] = 'E'

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unknown.z64")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := identifier.Identify(testFile)
	if err == nil {
		t.Error("Expected error for unknown game not in database")
	}
}

func TestN64Identify_LittleEndian(t *testing.T) {
	// Create a minimal test database
	testDB := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"N64": {
				"NTE": database.GameMetadata{
					"ID":    "NTE",
					"title": "Test N64 Game",
				},
			},
		},
	}

	identifier := &N64Identifier{db: testDB}

	// Create test ROM header in little-endian format
	header := make([]byte, 0x40)

	// N64 magic number (little-endian - swapped bytes)
	header[0] = 0x37
	header[1] = 0x80
	header[2] = 0x40
	header[3] = 0x12

	// Internal title at 0x20-0x33 (will be byte-swapped)
	titleBytes := []byte("Test N64 Game   ")
	for i := 0; i < len(titleBytes); i += 2 {
		if i+1 < len(titleBytes) {
			header[0x20+i] = titleBytes[i+1]
			header[0x20+i+1] = titleBytes[i]
		} else {
			header[0x20+i] = titleBytes[i]
		}
	}

	// Cartridge ID at 0x3C-0x3D (swapped)
	header[0x3C] = 'T' // Will become 'N' after swap
	header[0x3D] = 'N' // Will become 'T' after swap

	// Country code at 0x3E (swapped with next byte)
	header[0x3E] = 0x00 // Version (will be swapped)
	header[0x3F] = 'E'  // Country code (will be swapped)

	// Write test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "little_endian.v64")
	if err := os.WriteFile(testFile, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := identifier.Identify(testFile)
	if err != nil {
		t.Fatalf("Failed to identify game: %v", err)
	}

	// Should correctly identify the game after endian conversion
	if result["ID"] != "NTE" {
		t.Errorf("Expected ID 'NTE', got %q", result["ID"])
	}
}
