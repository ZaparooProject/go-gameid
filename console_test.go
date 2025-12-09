package gameid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/identifier"
)

func TestDetectConsole(t *testing.T) {
	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		filename string
		content  []byte
		want     identifier.Console
		wantErr  bool
	}{
		{
			name:     "GBA by extension",
			filename: "game.gba",
			content:  make([]byte, 0xC0),
			want:     identifier.ConsoleGBA,
		},
		{
			name:     "GB by extension",
			filename: "game.gb",
			content:  make([]byte, 0x150),
			want:     identifier.ConsoleGB,
		},
		{
			name:     "GBC by extension",
			filename: "game.gbc",
			content:  make([]byte, 0x150),
			want:     identifier.ConsoleGBC,
		},
		{
			name:     "N64 z64 extension",
			filename: "game.z64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "N64 v64 extension",
			filename: "game.v64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "N64 n64 extension",
			filename: "game.n64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "NES by extension",
			filename: "game.nes",
			content:  make([]byte, 0x100),
			want:     identifier.ConsoleNES,
		},
		{
			name:     "SNES by extension",
			filename: "game.sfc",
			content:  make([]byte, 0x8000),
			want:     identifier.ConsoleSNES,
		},
		{
			name:     "Genesis gen extension",
			filename: "game.gen",
			content:  make([]byte, 0x200),
			want:     identifier.ConsoleGenesis,
		},
		{
			name:     "Genesis md extension",
			filename: "game.md",
			content:  make([]byte, 0x200),
			want:     identifier.ConsoleGenesis,
		},
		{
			name:     "GameCube gcm extension",
			filename: "game.gcm",
			content:  make([]byte, 0x100),
			want:     identifier.ConsoleGC,
		},
		{
			name:     "Unsupported extension",
			filename: "game.xyz",
			content:  make([]byte, 0x100),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			path := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(path, tt.content, 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			got, err := DetectConsole(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectConsole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DetectConsole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectConsole_GzSuffix(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .gba.gz file
	path := filepath.Join(tmpDir, "game.gba.gz")
	if err := os.WriteFile(path, make([]byte, 0x100), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	got, err := DetectConsole(path)
	if err != nil {
		t.Errorf("DetectConsole() error = %v", err)
		return
	}
	if got != identifier.ConsoleGBA {
		t.Errorf("DetectConsole() = %v, want %v", got, identifier.ConsoleGBA)
	}
}

func TestDetectConsole_NonExistent(t *testing.T) {
	_, err := DetectConsole("/nonexistent/path/game.gba")
	if err == nil {
		t.Error("DetectConsole() should error for non-existent file")
	}
}
