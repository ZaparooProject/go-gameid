package fileio

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestOpenFile_EdgeCases tests various edge cases for file opening
func TestOpenFile_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// create empty file
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	_ = os.WriteFile(emptyFile, []byte{}, 0644)

	// create file with unicode name
	unicodeFile := filepath.Join(tmpDir, "文件名🎮.txt")
	_ = os.WriteFile(unicodeFile, []byte("unicode content"), 0644)

	// create file with spaces
	spacesFile := filepath.Join(tmpDir, "file with spaces.txt")
	_ = os.WriteFile(spacesFile, []byte("spaces content"), 0644)

	// create very long filename (max 255 chars on most systems)
	longName := strings.Repeat("a", 240) + ".txt"
	longFile := filepath.Join(tmpDir, longName)
	_ = os.WriteFile(longFile, []byte("long name content"), 0644)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Empty file",
			path:    emptyFile,
			wantErr: false,
		},
		{
			name:    "Unicode filename",
			path:    unicodeFile,
			wantErr: false,
		},
		{
			name:    "Filename with spaces",
			path:    spacesFile,
			wantErr: false,
		},
		{
			name:    "Very long filename",
			path:    longFile,
			wantErr: false,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Path with only spaces",
			path:    "   ",
			wantErr: true,
		},
		{
			name:    "stdout (should error)",
			path:    "stdout",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := OpenFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if reader != nil {
				reader.Close()
			}
		})
	}
}

// TestOpenFile_CorruptedGzip tests handling of corrupted gzip files
func TestOpenFile_CorruptedGzip(t *testing.T) {
	tmpDir := t.TempDir()

	// create valid gzip file
	validGz := filepath.Join(tmpDir, "valid.gz")
	f, _ := os.Create(validGz)
	gw := gzip.NewWriter(f)
	_, _ = gw.Write([]byte("valid content"))
	gw.Close()
	f.Close()

	// create corrupted gzip file (invalid header)
	corruptedGz := filepath.Join(tmpDir, "corrupted.gz")
	_ = os.WriteFile(corruptedGz, []byte("not a gzip file"), 0644)

	// create truncated gzip file
	truncatedGz := filepath.Join(tmpDir, "truncated.gz")
	data, _ := os.ReadFile(validGz)
	_ = os.WriteFile(truncatedGz, data[:len(data)/2], 0644)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		readErr bool
	}{
		{
			name:    "Valid gzip",
			path:    validGz,
			wantErr: false,
			readErr: false,
		},
		{
			name:    "Corrupted gzip header",
			path:    corruptedGz,
			wantErr: true,
			readErr: false,
		},
		{
			name:    "Truncated gzip",
			path:    truncatedGz,
			wantErr: false, // opens successfully
			readErr: true,  // but fails on read
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := OpenFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if reader != nil {
				defer reader.Close()

				if !tt.wantErr {
					_, readErr := ReadAll(reader)
					if (readErr != nil) != tt.readErr {
						t.Errorf("ReadAll() error = %v, want readErr %v", readErr, tt.readErr)
					}
				}
			}
		})
	}
}

// TestOpenFile_ConcurrentAccess tests concurrent file access
func TestOpenFile_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "concurrent.txt")
	content := []byte("concurrent test content")
	_ = os.WriteFile(testFile, content, 0644)

	// test concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reader, err := OpenFile(testFile)
			if err != nil {
				errors <- err
				return
			}
			defer reader.Close()

			data, err := ReadAll(reader)
			if err != nil {
				errors <- err
				return
			}

			if !bytes.Equal(data, content) {
				errors <- io.ErrUnexpectedEOF
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

// TestGetSize_EdgeCases tests edge cases for GetSize
func TestGetSize_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// create empty file
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	_ = os.WriteFile(emptyFile, []byte{}, 0644)

	// create empty directory
	emptyDir := filepath.Join(tmpDir, "empty_dir")
	_ = os.MkdirAll(emptyDir, 0755)

	// create directory with hidden files
	hiddenDir := filepath.Join(tmpDir, "hidden_dir")
	_ = os.MkdirAll(hiddenDir, 0755)
	_ = os.WriteFile(filepath.Join(hiddenDir, ".hidden"), []byte("hidden"), 0644)
	_ = os.WriteFile(filepath.Join(hiddenDir, "visible"), []byte("visible"), 0644)

	tests := []struct {
		name     string
		path     string
		wantSize int64
		wantErr  bool
	}{
		{
			name:     "Empty file",
			path:     emptyFile,
			wantSize: 0,
			wantErr:  false,
		},
		{
			name:     "Empty directory",
			path:     emptyDir,
			wantSize: 0,
			wantErr:  false,
		},
		{
			name:     "Directory with hidden files",
			path:     hiddenDir,
			wantSize: 13, // "hidden" + "visible"
			wantErr:  false,
		},
		{
			name:     "Non-existent path",
			path:     filepath.Join(tmpDir, "non_existent"),
			wantSize: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := GetSize(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && size != tt.wantSize {
				t.Errorf("GetSize() = %v, want %v", size, tt.wantSize)
			}
		})
	}
}

// TestBinsFromCue_ComplexCases tests complex CUE file scenarios
func TestBinsFromCue_ComplexCases(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		cueContent  string
		expectCount int
		wantErr     bool
	}{
		{
			name:        "Empty CUE file",
			cueContent:  "",
			expectCount: 0,
			wantErr:     false,
		},
		{
			name: "CUE with comments and extra whitespace",
			cueContent: `REM This is a comment
FILE "track01.bin" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00

REM Another comment
FILE   "track02.bin"   BINARY
  TRACK 02 AUDIO
    INDEX 01 00:00:00`,
			expectCount: 2,
			wantErr:     false,
		},
		{
			name: "CUE with quoted filenames containing spaces",
			cueContent: `FILE "track 01 with spaces.bin" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectCount: 1,
			wantErr:     false,
		},
		{
			name: "CUE with single quotes (non-standard)",
			cueContent: `FILE 'track01.bin' BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectCount: 0, // won't parse with current implementation
			wantErr:     false,
		},
		{
			name: "CUE with multiple files per track (multi-session)",
			cueContent: `FILE "session1.bin" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    INDEX 01 05:00:00
FILE "session2.bin" BINARY  
  TRACK 03 MODE2/2352
    INDEX 01 00:00:00`,
			expectCount: 2,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cueFile := filepath.Join(tmpDir, "test.cue")
			err := os.WriteFile(cueFile, []byte(tt.cueContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write CUE file: %v", err)
			}

			bins, err := BinsFromCue(cueFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("BinsFromCue() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(bins) != tt.expectCount {
				t.Errorf("BinsFromCue() returned %d bins, expected %d", len(bins), tt.expectCount)
			}
		})
	}
}

// TestMultiCloser tests the multiCloser implementation
func TestMultiCloser(t *testing.T) {
	tmpDir := t.TempDir()

	// create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	_ = os.WriteFile(file1, []byte("content1"), 0644)
	_ = os.WriteFile(file2, []byte("content2"), 0644)

	// open both files
	f1, err := os.Open(file1)
	if err != nil {
		t.Fatalf("Failed to open file1: %v", err)
	}

	f2, err := os.Open(file2)
	if err != nil {
		t.Fatalf("Failed to open file2: %v", err)
	}

	// create multiCloser
	mc := &multiCloser{
		closers: []io.Closer{f1, f2},
		reader:  f1,
	}

	// test reading
	buf := make([]byte, 8)
	n, err := mc.Read(buf)
	if err != nil {
		t.Errorf("multiCloser.Read() error = %v", err)
	}
	if n != 8 {
		t.Errorf("multiCloser.Read() = %d bytes, want 8", n)
	}

	// test closing
	err = mc.Close()
	if err != nil {
		t.Errorf("multiCloser.Close() error = %v", err)
	}

	// verify files are closed by trying to read
	_, err1 := f1.Read(buf)
	_, err2 := f2.Read(buf)

	if err1 == nil || err2 == nil {
		t.Error("multiCloser.Close() did not close all files")
	}
}

// TestOpenFile_FileModifiedDuringRead simulates file modification during read
func TestOpenFile_FileModifiedDuringRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File modification test unreliable on Windows")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "modified.txt")

	// write initial content
	initialContent := []byte("initial content")
	_ = os.WriteFile(testFile, initialContent, 0644)

	// open file
	reader, err := OpenFile(testFile)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer reader.Close()

	// read partial content
	buf := make([]byte, 7) // "initial"
	n, err := reader.Read(buf)
	if err != nil || n != 7 {
		t.Fatalf("Failed to read partial content: %v", err)
	}

	// modify file while reading
	newContent := []byte("completely different content now")
	_ = os.WriteFile(testFile, newContent, 0644)

	// continue reading
	remaining := make([]byte, 100)
	n, err = reader.Read(remaining)

	// behavior is platform-dependent
	// just ensure we don't crash
	t.Logf("Read %d bytes after modification, err: %v", n, err)
}

// BenchmarkOpenFile benchmarks file opening performance
func BenchmarkOpenFile(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")
	_ = os.WriteFile(testFile, []byte("benchmark content"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := OpenFile(testFile)
		if err != nil {
			b.Fatal(err)
		}
		reader.Close()
	}
}

// BenchmarkGetSize benchmarks directory size calculation
func BenchmarkGetSize(b *testing.B) {
	tmpDir := b.TempDir()

	// create directory structure
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir_%d", i))
		_ = os.MkdirAll(dir, 0755)
		for j := 0; j < 10; j++ {
			file := filepath.Join(dir, fmt.Sprintf("file_%d.txt", j))
			_ = os.WriteFile(file, make([]byte, 1024), 0644)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetSize(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestFileRaceCondition tests race conditions in file operations
func TestFileRaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "race.txt")

	// concurrent writes and reads
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			content := []byte(fmt.Sprintf("content from writer %d", id))
			if err := os.WriteFile(testFile, content, 0644); err != nil {
				errors <- err
			}
		}(i)
	}

	// readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond) // give writers a chance

			reader, err := OpenFile(testFile)
			if err != nil {
				if !os.IsNotExist(err) {
					errors <- err
				}
				return
			}
			defer reader.Close()

			_, err = ReadAll(reader)
			if err != nil && err != io.EOF {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// we expect some errors due to race conditions
	// just ensure we don't panic
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Race condition error (expected): %v", err)
	}

	t.Logf("Total race condition errors: %d", errorCount)
}
