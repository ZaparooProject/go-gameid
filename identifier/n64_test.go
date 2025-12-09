package identifier

import (
	"bytes"
	"testing"
)

// createN64Header creates a minimal valid N64 ROM header for testing.
// The N64 header layout:
// 0x00-0x03: First word (magic)
// 0x20-0x33: Internal name (20 bytes)
// 0x3C-0x3D: Cartridge ID (2 bytes)
// 0x3E: Country code
// 0x3F: Version
func createN64HeaderBigEndian(cartID, countryCode, title string) []byte {
	header := make([]byte, 0x40)

	// First word magic (big-endian .z64 format)
	header[0] = 0x80
	header[1] = 0x37
	header[2] = 0x12
	header[3] = 0x40

	// Title at 0x20 (20 bytes)
	titleBytes := []byte(title)
	if len(titleBytes) > 20 {
		titleBytes = titleBytes[:20]
	}
	for i := 0; i < 20; i++ {
		if i < len(titleBytes) {
			header[0x20+i] = titleBytes[i]
		} else {
			header[0x20+i] = ' '
		}
	}

	// Cartridge ID at 0x3C (2 bytes)
	if len(cartID) >= 2 {
		header[0x3C] = cartID[0]
		header[0x3D] = cartID[1]
	}

	// Country code at 0x3E (1 byte)
	if len(countryCode) >= 1 {
		header[0x3E] = countryCode[0]
	}

	// Version at 0x3F
	header[0x3F] = 0x00

	return header
}

// createN64HeaderByteSwapped creates a byte-swapped (.v64) format header
func createN64HeaderByteSwapped(cartID, countryCode, title string) []byte {
	// First create big-endian header
	header := createN64HeaderBigEndian(cartID, countryCode, title)

	// Then byte-swap it (swap pairs of bytes)
	for i := 0; i < len(header); i += 2 {
		header[i], header[i+1] = header[i+1], header[i]
	}

	return header
}

// createN64HeaderWordSwapped creates a word-swapped (.n64) format header
func createN64HeaderWordSwapped(cartID, countryCode, title string) []byte {
	// First create big-endian header
	header := createN64HeaderBigEndian(cartID, countryCode, title)

	// Then word-swap it (reverse each 4-byte word)
	for i := 0; i < len(header); i += 4 {
		header[i], header[i+1], header[i+2], header[i+3] =
			header[i+3], header[i+2], header[i+1], header[i]
	}

	return header
}

func TestN64Identifier_Identify(t *testing.T) {
	id := NewN64Identifier()

	tests := []struct {
		name      string
		header    []byte
		wantID    string
		wantTitle string
	}{
		{
			name:      "Big endian Z64",
			header:    createN64HeaderBigEndian("SM", "E", "SUPER MARIO 64"),
			wantID:    "SME",
			wantTitle: "SUPER MARIO 64",
		},
		{
			name:      "Byte-swapped V64",
			header:    createN64HeaderByteSwapped("ZL", "P", "ZELDA OCARINA"),
			wantID:    "ZLP",
			wantTitle: "ZELDA OCARINA",
		},
		{
			name:      "Word-swapped N64",
			header:    createN64HeaderWordSwapped("MK", "J", "MARIO KART 64"),
			wantID:    "MKJ",
			wantTitle: "MARIO KART 64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.header)

			result, err := id.Identify(r, int64(len(tt.header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}

			if result.InternalTitle != tt.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, tt.wantTitle)
			}

			if result.Console != ConsoleN64 {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleN64)
			}
		})
	}
}

func TestN64Identifier_InvalidMagic(t *testing.T) {
	id := NewN64Identifier()

	// Create header with invalid magic word
	header := make([]byte, 0x40)
	copy(header[0x20:], []byte("SOME GAME TITLE"))

	r := bytes.NewReader(header)
	_, err := id.Identify(r, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestN64Identifier_TooSmall(t *testing.T) {
	id := NewN64Identifier()

	header := make([]byte, 0x20) // Need at least 0x40

	r := bytes.NewReader(header)
	_, err := id.Identify(r, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestValidateN64(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Big endian",
			header: createN64HeaderBigEndian("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Byte-swapped",
			header: createN64HeaderByteSwapped("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Word-swapped",
			header: createN64HeaderWordSwapped("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Invalid",
			header: make([]byte, 0x40),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateN64(tt.header)
			if got != tt.want {
				t.Errorf("ValidateN64() = %v, want %v", got, tt.want)
			}
		})
	}
}
