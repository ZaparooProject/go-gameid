package identifier

import (
	"bytes"
	"testing"
)

// createGBAHeader creates a minimal valid GBA ROM header for testing.
func createGBAHeader(gameCode, internalTitle, makerCode string, version uint8) []byte {
	// GBA header is 192 bytes
	header := make([]byte, 0xC0)

	// Entry point (0x00-0x03)
	copy(header[0x00:], []byte{0x00, 0x00, 0x00, 0xEA})

	// Nintendo logo (0x04-0x9F) - 156 bytes
	copy(header[0x04:], gbaNintendoLogo)

	// Internal title (0xA0-0xAB) - 12 bytes
	if len(internalTitle) > 12 {
		internalTitle = internalTitle[:12]
	}
	copy(header[0xA0:], []byte(internalTitle))

	// Game code (0xAC-0xAF) - 4 bytes
	if len(gameCode) >= 4 {
		copy(header[0xAC:], []byte(gameCode[:4]))
	}

	// Maker code (0xB0-0xB1) - 2 bytes
	if len(makerCode) >= 2 {
		copy(header[0xB0:], []byte(makerCode[:2]))
	}

	// Fixed value (0xB2)
	header[0xB2] = 0x96

	// Main unit code (0xB3)
	header[0xB3] = 0x00

	// Device type (0xB4)
	header[0xB4] = 0x00

	// Reserved (0xB5-0xBB) - already zeros

	// Version (0xBC)
	header[0xBC] = version

	// Header checksum (0xBD) - simplified, real checksum calculation not needed for tests
	header[0xBD] = 0x00

	// Reserved (0xBE-0xBF) - already zeros

	return header
}

func TestGBAIdentifier_Identify(t *testing.T) {
	id := NewGBAIdentifier()

	tests := []struct {
		name          string
		gameCode      string
		internalTitle string
		makerCode     string
		version       uint8
		wantID        string
		wantInternal  string
	}{
		{
			name:          "Pokemon Emerald",
			gameCode:      "BPEE",
			internalTitle: "POKEMON EMER",
			makerCode:     "01",
			version:       0,
			wantID:        "BPEE",
			wantInternal:  "POKEMON EMER",
		},
		{
			name:          "Mario Kart",
			gameCode:      "AMKE",
			internalTitle: "MARIOKART",
			makerCode:     "01",
			version:       1,
			wantID:        "AMKE",
			wantInternal:  "MARIOKART",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createGBAHeader(tt.gameCode, tt.internalTitle, tt.makerCode, tt.version)
			r := bytes.NewReader(header)

			result, err := id.Identify(r, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}

			if result.InternalTitle != tt.wantInternal {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, tt.wantInternal)
			}

			if result.Console != ConsoleGBA {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleGBA)
			}
		})
	}
}

func TestGBAIdentifier_InvalidLogo(t *testing.T) {
	// GBA identifier doesn't fail on invalid logo, it just continues
	// But ValidateGBA should return false
	header := make([]byte, 0xC0)
	copy(header[0xAC:], []byte("TEST"))

	if ValidateGBA(header) {
		t.Error("ValidateGBA() should return false for invalid logo")
	}
}
