package identifiers

import (
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

func TestSegaCDIdentifier_Console(t *testing.T) {
	identifier := NewSegaCDIdentifier(nil)
	if identifier.Console() != "SegaCD" {
		t.Errorf("Expected console 'SegaCD', got %s", identifier.Console())
	}
}

func TestSegaCDIdentifier_Identify(t *testing.T) {
	tests := []struct {
		name        string
		isoData     []byte
		expected    map[string]string
		expectError bool
	}{
		{
			name: "Valid SegaCD ISO",
			isoData: createSegaCDISO(
				0x00,                          // Magic word at offset 0
				"SEGADISCSYSTEM  ",            // Disc ID (includes magic word)
				"MK-4402    ",                 // Volume ID
				"SEGA-CD     ",                // System name
				"09151993",                    // Build date (MMDDYYYY)
				"SEGA MEGA-CD   ",             // System type
				"1993",                        // Release year
				"09 ",                         // Release month
				"SONIC THE HEDGEHOG CD      ", // Title domestic
				"SONIC THE HEDGEHOG CD      ", // Title overseas
				"GM MK-4402 -00",              // ID (disc_kind ID version)
				"J6              ",            // Device support
			),
			expected: map[string]string{
				"disc_ID":        "SEGADISCSYSTEM",
				"volume_ID":      "MK-4402",
				"system_name":    "SEGA-CD",
				"build_date":     "1993-09-15",
				"system_type":    "SEGA MEGA-CD",
				"release_year":   "1993",
				"release_month":  "09",
				"title_domestic": "SONIC THE HEDGEHOG CD",
				"title_overseas": "SONIC THE HEDGEHOG CD",
				"ID":             "MK-4402",
				"disc_kind":      "GM",
				"version":        "00",
				"device_support": "J6",
				"title":          "SONIC THE HEDGEHOG CD",
			},
		},
		{
			name: "SegaCD with different magic word",
			isoData: createSegaCDISO(
				0x10,                          // Magic word at offset 0x10
				"SEGABOOTDISC   ",             // Disc ID (includes magic word)
				"MK-4407    ",                 // Volume ID
				"SEGA-CD     ",                // System name
				"11201994",                    // Build date
				"SEGA MEGA-CD   ",             // System type
				"1994",                        // Release year
				"11 ",                         // Release month
				"LUNAR: ETERNAL BLUE        ", // Title domestic
				"LUNAR: ETERNAL BLUE        ", // Title overseas
				"GM MK-4407-00-01",            // ID (16 chars max)
				"J               ",            // Device support
			),
			expected: map[string]string{
				"disc_ID":        "SEGABOOTDISC",
				"volume_ID":      "MK-4407",
				"system_name":    "SEGA-CD",
				"build_date":     "1994-11-20",
				"system_type":    "SEGA MEGA-CD",
				"release_year":   "1994",
				"release_month":  "11",
				"title_domestic": "LUNAR: ETERNAL BLUE",
				"title_overseas": "LUNAR: ETERNAL BLUE",
				"ID":             "MK-4407-00",
				"disc_kind":      "GM",
				"version":        "01",
				"device_support": "J",
				"title":          "LUNAR: ETERNAL BLUE",
			},
		},
		{
			name: "SegaCD with database match",
			isoData: createSegaCDISO(
				0x00,
				"SEGADISCSYSTEM  ",
				"MK-4651    ",
				"SEGA-CD     ",
				"03151995",
				"SEGA MEGA-CD   ",
				"1995",
				"03 ",
				"ECCO THE DOLPHIN           ",
				"ECCO THE DOLPHIN           ",
				"GM MK-4651 -00",
				"J               ",
			),
			expected: map[string]string{
				"disc_ID":        "SEGADISCSYSTEM",
				"volume_ID":      "MK-4651",
				"system_name":    "SEGA-CD",
				"build_date":     "1995-03-15",
				"system_type":    "SEGA MEGA-CD",
				"release_year":   "1995",
				"release_month":  "03",
				"title_domestic": "ECCO THE DOLPHIN",
				"title_overseas": "ECCO THE DOLPHIN",
				"ID":             "MK-4651",
				"disc_kind":      "GM",
				"version":        "00",
				"device_support": "J",
				"title":          "Ecco Test Game",
				"developer":      "Novotrade",
			},
		},
		{
			name:        "No magic word found",
			isoData:     make([]byte, 0x300),
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
			"SegaCD": {
				"MK-4651": {
					"title":     "Ecco Test Game",
					"developer": "Novotrade",
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

			identifier := NewSegaCDIdentifier(db)
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

// Helper function to create a minimal SegaCD ISO for testing
func createSegaCDISO(magicOffset int, discID, volumeID, systemName, buildDate, systemType, releaseYear, releaseMonth, titleDomestic, titleOverseas, id, deviceSupport string) []byte {
	data := make([]byte, 0x300)

	// Disc ID at magic + 0x000 (16 bytes) - this includes the magic word!
	copy(data[magicOffset+0x000:], []byte(discID))

	// Volume ID at magic + 0x010 (11 bytes)
	copy(data[magicOffset+0x010:], []byte(volumeID))

	// System name at magic + 0x020 (11 bytes)
	copy(data[magicOffset+0x020:], []byte(systemName))

	// Build date at magic + 0x050 (8 bytes) - MMDDYYYY format
	copy(data[magicOffset+0x050:], []byte(buildDate))

	// System type at magic + 0x100 (16 bytes)
	copy(data[magicOffset+0x100:], []byte(systemType))

	// Release year at magic + 0x118 (4 bytes)
	copy(data[magicOffset+0x118:], []byte(releaseYear))

	// Release month at magic + 0x11D (3 bytes)
	copy(data[magicOffset+0x11D:], []byte(releaseMonth))

	// Title domestic at magic + 0x120 (48 bytes)
	titleDomBytes := []byte(titleDomestic)
	if len(titleDomBytes) > 48 {
		titleDomBytes = titleDomBytes[:48]
	}
	copy(data[magicOffset+0x120:], titleDomBytes)

	// Title overseas at magic + 0x150 (48 bytes)
	titleOvBytes := []byte(titleOverseas)
	if len(titleOvBytes) > 48 {
		titleOvBytes = titleOvBytes[:48]
	}
	copy(data[magicOffset+0x150:], titleOvBytes)

	// ID at magic + 0x180 (16 bytes)
	copy(data[magicOffset+0x180:], []byte(id))

	// Device support at magic + 0x190 (16 bytes)
	copy(data[magicOffset+0x190:], []byte(deviceSupport))

	return data
}
