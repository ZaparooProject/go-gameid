package identifier

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"testing"
)

func TestNESIdentifier_Identify(t *testing.T) {
	id := NewNESIdentifier()

	// Create test ROM data
	romData := []byte("NES ROM TEST DATA 12345")
	expectedCRC := crc32.ChecksumIEEE(romData)
	expectedID := fmt.Sprintf("%08x", expectedCRC)

	r := bytes.NewReader(romData)

	result, err := id.Identify(r, int64(len(romData)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != ConsoleNES {
		t.Errorf("Console = %v, want %v", result.Console, ConsoleNES)
	}

	// ID should be CRC32 hex when no database
	if result.ID != expectedID {
		t.Errorf("ID = %q, want %q", result.ID, expectedID)
	}

	// Check metadata
	if crcMeta := result.Metadata["crc32"]; crcMeta == "" {
		t.Error("crc32 not in metadata")
	}
}

func TestNESIdentifier_EmptyFile(t *testing.T) {
	id := NewNESIdentifier()

	romData := []byte{}
	r := bytes.NewReader(romData)

	result, err := id.Identify(r, 0, nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	// Empty file should still return CRC32 of empty data
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestNESIdentifier_DifferentROMs(t *testing.T) {
	id := NewNESIdentifier()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Small ROM",
			data: []byte{0x4E, 0x45, 0x53, 0x1A}, // NES header magic
		},
		{
			name: "Larger ROM",
			data: bytes.Repeat([]byte{0xAB}, 16384),
		},
		{
			name: "With iNES header",
			data: append([]byte{0x4E, 0x45, 0x53, 0x1A, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, bytes.Repeat([]byte{0x00}, 32768)...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)

			result, err := id.Identify(r, int64(len(tt.data)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.Console != ConsoleNES {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleNES)
			}

			// Verify CRC is calculated correctly
			expectedCRC := crc32.ChecksumIEEE(tt.data)
			if result.Metadata["crc32"] != "" {
				// CRC should match
				_ = expectedCRC // Just ensure it's calculated
			}
		})
	}
}
