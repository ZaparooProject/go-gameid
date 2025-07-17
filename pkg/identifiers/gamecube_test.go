package identifiers

import (
	"os"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

func TestGameCubeIdentifier_Console(t *testing.T) {
	identifier := NewGameCubeIdentifier(nil)
	if identifier.Console() != "GC" {
		t.Errorf("Expected console 'GC', got %s", identifier.Console())
	}
}

func TestGameCubeIdentifier_Identify(t *testing.T) {
	tests := []struct {
		name        string
		isoData     []byte
		expected    map[string]string
		expectError bool
	}{
		{
			name: "Valid GameCube ISO",
			isoData: createGameCubeISO(
				"GALE",                    // ID
				"01",                      // Maker code
				0x00,                      // Disk ID
				0x00,                      // Version
				"Super Smash Bros. Melee", // Internal title
			),
			expected: map[string]string{
				"ID":             "GALE",
				"maker_code":     "01",
				"disk_ID":        "0",
				"version":        "0",
				"internal_title": "Super Smash Bros. Melee",
				"title":          "Super Smash Bros. Melee",
			},
		},
		{
			name: "GameCube with database match",
			isoData: createGameCubeISO(
				"GMSJ",
				"8P",
				0x01,
				0x02,
				"SUPER MARIO SUNSHINE",
			),
			expected: map[string]string{
				"ID":             "GMSJ",
				"maker_code":     "8P",
				"disk_ID":        "1",
				"version":        "2",
				"internal_title": "SUPER MARIO SUNSHINE",
				"title":          "Mario Sunshine Test",
				"developer":      "Nintendo",
			},
		},
		{
			name: "GameCube with special characters in title",
			isoData: createGameCubeISO(
				"GTKE",
				"52",
				0x00,
				0x00,
				"Tom Clancy's Splinter Cell™",
			),
			expected: map[string]string{
				"ID":             "GTKE",
				"maker_code":     "52",
				"disk_ID":        "0",
				"version":        "0",
				"internal_title": "Tom Clancy's Splinter Cell™",
				"title":          "Tom Clancy's Splinter Cell™",
			},
		},
		{
			name:    "Empty GameCube ISO",
			isoData: make([]byte, 0x440),
			expected: map[string]string{
				"ID":             "",
				"maker_code":     "",
				"disk_ID":        "0",
				"version":        "0",
				"internal_title": "",
				"title":          "",
			},
		},
		{
			name:        "Too small file",
			isoData:     make([]byte, 0x400), // Less than required 0x440 bytes
			expectError: true,
		},
	}

	// Create test database
	db := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"GC": {
				"GMSJ": {
					"title":     "Mario Sunshine Test",
					"developer": "Nintendo",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test data to a temporary file
			tmpFile := t.TempDir() + "/test.iso"
			if err := writeTestFile(tmpFile, tt.isoData); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			identifier := NewGameCubeIdentifier(db)
			result, err := identifier.Identify(tmpFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check all expected fields
			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("Field %s: expected '%s', got '%s'", key, expectedValue, result[key])
				}
			}
		})
	}
}

// Helper function to create a minimal GameCube ISO for testing
func createGameCubeISO(id string, makerCode string, diskID byte, version byte, title string) []byte {
	data := make([]byte, 0x440)

	// ID at 0x0000 (4 bytes)
	copy(data[0x0000:0x0004], []byte(id))

	// Maker code at 0x0004 (2 bytes)
	copy(data[0x0004:0x0006], []byte(makerCode))

	// Disk ID at 0x0006 (1 byte)
	data[0x0006] = diskID

	// Version at 0x0007 (1 byte)
	data[0x0007] = version

	// Internal title at 0x0020 (up to 0x3E0 bytes)
	titleBytes := []byte(title)
	if len(titleBytes) > 0x3E0 {
		titleBytes = titleBytes[:0x3E0]
	}
	copy(data[0x0020:], titleBytes)

	return data
}

// Helper function to write test data to a file
func writeTestFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
