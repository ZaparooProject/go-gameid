package identifier

import (
	"bytes"
	"testing"
)

// createGCHeader creates a minimal valid GameCube disc header for testing.
func createGCHeader(gameID, makerCode, internalTitle string, diskID, version byte) []byte {
	// GameCube header is 0x0440 bytes
	header := make([]byte, gcHeaderSize)

	// Game ID (4 bytes at 0x0000)
	if len(gameID) >= 4 {
		copy(header[gcGameIDOffset:], []byte(gameID[:4]))
	}

	// Maker code (2 bytes at 0x0004)
	if len(makerCode) >= 2 {
		copy(header[gcMakerCodeOffset:], []byte(makerCode[:2]))
	}

	// Disk ID (1 byte at 0x0006)
	header[gcDiskIDOffset] = diskID

	// Version (1 byte at 0x0007)
	header[gcVersionOffset] = version

	// Magic word (4 bytes at 0x001C)
	copy(header[0x1C:], gcMagicWord)

	// Internal title (at 0x0020, up to 0x03E0 bytes)
	titleBytes := []byte(internalTitle)
	if len(titleBytes) > gcInternalNameSize {
		titleBytes = titleBytes[:gcInternalNameSize]
	}
	copy(header[gcInternalNameOffset:], titleBytes)

	return header
}

func TestGCIdentifier_Identify(t *testing.T) {
	id := NewGCIdentifier()

	tests := []struct {
		name          string
		gameID        string
		makerCode     string
		internalTitle string
		diskID        byte
		version       byte
		wantID        string
		wantTitle     string
	}{
		{
			name:          "Super Smash Bros Melee",
			gameID:        "GALE",
			makerCode:     "01",
			internalTitle: "Super Smash Bros. Melee",
			diskID:        0,
			version:       2,
			wantID:        "GALE",
			wantTitle:     "Super Smash Bros. Melee",
		},
		{
			name:          "Wind Waker",
			gameID:        "GZLE",
			makerCode:     "01",
			internalTitle: "The Legend of Zelda: The Wind Waker",
			diskID:        0,
			version:       0,
			wantID:        "GZLE",
			wantTitle:     "The Legend of Zelda: The Wind Waker",
		},
		{
			name:          "Multi-disc game",
			gameID:        "GXXE",
			makerCode:     "08",
			internalTitle: "Multi Disc Game",
			diskID:        1,
			version:       1,
			wantID:        "GXXE",
			wantTitle:     "Multi Disc Game",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createGCHeader(tt.gameID, tt.makerCode, tt.internalTitle, tt.diskID, tt.version)
			r := bytes.NewReader(header)

			result, err := id.Identify(r, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}

			if result.InternalTitle != tt.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, tt.wantTitle)
			}

			if result.Console != ConsoleGC {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleGC)
			}

			// Check metadata
			if result.Metadata["maker_code"] != tt.makerCode {
				t.Errorf("maker_code = %q, want %q", result.Metadata["maker_code"], tt.makerCode)
			}
		})
	}
}

func TestGCIdentifier_InvalidMagic(t *testing.T) {
	id := NewGCIdentifier()

	// Create header without magic word
	header := make([]byte, gcHeaderSize)
	copy(header[gcGameIDOffset:], []byte("GALE"))
	// Don't set magic word

	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestGCIdentifier_TooSmall(t *testing.T) {
	id := NewGCIdentifier()

	header := make([]byte, 0x100) // Too small
	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestValidateGC(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid GameCube",
			header: createGCHeader("GALE", "01", "Test Game", 0, 0),
			want:   true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, gcHeaderSize),
			want:   false,
		},
		{
			name:   "Too small",
			header: make([]byte, 0x10),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateGC(tt.header)
			if got != tt.want {
				t.Errorf("ValidateGC() = %v, want %v", got, tt.want)
			}
		})
	}
}
