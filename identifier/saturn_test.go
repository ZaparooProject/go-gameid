package identifier

import (
	"bytes"
	"strings"
	"testing"
)

// createSaturnHeader creates a minimal valid Saturn disc header for testing.
func createSaturnHeader(manufacturerID, gameID, version, internalTitle string) []byte {
	// Saturn header needs at least 0x100 bytes
	header := make([]byte, 0x100)

	// Magic word at offset 0x00
	copy(header[0x00:], saturnMagicWord)

	// Manufacturer ID at offset 0x10 (16 bytes)
	if len(manufacturerID) > 16 {
		manufacturerID = manufacturerID[:16]
	}
	copy(header[0x10:], []byte(manufacturerID))

	// Game ID at offset 0x20 (10 bytes)
	if len(gameID) > 10 {
		gameID = gameID[:10]
	}
	copy(header[0x20:], []byte(gameID))

	// Version at offset 0x2A (6 bytes)
	if len(version) > 6 {
		version = version[:6]
	}
	copy(header[0x2A:], []byte(version))

	// Release date at offset 0x30 (8 bytes YYYYMMDD)
	copy(header[0x30:], []byte("19961122"))

	// Device info at offset 0x38 (8 bytes)
	copy(header[0x38:], []byte("CD-1/1  "))

	// Target area at offset 0x40 (16 bytes)
	copy(header[0x40:], []byte("JUE             "))

	// Device support at offset 0x50 (16 bytes)
	copy(header[0x50:], []byte("J               "))

	// Internal title at offset 0x60 (112 bytes)
	if len(internalTitle) > 112 {
		internalTitle = internalTitle[:112]
	}
	copy(header[0x60:], []byte(internalTitle))

	return header
}

func TestSaturnIdentifier_Identify(t *testing.T) {
	id := NewSaturnIdentifier()

	tests := []struct {
		name           string
		manufacturerID string
		gameID         string
		version        string
		internalTitle  string
		wantID         string
	}{
		{
			name:           "Nights into Dreams",
			manufacturerID: "SEGA ENTERPRISES",
			gameID:         "GS-9046  ",
			version:        "V1.000",
			internalTitle:  "NiGHTS into Dreams...",
			wantID:         "GS-9046",
		},
		{
			name:           "Virtua Fighter 2",
			manufacturerID: "SEGA ENTERPRISES",
			gameID:         "GS-9001  ",
			version:        "V1.001",
			internalTitle:  "Virtua Fighter 2",
			wantID:         "GS-9001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createSaturnHeader(tt.manufacturerID, tt.gameID, tt.version, tt.internalTitle)
			r := bytes.NewReader(header)

			result, err := id.Identify(r, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}

			// InternalTitle includes null padding - just check it starts with expected title
			if !strings.HasPrefix(result.InternalTitle, tt.internalTitle) {
				t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, tt.internalTitle)
			}

			if result.Console != ConsoleSaturn {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSaturn)
			}

			// Check metadata
			if result.Metadata["manufacturer_ID"] != tt.manufacturerID {
				t.Errorf("manufacturer_ID = %q, want %q", result.Metadata["manufacturer_ID"], tt.manufacturerID)
			}
		})
	}
}

func TestSaturnIdentifier_InvalidMagic(t *testing.T) {
	id := NewSaturnIdentifier()

	// Create header without magic word
	header := make([]byte, 0x100)
	copy(header[0x00:], []byte("NOT A SATURN"))

	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestSaturnIdentifier_TooSmall(t *testing.T) {
	id := NewSaturnIdentifier()

	header := make([]byte, 0x50) // Too small
	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestSaturnIdentifier_DeviceSupport(t *testing.T) {
	id := NewSaturnIdentifier()

	header := createSaturnHeader("SEGA", "TEST-001", "V1.000", "Test Game")
	// Add more device support codes
	copy(header[0x50:], []byte("JMG             "))

	r := bytes.NewReader(header)

	result, err := id.Identify(r, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	deviceSupport := result.Metadata["device_support"]
	if deviceSupport == "" {
		t.Error("device_support should not be empty")
	}
}

func TestSaturnIdentifier_TargetArea(t *testing.T) {
	id := NewSaturnIdentifier()

	header := createSaturnHeader("SEGA", "TEST-001", "V1.000", "Test Game")
	// Set target area to Japan/US/Europe
	copy(header[0x40:], []byte("JUE             "))

	r := bytes.NewReader(header)

	result, err := id.Identify(r, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	targetArea := result.Metadata["target_area"]
	if targetArea == "" {
		t.Error("target_area should not be empty")
	}
}

func TestValidateSaturn(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid Saturn",
			header: createSaturnHeader("SEGA", "TEST", "V1.000", "Test"),
			want:   true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, 0x100),
			want:   false,
		},
		{
			name:   "Invalid - wrong magic",
			header: append([]byte("SEGA GENESIS    "), make([]byte, 0xF0)...),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSaturn(tt.header)
			if got != tt.want {
				t.Errorf("ValidateSaturn() = %v, want %v", got, tt.want)
			}
		})
	}
}
