package fileio

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFile_RegularFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")
	
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test opening regular file
	reader, err := OpenFile(testFile)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer reader.Close()

	// Read and verify content
	data, err := ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("Content mismatch: got %s, want %s", data, content)
	}
}

func TestOpenFile_GzipFile(t *testing.T) {
	// Create a gzip file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gz")
	content := []byte("Compressed content")

	// Write gzip file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create gzip file: %v", err)
	}
	
	gw := gzip.NewWriter(f)
	if _, err := gw.Write(content); err != nil {
		t.Fatalf("Failed to write gzip content: %v", err)
	}
	gw.Close()
	f.Close()

	// Test opening gzip file
	reader, err := OpenFile(testFile)
	if err != nil {
		t.Fatalf("Failed to open gzip file: %v", err)
	}
	defer reader.Close()

	// Read and verify content
	data, err := ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read gzip file: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("Content mismatch: got %s, want %s", data, content)
	}
}

func TestOpenFile_Stdin(t *testing.T) {
	// Test that stdin is recognized (actual reading would require mocking)
	reader, err := OpenFile("stdin")
	if err != nil {
		t.Fatalf("Failed to open stdin: %v", err)
	}
	
	// We can't easily test reading from stdin in unit tests
	// Just ensure it doesn't error
	if reader == nil {
		t.Error("Expected non-nil reader for stdin")
	}
}

func TestOpenFile_NonExistent(t *testing.T) {
	_, err := OpenFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetSize_RegularFile(t *testing.T) {
	// Create test file with known size
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")
	content := make([]byte, 1234) // 1234 bytes
	
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := GetSize(testFile)
	if err != nil {
		t.Fatalf("Failed to get file size: %v", err)
	}

	if size != 1234 {
		t.Errorf("Size mismatch: got %d, want 1234", size)
	}
}

func TestGetSize_Directory(t *testing.T) {
	// Create directory with files
	tmpDir := t.TempDir()
	
	// Create some files
	files := []struct {
		name string
		size int
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"subdir/file3.txt", 300},
	}

	totalSize := int64(0)
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		os.MkdirAll(filepath.Dir(path), 0755)
		content := make([]byte, f.size)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f.name, err)
		}
		totalSize += int64(f.size)
	}

	size, err := GetSize(tmpDir)
	if err != nil {
		t.Fatalf("Failed to get directory size: %v", err)
	}

	if size != totalSize {
		t.Errorf("Size mismatch: got %d, want %d", size, totalSize)
	}
}

func TestGetExtension(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"game.gba", "gba"},
		{"game.GBA", "gba"},
		{"game.bin.gz", "bin"},
		{"game.tar.gz", "tar"},
		{"game", ""},
		{"game.ISO.gz", "iso"},
		{"/path/to/game.n64", "n64"},
		{"game.multiple.dots.snes", "snes"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := GetExtension(tt.filename)
			if got != tt.want {
				t.Errorf("GetExtension(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestBinsFromCue(t *testing.T) {
	// Create a test CUE file
	tmpDir := t.TempDir()
	cueFile := filepath.Join(tmpDir, "test.cue")
	
	cueContent := `FILE "track01.bin" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00
FILE "track02.bin" BINARY
  TRACK 02 AUDIO
    INDEX 00 00:00:00
    INDEX 01 00:02:00`

	if err := os.WriteFile(cueFile, []byte(cueContent), 0644); err != nil {
		t.Fatalf("Failed to create CUE file: %v", err)
	}

	bins, err := BinsFromCue(cueFile)
	if err != nil {
		t.Fatalf("Failed to parse CUE file: %v", err)
	}

	expected := []string{
		filepath.Join(tmpDir, "track01.bin"),
		filepath.Join(tmpDir, "track02.bin"),
	}

	if len(bins) != len(expected) {
		t.Fatalf("Wrong number of bins: got %d, want %d", len(bins), len(expected))
	}

	for i, bin := range bins {
		if bin != expected[i] {
			t.Errorf("Bin %d mismatch: got %q, want %q", i, bin, expected[i])
		}
	}
}

func TestBinsFromCue_NotCueFile(t *testing.T) {
	_, err := BinsFromCue("test.iso")
	if err == nil {
		t.Error("Expected error for non-CUE file")
	}
}

func TestCheckExists(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existingFile, []byte("test"), 0644)

	tests := []struct {
		path    string
		wantErr bool
	}{
		{existingFile, false},
		{tmpDir, false},
		{"/nonexistent/file", true},
		{"/dev/null", false}, // Special case for /dev/ paths
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := CheckExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckExists(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
			}
		})
	}
}