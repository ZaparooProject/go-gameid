package iso9660

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFile_CUE(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a minimal ISO as BIN file
	binPath := filepath.Join(tmpDir, "test.bin")
	binData := createMinimalISO(2352) // CD-ROM format with 2352 byte sectors

	if err := os.WriteFile(binPath, binData, 0644); err != nil {
		t.Fatalf("Failed to create BIN file: %v", err)
	}

	// Create CUE file
	cuePath := filepath.Join(tmpDir, "test.cue")
	cueContent := `FILE "test.bin" BINARY
  TRACK 01 MODE1/2352
    INDEX 01 00:00:00`

	if err := os.WriteFile(cuePath, []byte(cueContent), 0644); err != nil {
		t.Fatalf("Failed to create CUE file: %v", err)
	}

	// Test opening CUE file
	iso, err := OpenFile(cuePath)
	if err != nil {
		t.Fatalf("Failed to open CUE file: %v", err)
	}
	defer iso.Close()

	// Verify it loaded correctly
	if iso.SectorSize != 2352 {
		t.Errorf("Expected sector size 2352, got %d", iso.SectorSize)
	}

	if iso.PVD.VolumeID != "TEST_ISO" {
		t.Errorf("Expected volume ID 'TEST_ISO', got '%s'", iso.PVD.VolumeID)
	}
}

func TestOpenFile_ISO(t *testing.T) {
	// Create a temporary ISO file
	tmpDir := t.TempDir()
	isoPath := filepath.Join(tmpDir, "test.iso")

	isoData := createMinimalISO(2048)
	if err := os.WriteFile(isoPath, isoData, 0644); err != nil {
		t.Fatalf("Failed to create ISO file: %v", err)
	}

	// Test opening ISO file
	iso, err := OpenFile(isoPath)
	if err != nil {
		t.Fatalf("Failed to open ISO file: %v", err)
	}
	defer iso.Close()

	// Verify it loaded correctly
	if iso.SectorSize != 2048 {
		t.Errorf("Expected sector size 2048, got %d", iso.SectorSize)
	}

	if iso.PVD.VolumeID != "TEST_ISO" {
		t.Errorf("Expected volume ID 'TEST_ISO', got '%s'", iso.PVD.VolumeID)
	}
}

func TestMultiFileReader(t *testing.T) {
	// Create test files
	tmpDir := t.TempDir()

	file1Path := filepath.Join(tmpDir, "file1.bin")
	file2Path := filepath.Join(tmpDir, "file2.bin")

	data1 := []byte("Hello, ")
	data2 := []byte("World!")

	if err := os.WriteFile(file1Path, data1, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	if err := os.WriteFile(file2Path, data2, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Test multi-file reader
	reader, totalSize, err := newMultiFileReader([]string{file1Path, file2Path})
	if err != nil {
		t.Fatalf("Failed to create multi-file reader: %v", err)
	}
	defer reader.Close()

	if totalSize != int64(len(data1)+len(data2)) {
		t.Errorf("Expected total size %d, got %d", len(data1)+len(data2), totalSize)
	}

	// Test reading across files
	buf := make([]byte, totalSize)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if n != int(totalSize) {
		t.Errorf("Expected to read %d bytes, got %d", totalSize, n)
	}

	expected := "Hello, World!"
	if string(buf) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(buf))
	}

	// Test reading from middle
	buf2 := make([]byte, 5)
	n, err = reader.ReadAt(buf2, 4)
	if err != nil {
		t.Fatalf("Failed to read from middle: %v", err)
	}

	if string(buf2) != "o, Wo" {
		t.Errorf("Expected 'o, Wo', got '%s'", string(buf2))
	}
}
