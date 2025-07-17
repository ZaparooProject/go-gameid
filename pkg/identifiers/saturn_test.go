package identifiers

import (
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

func TestSaturnIdentifier_Console(t *testing.T) {
	identifier := NewSaturnIdentifier(nil)
	if identifier.Console() != "Saturn" {
		t.Errorf("Expected console 'Saturn', got %s", identifier.Console())
	}
}

func TestSaturnIdentifier_Identify(t *testing.T) {
	tests := []struct {
		name        string
		isoData     []byte
		expected    map[string]string
		expectError bool
	}{
		{
			name: "Valid Saturn ISO",
			isoData: createSaturnISO(
				0x00,                       // Magic word at offset 0
				"SEGA ENTERPRISES",         // Manufacturer ID
				"MK-81036     ",            // Product ID
				"V1.002",                   // Version
				"19950915",                 // Release date
				"CDEFGJM ",                 // Device info (with trailing space)
				"J         ",               // Device support (Joypad)
				"JU        ",               // Target area (Japan/USA)
				"Virtua Fighter Remix    ", // Internal title
			),
			expected: map[string]string{
				"ID":              "MK-81036",
				"manufacturer_ID": "SEGA ENTERPRISES",
				"version":         "V1.002",
				"device_info":     "CDEFGJM",
				"release_date":    "1995-09-15",
				"device_support":  "Joypad",
				"target_area":     "Japan / North America (USA, Canada)",
				"internal_title":  "Virtua Fighter Remix",
				"title":           "Virtua Fighter Remix",
			},
		},
		{
			name: "Saturn with database match",
			isoData: createSaturnISO(
				0x10, // Magic word at offset 0x10
				"SEGA ENTERPRISES",
				"GS-9001      ",
				"V1.000",
				"19941122",
				"CD-1/1  ",
				"JE        ",
				"JT        ",
				"PANZER DRAGOON",
			),
			expected: map[string]string{
				"ID":              "GS-9001",
				"manufacturer_ID": "SEGA ENTERPRISES",
				"version":         "V1.000",
				"device_info":     "CD-1/1",
				"release_date":    "1994-11-22",
				"device_support":  "Joypad / Analog Controller (3D-pad)",
				"target_area":     "Japan / Asia NTSC (Taiwan, Philippines)",
				"internal_title":  "PANZER DRAGOON",
				"title":           "Panzer Dragoon Test",
				"developer":       "Team Andromeda",
			},
		},
		{
			name: "Saturn with multiple device support",
			isoData: createSaturnISO(
				0x00,
				"SEGA ENTERPRISES",
				"MK-81071     ",
				"V1.003",
				"19960628",
				"CD-1/1  ",
				"JGMW      ",
				"JUE       ",
				"VIRTUA COP",
			),
			expected: map[string]string{
				"ID":              "MK-81071",
				"manufacturer_ID": "SEGA ENTERPRISES",
				"version":         "V1.003",
				"device_info":     "CD-1/1",
				"release_date":    "1996-06-28",
				"device_support":  "Joypad / Gun / Mouse / RAM Cart",
				"target_area":     "Japan / North America (USA, Canada) / Europe PAL",
				"internal_title":  "VIRTUA COP",
				"title":           "VIRTUA COP",
			},
		},
		{
			name:        "No magic word found",
			isoData:     make([]byte, 0x100),
			expectError: true,
		},
		{
			name:        "Too small file",
			isoData:     []byte("small"),
			expectError: true,
		},
	}

	// Create test database
	db := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"Saturn": {
				"GS9001": { // Note: dashes and spaces removed from serial
					"title":     "Panzer Dragoon Test",
					"developer": "Team Andromeda",
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

			identifier := NewSaturnIdentifier(db)
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

// Helper function to create a minimal Saturn ISO for testing
func createSaturnISO(magicOffset int, mfgID, productID, version, releaseDate, deviceInfo, deviceSupport, targetArea, title string) []byte {
	data := make([]byte, 0x100)

	// Magic word "SEGA SEGASATURN" at specified offset
	magicWord := "SEGA SEGASATURN"
	copy(data[magicOffset:], []byte(magicWord))

	// Manufacturer ID at magic + 0x10 (16 bytes)
	copy(data[magicOffset+0x10:], []byte(mfgID))

	// Product ID at magic + 0x20 (10 bytes)
	copy(data[magicOffset+0x20:], []byte(productID))

	// Version at magic + 0x2A (6 bytes)
	copy(data[magicOffset+0x2A:], []byte(version))

	// Release date at magic + 0x30 (8 bytes)
	copy(data[magicOffset+0x30:], []byte(releaseDate))

	// Device info at magic + 0x38 (8 bytes)
	copy(data[magicOffset+0x38:], []byte(deviceInfo))

	// Target area at magic + 0x40 (16 bytes)
	copy(data[magicOffset+0x40:], []byte(targetArea))

	// Device support at magic + 0x50 (16 bytes)
	copy(data[magicOffset+0x50:], []byte(deviceSupport))

	// Internal title at magic + 0x60 (112 bytes, up to 0xD0)
	titleBytes := []byte(title)
	if len(titleBytes) > 112 {
		titleBytes = titleBytes[:112]
	}
	copy(data[magicOffset+0x60:], titleBytes)

	return data
}
