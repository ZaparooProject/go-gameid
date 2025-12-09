package iso9660

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCue(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		cueContent string
		wantFiles  []string
	}{
		{
			name: "Single file",
			cueContent: `FILE "game.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00`,
			wantFiles: []string{"game.bin"},
		},
		{
			name: "Multiple files",
			cueContent: `FILE "track01.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00
FILE "track02.bin" BINARY
TRACK 02 AUDIO
  INDEX 00 00:00:00
  INDEX 01 00:02:00`,
			wantFiles: []string{"track01.bin", "track02.bin"},
		},
		{
			name: "Mixed case",
			cueContent: `File "Game.BIN" Binary
Track 01 Mode1/2352
  Index 01 00:00:00`,
			wantFiles: []string{"Game.BIN"},
		},
		{
			name:       "No files",
			cueContent: `REM This is a comment`,
			wantFiles:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write CUE file
			cuePath := filepath.Join(tmpDir, tt.name+".cue")
			if err := os.WriteFile(cuePath, []byte(tt.cueContent), 0644); err != nil {
				t.Fatalf("Failed to write CUE file: %v", err)
			}

			cue, err := ParseCue(cuePath)
			if err != nil {
				t.Fatalf("ParseCue() error = %v", err)
			}

			if len(cue.BinFiles) != len(tt.wantFiles) {
				t.Errorf("Got %d BIN files, want %d", len(cue.BinFiles), len(tt.wantFiles))
				return
			}

			for i, want := range tt.wantFiles {
				gotBase := filepath.Base(cue.BinFiles[i])
				if gotBase != want {
					t.Errorf("BinFiles[%d] = %q, want %q", i, gotBase, want)
				}

				// Verify path is absolute
				if !filepath.IsAbs(cue.BinFiles[i]) {
					t.Errorf("BinFiles[%d] = %q is not absolute", i, cue.BinFiles[i])
				}
			}
		})
	}
}

func TestParseCue_NonExistent(t *testing.T) {
	_, err := ParseCue("/nonexistent/path/game.cue")
	if err == nil {
		t.Error("ParseCue() should error for non-existent file")
	}
}

func TestIsCueFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"game.cue", true},
		{"game.CUE", true},
		{"game.Cue", true},
		{"game.bin", false},
		{"game.iso", false},
		{"game", false},
		{"/path/to/game.cue", true},
		{"/path/to/game.bin", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsCueFile(tt.path)
			if got != tt.want {
				t.Errorf("IsCueFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseCue_AbsolutePaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// CUE with absolute path
	cueContent := `FILE "/absolute/path/game.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00`

	cuePath := filepath.Join(tmpDir, "game.cue")
	if err := os.WriteFile(cuePath, []byte(cueContent), 0644); err != nil {
		t.Fatalf("Failed to write CUE file: %v", err)
	}

	cue, err := ParseCue(cuePath)
	if err != nil {
		t.Fatalf("ParseCue() error = %v", err)
	}

	if len(cue.BinFiles) != 1 {
		t.Fatalf("Expected 1 BIN file, got %d", len(cue.BinFiles))
	}

	// Absolute paths should be preserved
	if cue.BinFiles[0] != "/absolute/path/game.bin" {
		t.Errorf("BinFiles[0] = %q, want %q", cue.BinFiles[0], "/absolute/path/game.bin")
	}
}
