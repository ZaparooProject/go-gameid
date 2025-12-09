package identifier

import (
	"bytes"
	"testing"
)

// createSNESHeader creates a minimal valid SNES ROM with LoROM header for testing.
// The header is at 0x7FC0 for LoROM.
func createSNESHeader(internalName string, developerID, romVersion byte, checksum uint16) []byte {
	// Need at least 0x8000 bytes for LoROM header
	rom := make([]byte, 0x8000)

	headerStart := snesLoROMHeaderStart

	// Internal name (21 bytes at 0x00)
	nameBytes := []byte(internalName)
	if len(nameBytes) > 21 {
		nameBytes = nameBytes[:21]
	}
	copy(rom[headerStart+snesInternalNameOffset:], nameBytes)

	// Map mode (0x15) - LoROM, SlowROM
	rom[headerStart+snesMapModeOffset] = 0x20 // LoROM

	// ROM type (0x16)
	rom[headerStart+snesROMTypeOffset] = 0x00 // ROM only

	// Developer ID (0x1A)
	rom[headerStart+snesDeveloperIDOffset] = developerID

	// ROM version (0x1B)
	rom[headerStart+snesROMVersionOffset] = romVersion

	// Checksum complement (0x1C-0x1D) - checksum + complement = 0xFFFF
	complement := 0xFFFF - checksum
	rom[headerStart+snesChecksumComplementOffset] = byte(complement & 0xFF)
	rom[headerStart+snesChecksumComplementOffset+1] = byte(complement >> 8)

	// Checksum (0x1E-0x1F)
	rom[headerStart+snesChecksumOffset] = byte(checksum & 0xFF)
	rom[headerStart+snesChecksumOffset+1] = byte(checksum >> 8)

	return rom
}

// createSNESHeaderHiROM creates a SNES ROM with HiROM header.
func createSNESHeaderHiROM(internalName string, developerID, romVersion byte, checksum uint16) []byte {
	// Need at least 0x10000 bytes for HiROM header at 0xFFC0
	rom := make([]byte, 0x10000)

	headerStart := snesHiROMHeaderStart

	// Internal name (21 bytes at 0x00)
	nameBytes := []byte(internalName)
	if len(nameBytes) > 21 {
		nameBytes = nameBytes[:21]
	}
	copy(rom[headerStart+snesInternalNameOffset:], nameBytes)

	// Map mode (0x15) - HiROM, SlowROM
	rom[headerStart+snesMapModeOffset] = 0x21 // HiROM

	// ROM type (0x16)
	rom[headerStart+snesROMTypeOffset] = 0x02 // ROM + RAM + Battery

	// Developer ID (0x1A)
	rom[headerStart+snesDeveloperIDOffset] = developerID

	// ROM version (0x1B)
	rom[headerStart+snesROMVersionOffset] = romVersion

	// Checksum complement (0x1C-0x1D)
	complement := 0xFFFF - checksum
	rom[headerStart+snesChecksumComplementOffset] = byte(complement & 0xFF)
	rom[headerStart+snesChecksumComplementOffset+1] = byte(complement >> 8)

	// Checksum (0x1E-0x1F)
	rom[headerStart+snesChecksumOffset] = byte(checksum & 0xFF)
	rom[headerStart+snesChecksumOffset+1] = byte(checksum >> 8)

	return rom
}

func TestSNESIdentifier_Identify(t *testing.T) {
	id := NewSNESIdentifier()

	tests := []struct {
		name         string
		rom          []byte
		wantTitle    string
		wantROMType  string
		wantFastSlow string
	}{
		{
			name:         "LoROM Game",
			rom:          createSNESHeader("SUPER MARIO WORLD", 0x01, 0, 0x1234),
			wantTitle:    "SUPER MARIO WORLD",
			wantROMType:  "LoROM",
			wantFastSlow: "SlowROM",
		},
		{
			name:         "HiROM Game",
			rom:          createSNESHeaderHiROM("ZELDA3", 0x01, 1, 0xABCD),
			wantTitle:    "ZELDA3",
			wantROMType:  "HiROM",
			wantFastSlow: "SlowROM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.rom)

			result, err := id.Identify(r, int64(len(tt.rom)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.InternalTitle != tt.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, tt.wantTitle)
			}

			if result.Console != ConsoleSNES {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSNES)
			}

			if romType := result.Metadata["rom_type"]; romType != tt.wantROMType {
				t.Errorf("rom_type = %q, want %q", romType, tt.wantROMType)
			}

			if fastSlow := result.Metadata["fast_slow_rom"]; fastSlow != tt.wantFastSlow {
				t.Errorf("fast_slow_rom = %q, want %q", fastSlow, tt.wantFastSlow)
			}
		})
	}
}

func TestSNESIdentifier_SMCHeader(t *testing.T) {
	id := NewSNESIdentifier()

	// Create a ROM with 512-byte SMC header
	baseROM := createSNESHeader("SMC TEST GAME", 0x02, 0, 0x5678)
	smcHeader := make([]byte, 512)
	romWithSMC := append(smcHeader, baseROM...)

	r := bytes.NewReader(romWithSMC)

	result, err := id.Identify(r, int64(len(romWithSMC)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.InternalTitle != "SMC TEST GAME" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "SMC TEST GAME")
	}
}

func TestSNESIdentifier_InvalidChecksum(t *testing.T) {
	id := NewSNESIdentifier()

	// Create a ROM with invalid checksum (doesn't sum to 0xFFFF)
	rom := make([]byte, 0x8000)
	headerStart := snesLoROMHeaderStart
	copy(rom[headerStart:], []byte("INVALID"))
	// Don't set valid checksum+complement

	r := bytes.NewReader(rom)

	_, err := id.Identify(r, int64(len(rom)), nil)
	if err == nil {
		t.Error("expected error for invalid checksum, got nil")
	}
}

func TestValidateSNES(t *testing.T) {
	tests := []struct {
		name string
		rom  []byte
		want bool
	}{
		{
			name: "Valid LoROM",
			rom:  createSNESHeader("TEST", 0x01, 0, 0x1234),
			want: true,
		},
		{
			name: "Valid HiROM",
			rom:  createSNESHeaderHiROM("TEST", 0x01, 0, 0x1234),
			want: true,
		},
		{
			name: "Invalid",
			rom:  make([]byte, 0x8000),
			want: false,
		},
		{
			name: "With SMC header",
			rom:  append(make([]byte, 512), createSNESHeader("TEST", 0x01, 0, 0x1234)...),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSNES(tt.rom)
			if got != tt.want {
				t.Errorf("ValidateSNES() = %v, want %v", got, tt.want)
			}
		})
	}
}
