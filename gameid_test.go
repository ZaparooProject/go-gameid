package gameid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/identifier"
)

func TestParseConsole(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Console
		wantErr bool
	}{
		{"GB lowercase", "gb", ConsoleGB, false},
		{"GB uppercase", "GB", ConsoleGB, false},
		{"GameBoy", "gameboy", ConsoleGB, false},
		{"GBC", "gbc", ConsoleGBC, false},
		{"GBA", "gba", ConsoleGBA, false},
		{"GameCube", "gamecube", ConsoleGC, false},
		{"GC", "gc", ConsoleGC, false},
		{"NGC", "ngc", ConsoleGC, false},
		{"Genesis", "genesis", ConsoleGenesis, false},
		{"MegaDrive", "megadrive", ConsoleGenesis, false},
		{"MD", "md", ConsoleGenesis, false},
		{"N64", "n64", ConsoleN64, false},
		{"Nintendo64", "nintendo64", ConsoleN64, false},
		{"NES", "nes", ConsoleNES, false},
		{"Famicom", "famicom", ConsoleNES, false},
		{"SNES", "snes", ConsoleSNES, false},
		{"SuperFamicom", "superfamicom", ConsoleSNES, false},
		{"PSX", "psx", ConsolePSX, false},
		{"PS1", "ps1", ConsolePSX, false},
		{"PlayStation", "playstation", ConsolePSX, false},
		{"PS2", "ps2", ConsolePS2, false},
		{"PlayStation2", "playstation2", ConsolePS2, false},
		{"PSP", "psp", ConsolePSP, false},
		{"Saturn", "saturn", ConsoleSaturn, false},
		{"SegaSaturn", "segasaturn", ConsoleSaturn, false},
		{"SegaCD", "segacd", ConsoleSegaCD, false},
		{"MegaCD", "megacd", ConsoleSegaCD, false},
		{"NeoGeoCD", "neogeocd", ConsoleNeoGeoCD, false},
		{"Unknown", "xbox", "", true},
		{"Empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConsole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConsole(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseConsole(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSupportedConsoles(t *testing.T) {
	consoles := SupportedConsoles()
	if len(consoles) != len(AllConsoles) {
		t.Errorf("SupportedConsoles() returned %d consoles, want %d", len(consoles), len(AllConsoles))
	}

	// Check that all expected consoles are present
	expected := map[string]bool{
		"GB": true, "GBC": true, "GBA": true, "GC": true,
		"Genesis": true, "N64": true, "NeoGeoCD": true, "NES": true,
		"PSP": true, "PSX": true, "PS2": true, "Saturn": true,
		"SegaCD": true, "SNES": true,
	}

	for _, c := range consoles {
		if !expected[c] {
			t.Errorf("Unexpected console: %s", c)
		}
	}
}

func TestIsDiscBased(t *testing.T) {
	discBased := []Console{ConsoleGC, ConsoleNeoGeoCD, ConsolePSP, ConsolePSX, ConsolePS2, ConsoleSaturn, ConsoleSegaCD}
	cartBased := []Console{ConsoleGB, ConsoleGBC, ConsoleGBA, ConsoleGenesis, ConsoleN64, ConsoleNES, ConsoleSNES}

	for _, c := range discBased {
		if !IsDiscBased(c) {
			t.Errorf("IsDiscBased(%s) = false, want true", c)
		}
		if IsCartridgeBased(c) {
			t.Errorf("IsCartridgeBased(%s) = true, want false", c)
		}
	}

	for _, c := range cartBased {
		if IsDiscBased(c) {
			t.Errorf("IsDiscBased(%s) = true, want false", c)
		}
		if !IsCartridgeBased(c) {
			t.Errorf("IsCartridgeBased(%s) = false, want true", c)
		}
	}
}

// createTestGBAFile creates a minimal valid GBA ROM for testing
func createTestGBAFile(t *testing.T, tmpDir string) string {
	// GBA header is 192 bytes
	header := make([]byte, 0xC0)

	// Entry point
	copy(header[0x00:], []byte{0x00, 0x00, 0x00, 0xEA})

	// Nintendo logo (156 bytes at 0x04) - use the actual logo
	nintendoLogo := []byte{
		0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A,
		0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21,
		0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A,
		0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
		0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0,
		0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61,
		0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61,
		0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
		0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85,
		0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2,
		0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB,
		0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
		0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
	}
	copy(header[0x04:], nintendoLogo)

	// Internal title (12 bytes at 0xA0)
	copy(header[0xA0:], []byte("TESTGAME    "))

	// Game code (4 bytes at 0xAC)
	copy(header[0xAC:], []byte("ATST"))

	// Maker code (2 bytes at 0xB0)
	copy(header[0xB0:], []byte("01"))

	// Fixed value (0xB2)
	header[0xB2] = 0x96

	path := filepath.Join(tmpDir, "test.gba")
	if err := os.WriteFile(path, header, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	return path
}

func TestIdentifyWithConsole(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a GBA test file
	gbaPath := createTestGBAFile(t, tmpDir)

	// Test identification
	result, err := IdentifyWithConsole(gbaPath, ConsoleGBA, nil)
	if err != nil {
		t.Fatalf("IdentifyWithConsole() error = %v", err)
	}

	if result.Console != identifier.ConsoleGBA {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGBA)
	}

	if result.ID != "ATST" {
		t.Errorf("ID = %q, want %q", result.ID, "ATST")
	}

	if result.InternalTitle != "TESTGAME" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "TESTGAME")
	}
}

func TestIdentify(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a GBA test file
	gbaPath := createTestGBAFile(t, tmpDir)

	// Test auto-detection and identification
	result, err := Identify(gbaPath, nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleGBA {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGBA)
	}

	if result.ID != "ATST" {
		t.Errorf("ID = %q, want %q", result.ID, "ATST")
	}
}

func TestIdentify_NonExistent(t *testing.T) {
	_, err := Identify("/nonexistent/path/game.gba", nil)
	if err == nil {
		t.Error("Identify() should error for non-existent file")
	}
}

func TestIdentify_UnsupportedFormat(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with unsupported extension
	path := filepath.Join(tmpDir, "game.xyz")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = Identify(path, nil)
	if err == nil {
		t.Error("Identify() should error for unsupported format")
	}
}

func TestIsBlockDevice(t *testing.T) {
	// Regular files should not be detected as block devices
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	regularFile := filepath.Join(tmpDir, "test.iso")
	if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if isBlockDevice(regularFile) {
		t.Error("isBlockDevice() should return false for regular file")
	}

	// Non-/dev/ paths should return false
	if isBlockDevice("/tmp/not-a-device") {
		t.Error("isBlockDevice() should return false for non-/dev/ paths")
	}

	// Non-existent paths should return false
	if isBlockDevice("/dev/nonexistent123456789") {
		t.Error("isBlockDevice() should return false for non-existent device")
	}
}
