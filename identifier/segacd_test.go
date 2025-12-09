package identifier

import (
	"bytes"
	"strings"
	"testing"
)

// createSegaCDHeader creates a minimal valid Sega CD disc header for testing.
// SegaCD header layout (offsets relative to magic word at 0x00):
// 0x000: Disc ID (16 bytes) - contains magic word like "SEGADISCSYSTEM"
// 0x010: Disc volume name (11 bytes)
// 0x020: System name (11 bytes)
// 0x050: Build date (8 bytes, MMDDYYYY)
// 0x100: System type (16 bytes)
// 0x118: Release year (4 bytes)
// 0x11D: Release month (3 bytes)
// 0x120: Title domestic (48 bytes)
// 0x150: Title overseas (48 bytes)
// 0x180: Game ID (16 bytes)
// 0x190: Device support (16 bytes)
// 0x1F0: Region support (3 bytes)
func createSegaCDHeader(discID, discVolumeName, systemName, titleDomestic, titleOverseas, gameID string) []byte {
	// Sega CD header needs at least 0x300 bytes
	header := make([]byte, 0x300)

	// Magic word / Disc ID at offset 0x00 (16 bytes)
	copy(header[0x00:], []byte("SEGADISCSYSTEM  "))

	// Disc volume name at offset 0x10 (11 bytes)
	if len(discVolumeName) > 11 {
		discVolumeName = discVolumeName[:11]
	}
	copy(header[0x10:], []byte(discVolumeName))

	// System name at offset 0x20 (11 bytes)
	if len(systemName) > 11 {
		systemName = systemName[:11]
	}
	copy(header[0x20:], []byte(systemName))

	// Build date at offset 0x50 (8 bytes, MMDDYYYY)
	copy(header[0x50:], []byte("08011993"))

	// System type at offset 0x100 (16 bytes)
	copy(header[0x100:], []byte("SEGA MEGA CD    "))

	// Release year at offset 0x118 (4 bytes)
	copy(header[0x118:], []byte("1993"))

	// Release month at offset 0x11D (3 bytes)
	copy(header[0x11D:], []byte("AUG"))

	// Title domestic at offset 0x120 (48 bytes)
	if len(titleDomestic) > 48 {
		titleDomestic = titleDomestic[:48]
	}
	copy(header[0x120:], []byte(titleDomestic))

	// Title overseas at offset 0x150 (48 bytes)
	if len(titleOverseas) > 48 {
		titleOverseas = titleOverseas[:48]
	}
	copy(header[0x150:], []byte(titleOverseas))

	// Game ID at offset 0x180 (16 bytes)
	if len(gameID) > 16 {
		gameID = gameID[:16]
	}
	copy(header[0x180:], []byte(gameID))

	// Device support at offset 0x190 (16 bytes)
	copy(header[0x190:], []byte("J               "))

	// Region support at offset 0x1F0 (3 bytes)
	copy(header[0x1F0:], []byte("JUE"))

	return header
}

func TestSegaCDIdentifier_Identify(t *testing.T) {
	id := NewSegaCDIdentifier()

	tests := []struct {
		name           string
		discVolumeName string
		systemName     string
		titleDomestic  string
		titleOverseas  string
		gameID         string
		wantID         string
		wantTitle      string
	}{
		{
			name:           "Sonic CD",
			discVolumeName: "SONICCD    ",
			systemName:     "SEGA ",
			titleDomestic:  "SONIC CD",
			titleOverseas:  "Sonic the Hedgehog CD",
			gameID:         "G-6014     ",
			wantID:         "G-6014",
			wantTitle:      "Sonic the Hedgehog CD",
		},
		{
			name:           "Lunar",
			discVolumeName: "LUNAR      ",
			systemName:     "SEGA ",
			titleDomestic:  "LUNAR",
			titleOverseas:  "Lunar: The Silver Star",
			gameID:         "T-127015   ",
			wantID:         "T-127015",
			wantTitle:      "Lunar: The Silver Star",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createSegaCDHeader("", tt.discVolumeName, tt.systemName, tt.titleDomestic, tt.titleOverseas, tt.gameID)
			r := bytes.NewReader(header)

			result, err := id.Identify(r, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			// ID may include null padding - just check it starts with expected ID
			if !strings.HasPrefix(result.ID, tt.wantID) {
				t.Errorf("ID = %q, want prefix %q", result.ID, tt.wantID)
			}

			// InternalTitle includes null padding - just check it starts with expected title
			if !strings.HasPrefix(result.InternalTitle, tt.wantTitle) {
				t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, tt.wantTitle)
			}

			if result.Console != ConsoleSegaCD {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSegaCD)
			}
		})
	}
}

func TestSegaCDIdentifier_DifferentMagicWords(t *testing.T) {
	id := NewSegaCDIdentifier()

	magicWords := []string{
		"SEGADISCSYSTEM  ",
		"SEGABOOTDISC    ",
		"SEGADISC        ",
		"SEGADATADISC    ",
	}

	for _, magic := range magicWords {
		t.Run(magic, func(t *testing.T) {
			header := make([]byte, 0x300)
			copy(header[0x00:], []byte(magic))
			copy(header[0x78:], []byte("Test Game"))
			copy(header[0xA8:], []byte("TEST-001"))

			r := bytes.NewReader(header)

			result, err := id.Identify(r, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() with magic %q error = %v", magic, err)
			}

			if result.Console != ConsoleSegaCD {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSegaCD)
			}
		})
	}
}

func TestSegaCDIdentifier_InvalidMagic(t *testing.T) {
	id := NewSegaCDIdentifier()

	header := make([]byte, 0x300)
	copy(header[0x00:], []byte("NOT A SEGA CD"))

	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestSegaCDIdentifier_TooSmall(t *testing.T) {
	id := NewSegaCDIdentifier()

	header := make([]byte, 0x100) // Too small
	r := bytes.NewReader(header)

	_, err := id.Identify(r, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestSegaCDIdentifier_OverseasTitlePreferred(t *testing.T) {
	id := NewSegaCDIdentifier()

	// Create header with both titles - overseas should be preferred
	header := createSegaCDHeader("", "TEST", "SEGA", "DOMESTIC TITLE", "OVERSEAS TITLE", "TEST-001")
	r := bytes.NewReader(header)

	result, err := id.Identify(r, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	// Should use overseas title (may have null padding)
	if !strings.HasPrefix(result.InternalTitle, "OVERSEAS TITLE") {
		t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, "OVERSEAS TITLE")
	}
}

func TestValidateSegaCD(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid - SEGADISCSYSTEM",
			header: createSegaCDHeader("", "TEST", "SEGA", "Test", "Test", "TEST-001"),
			want:   true,
		},
		{
			name: "Valid - SEGABOOTDISC",
			header: func() []byte {
				h := make([]byte, 0x300)
				copy(h[0x00:], []byte("SEGABOOTDISC"))
				return h
			}(),
			want: true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, 0x300),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSegaCD(tt.header)
			if got != tt.want {
				t.Errorf("ValidateSegaCD() = %v, want %v", got, tt.want)
			}
		})
	}
}
