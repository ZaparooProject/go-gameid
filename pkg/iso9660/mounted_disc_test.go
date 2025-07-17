package iso9660

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMountedDisc(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFiles := map[string]string{
		"SLUS_123.45": "Test PSX game",
		"README.TXT":  "This is a test",
		"DATA.BIN":    "Binary data here",
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Open as mounted disc
	disc, err := OpenMountedDisc(tmpDir, "2024-01-01-12-00-00-00", "TEST_DISC")
	if err != nil {
		t.Fatalf("Failed to open mounted disc: %v", err)
	}
	defer disc.Close()

	// Test GetPVD
	pvd := disc.GetPVD()
	if pvd == nil {
		t.Fatal("GetPVD returned nil")
	}
	if pvd.VolumeID != "TEST_DISC" {
		t.Errorf("Expected VolumeID 'TEST_DISC', got '%s'", pvd.VolumeID)
	}
	if pvd.CreationDateTime != "2024-01-01-12-00-00-00" {
		t.Errorf("Expected CreationDateTime '2024-01-01-12-00-00-00', got '%s'", pvd.CreationDateTime)
	}

	// Test ListFiles
	files, err := disc.ListFiles(true)
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
	}

	// Check that all files are present
	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file.Name] = true
	}

	for name := range testFiles {
		expectedName := "/" + name
		if !fileMap[expectedName] {
			t.Errorf("File %s not found in listing", expectedName)
		}
	}

	// Test ReadFileByName
	data, err := disc.ReadFileByName("README.TXT")
	if err != nil {
		t.Fatalf("Failed to read README.TXT: %v", err)
	}
	if string(data) != testFiles["README.TXT"] {
		t.Errorf("Expected content '%s', got '%s'", testFiles["README.TXT"], string(data))
	}

	// Test ReadFileByEntry
	var readmeEntry *FileEntry
	for _, file := range files {
		if file.Name == "/README.TXT" {
			readmeEntry = &file
			break
		}
	}
	if readmeEntry == nil {
		t.Fatal("README.TXT not found in file list")
	}

	data, err = disc.ReadFileByEntry(readmeEntry)
	if err != nil {
		t.Fatalf("Failed to read README.TXT by entry: %v", err)
	}
	if string(data) != testFiles["README.TXT"] {
		t.Errorf("Expected content '%s', got '%s'", testFiles["README.TXT"], string(data))
	}
}

func TestOpenImage_Directory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open as disc image
	disc, err := OpenImage(tmpDir, "test-uuid", "test-label")
	if err != nil {
		t.Fatalf("Failed to open image: %v", err)
	}
	defer disc.Close()

	// Verify it's a mounted disc
	if _, ok := disc.(*MountedDisc); !ok {
		t.Error("Expected MountedDisc type")
	}

	// Check PVD
	pvd := disc.GetPVD()
	if pvd == nil {
		t.Fatal("GetPVD returned nil")
	}
	if pvd.VolumeID != "test-label" {
		t.Errorf("Expected VolumeID 'test-label', got '%s'", pvd.VolumeID)
	}
	if pvd.CreationDateTime != "test-uuid" {
		t.Errorf("Expected CreationDateTime 'test-uuid', got '%s'", pvd.CreationDateTime)
	}
}
