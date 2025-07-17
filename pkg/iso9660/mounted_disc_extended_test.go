package iso9660_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

// createTestDir is a helper function to set up a temporary directory with a predefined file structure.
// It takes a map where keys are relative file paths and values are their content.
// Directories are created automatically.
func createTestDir(t *testing.T, files map[string]string) string {
	tempDir := t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}
	return tempDir
}

func TestOpenMountedDisc(t *testing.T) {
	t.Run("Success_EmptyDirectory", func(t *testing.T) {
		// Arrange: Create an empty temporary directory.
		tempDir := createTestDir(t, nil)
		uuid := "test-uuid-1"
		volumeID := "TEST_VOLUME_1"

		// Act: Open the directory as a mounted disc.
		disc, err := iso9660.OpenMountedDisc(tempDir, uuid, volumeID)
		// Assert: No error, disc and PVD are not nil, and PVD fields are correctly populated.
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}
		defer disc.Close()

		if disc == nil {
			t.Fatal("Expected disc not to be nil")
		}
		if disc.GetPVD() == nil {
			t.Fatal("Expected PVD not to be nil")
		}
		if disc.GetPVD().CreationDateTime != uuid {
			t.Errorf("Expected PVD UUID %q, got %q", uuid, disc.GetPVD().CreationDateTime)
		}
		if disc.GetPVD().VolumeID != volumeID {
			t.Errorf("Expected PVD VolumeID %q, got %q", volumeID, disc.GetPVD().VolumeID)
		}
		// Verify default sector size and offset for mounted discs (as per LINE 48-49)
		if disc.SectorSize != iso9660.SectorSize2048 {
			t.Errorf("Expected SectorSize %d, got %d", iso9660.SectorSize2048, disc.SectorSize)
		}
		if disc.SectorOffset != 0 {
			t.Errorf("Expected SectorOffset %d, got %d", 0, disc.SectorOffset)
		}
	})

	t.Run("Error_PathDoesNotExist", func(t *testing.T) {
		// Arrange: A path that does not exist.
		nonExistentPath := filepath.Join(t.TempDir(), "non_existent_dir") // Use TempDir to ensure parent exists but target doesn't

		// Act: Attempt to open the non-existent path.
		disc, err := iso9660.OpenMountedDisc(nonExistentPath, "", "")
		// Assert: An error is returned, and disc is nil. Error message should indicate stat failure (LINE 27).
		if err == nil {
			t.Fatal("Expected an error for non-existent path, got nil")
		}
		if disc != nil {
			t.Fatal("Expected disc to be nil for non-existent path")
		}
		if !strings.Contains(err.Error(), "failed to stat path") {
			t.Errorf("Expected 'failed to stat path' error, got: %v", err)
		}
	})

	t.Run("Error_PathIsFile", func(t *testing.T) {
		// Arrange: Create a temporary file.
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test_file.txt")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Act: Attempt to open the file as a mounted disc.
		disc, err := iso9660.OpenMountedDisc(filePath, "", "")
		// Assert: An error is returned, and disc is nil. Error message should indicate path is not a directory (LINE 30).
		if err == nil {
			t.Fatal("Expected an error for file path, got nil")
		}
		if disc != nil {
			t.Fatal("Expected disc to be nil for file path")
		}
		if !strings.Contains(err.Error(), "path must be a directory") {
			t.Errorf("Expected 'path must be a directory' error, got: %v", err)
		}
	})

	t.Run("Success_PathCleaned", func(t *testing.T) {
		// Arrange: Create a directory with a path that needs cleaning (e.g., with redundant slashes or '..').
		tempDir := t.TempDir()
		actualDir := filepath.Join(tempDir, "actual_dir")
		dirtyPath := filepath.Join(tempDir, "subdir", "..", "actual_dir") // This resolves to actualDir
		if err := os.MkdirAll(actualDir, 0755); err != nil {
			t.Fatalf("Failed to create actual directory: %v", err)
		}
		// Create a file inside the actual directory to verify access later
		if err := os.WriteFile(filepath.Join(actualDir, "test.txt"), []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to write file in actual directory: %v", err)
		}

		// Act: Open the disc using the dirty path.
		disc, err := iso9660.OpenMountedDisc(dirtyPath, "", "")
		if err != nil {
			t.Fatalf("OpenMountedDisc failed for dirty path: %v", err)
		}
		defer disc.Close()

		// Assert: Verify that operations on the disc work correctly, implying the path was cleaned internally (LINE 34).
		// This is an implicit test of filepath.Clean.
		files, err := disc.ListFiles(true)
		if err != nil {
			t.Fatalf("ListFiles failed on disc opened with dirty path: %v", err)
		}
		if len(files) != 1 || files[0].Name != "/test.txt" {
			t.Errorf("Expected 1 file named /test.txt, got %+v", files)
		}
	})
}

func TestMountedDiscListFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string // map of relative path to content
		onlyRootDir bool
		expected    []iso9660.FileEntry
	}{
		{
			name:        "OnlyRootDir_Empty",
			files:       nil, // Represents an empty directory
			onlyRootDir: true,
			expected:    []iso9660.FileEntry{},
		},
		{
			name: "OnlyRootDir_RootFiles",
			files: map[string]string{
				"fileA.txt": "contentA",
				"fileB.txt": "contentB",
			},
			onlyRootDir: true,
			expected: []iso9660.FileEntry{
				{Name: "/fileA.txt", Size: 8}, // Files are sorted alphabetically (LINE 104)
				{Name: "/fileB.txt", Size: 8},
			},
		},
		{
			name: "OnlyRootDir_RootFilesAndSubdir",
			files: map[string]string{
				"file1.txt":        "content1",
				"subdir/file2.txt": "content2", // This should be ignored
				"another_file.txt": "content3",
			},
			onlyRootDir: true,
			expected: []iso9660.FileEntry{
				{Name: "/another_file.txt", Size: 8},
				{Name: "/file1.txt", Size: 8},
			},
		},
		{
			name:        "Recursive_Empty",
			files:       nil,
			onlyRootDir: false,
			expected:    []iso9660.FileEntry{},
		},
		{
			name: "Recursive_RootFiles",
			files: map[string]string{
				"fileA.txt": "contentA",
				"fileB.txt": "contentB",
			},
			onlyRootDir: false,
			expected: []iso9660.FileEntry{
				{Name: "/fileA.txt", Size: 8},
				{Name: "/fileB.txt", Size: 8},
			},
		},
		{
			name: "Recursive_NestedFiles",
			files: map[string]string{
				"file1.txt":                  "content1",
				"subdir/file2.txt":           "content2",
				"subdir/subsubdir/file3.txt": "content3",
				"another_file.txt":           "content4",
			},
			onlyRootDir: false,
			expected: []iso9660.FileEntry{
				{Name: "/another_file.txt", Size: 8},
				{Name: "/file1.txt", Size: 8},
				{Name: "/subdir/file2.txt", Size: 8},
				{Name: "/subdir/subsubdir/file3.txt", Size: 8},
			},
		},
		{
			name: "Recursive_SpecialCharsInNames",
			files: map[string]string{
				"file with spaces.txt": "space content",
				"file-with_dashes.txt": "dash content",
				"file!@#$%^&*.txt":     "special content",
			},
			onlyRootDir: false,
			expected: []iso9660.FileEntry{
				{Name: "/file with spaces.txt", Size: 13},
				{Name: "/file!@#$%^&*.txt", Size: 15},
				{Name: "/file-with_dashes.txt", Size: 12},
			},
		},
		{
			name: "Recursive_ZeroSizeFile",
			files: map[string]string{
				"empty.txt": "",
				"data.txt":  "some data",
			},
			onlyRootDir: false,
			expected: []iso9660.FileEntry{
				{Name: "/data.txt", Size: 9},
				{Name: "/empty.txt", Size: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create the test directory structure.
			tempDir := createTestDir(t, tt.files)
			disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
			if err != nil {
				t.Fatalf("OpenMountedDisc failed: %v", err)
			}
			defer disc.Close()

			// Act: List files.
			files, err := disc.ListFiles(tt.onlyRootDir)
			// Assert: No error, and the list of files matches expectations.
			if err != nil {
				t.Fatalf("ListFiles failed: %v", err)
			}

			if len(files) != len(tt.expected) {
				t.Errorf("Expected %d files, got %d", len(tt.expected), len(files))
				t.Logf("Got: %+v", files)
				t.Logf("Expected: %+v", tt.expected)
				return
			}

			for i, f := range files {
				if f.Name != tt.expected[i].Name || f.Size != tt.expected[i].Size {
					t.Errorf("File %d mismatch: Expected {Name: %q, Size: %d}, Got {Name: %q, Size: %d}",
						i, tt.expected[i].Name, tt.expected[i].Size, f.Name, f.Size)
				}
			}
		})
	}

	t.Run("Error_DirectoryUnreadable", func(t *testing.T) {
		// This scenario (e.g., permissions changed after opening) is difficult to test
		// deterministically and portably across different operating systems without
		// elevated privileges or complex mocking of OS functions.
		// For now, this test is skipped. In a real-world scenario, this might be
		// covered by integration tests or require a more sophisticated test setup.
		t.Skip("Skipping directory unreadable test due to portability issues")
	})
}

func TestMountedDiscReadFileByName(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string // map of relative path to content
		fileName        string            // The name to pass to ReadFileByName
		expectedContent string
		expectError     bool
		errorContains   string // Substring expected in the error message
	}{
		{
			name:            "Success_ExistingFile",
			files:           map[string]string{"test.txt": "hello world"},
			fileName:        "test.txt",
			expectedContent: "hello world",
			expectError:     false,
		},
		{
			name:            "Success_EmptyFile",
			files:           map[string]string{"empty.txt": ""},
			fileName:        "empty.txt",
			expectedContent: "",
			expectError:     false,
		},
		{
			name:            "Success_NestedFile",
			files:           map[string]string{"subdir/nested.txt": "nested content"},
			fileName:        "subdir/nested.txt",
			expectedContent: "nested content",
			expectError:     false,
		},
		{
			name:            "Error_NonExistentFile",
			files:           map[string]string{"test.txt": "content"},
			fileName:        "non_existent.txt",
			expectedContent: "",
			expectError:     true,
			errorContains:   "failed to read file", // Error from os.ReadFile (LINE 128)
		},
		{
			name:            "Error_FileIsDirectory",
			files:           map[string]string{}, // Empty - we'll create directory manually
			fileName:        "subdir",
			expectedContent: "",
			expectError:     true,
			errorContains:   "is a directory", // Error from os.ReadFile when trying to read a directory
		},
		{
			name:            "Input_LeadingSlash",
			files:           map[string]string{"test.txt": "content"},
			fileName:        "/test.txt", // Should be handled by strings.TrimPrefix (LINE 121)
			expectedContent: "content",
			expectError:     false,
		},
		{
			name:            "Input_MultipleSlashes",
			files:           map[string]string{"a/b/c.txt": "content"},
			fileName:        "a//b/c.txt", // filepath.Join should normalize this
			expectedContent: "content",
			expectError:     false,
		},
		{
			name:            "Security_PathTraversalAttempt_WithinRoot",
			files:           map[string]string{"safe.txt": "safe content", "secret/file.txt": "secret"},
			fileName:        "secret/../safe.txt", // Should resolve to safe.txt within the root
			expectedContent: "safe content",
			expectError:     false,
		},
		{
			name:            "Security_PathTraversalAttempt_OutsideRoot",
			files:           map[string]string{"safe.txt": "safe content"},
			fileName:        "../../outside.txt", // This path will be joined with tempDir, likely resulting in a non-existent path
			expectedContent: "",
			expectError:     true,
			errorContains:   "attempted path traversal", // Updated error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create the test directory structure.
			tempDir := createTestDir(t, tt.files)
			// For the directory test case, create the directory manually
			if tt.name == "Error_FileIsDirectory" {
				dirPath := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dirPath, err)
				}
			}

			disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
			if err != nil {
				t.Fatalf("OpenMountedDisc failed: %v", err)
			}
			defer disc.Close()

			// Act: Read the file by name.
			content, err := disc.ReadFileByName(tt.fileName)

			// Assert: Check for expected error or content.
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error for %q, got nil", tt.fileName)
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error for %q, got %v", tt.fileName, err)
				}
				if string(content) != tt.expectedContent {
					t.Errorf("Expected content %q, got %q", tt.expectedContent, string(content))
				}
			}
		})
	}
}

func TestMountedDiscReadFileByEntry(t *testing.T) {
	t.Run("Success_ValidEntry", func(t *testing.T) {
		// Arrange: Create a file and a corresponding FileEntry.
		tempDir := createTestDir(t, map[string]string{"data.bin": "binary data"})
		disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}
		defer disc.Close()

		entry := &iso9660.FileEntry{
			Name: "/data.bin",
			LBA:  0, // LBA is not used for mounted discs (LINE 147)
			Size: uint32(len("binary data")),
		}

		// Act: Read the file using its entry.
		content, err := disc.ReadFileByEntry(entry)
		// Assert: No error, and content matches.
		if err != nil {
			t.Fatalf("ReadFileByEntry failed: %v", err)
		}
		if string(content) != "binary data" {
			t.Errorf("Expected content %q, got %q", "binary data", string(content))
		}
	})

	t.Run("Error_InvalidEntryName", func(t *testing.T) {
		// Arrange: Create a disc and an entry for a non-existent file.
		tempDir := createTestDir(t, map[string]string{"file.txt": "content"})
		disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}
		defer disc.Close()

		entry := &iso9660.FileEntry{
			Name: "/non_existent.txt",
			LBA:  0,
			Size: 0,
		}

		// Act: Attempt to read using the invalid entry.
		_, err = disc.ReadFileByEntry(entry)
		// Assert: An error is returned, indicating file read failure.
		if err == nil {
			t.Fatal("Expected an error for non-existent file entry, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read file") {
			t.Errorf("Expected 'failed to read file' error, got: %v", err)
		}
	})
}

func TestMountedDiscReadFileLBA(t *testing.T) {
	t.Run("AlwaysReturnsNotSupported", func(t *testing.T) {
		// Arrange: Open a mounted disc.
		tempDir := createTestDir(t, nil)
		disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}
		defer disc.Close()

		// Act: Call ReadFile with arbitrary LBA and size.
		_, err = disc.ReadFile(100, 2048) // Arbitrary LBA and size
		// Assert: An error is always returned, explicitly stating LBA is not supported (LINE 115).
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "ReadFile with LBA not supported for mounted discs") {
			t.Errorf("Expected 'not supported' error, got: %v", err)
		}
	})
}

func TestMountedDiscClose(t *testing.T) {
	t.Run("AlwaysReturnsNil", func(t *testing.T) {
		// Arrange: Open a mounted disc.
		tempDir := createTestDir(t, nil)
		disc, err := iso9660.OpenMountedDisc(tempDir, "", "")
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}

		// Act: Close the disc.
		err = disc.Close()
		// Assert: Close always returns nil (LINE 138).
		if err != nil {
			t.Errorf("Expected Close to return nil, got %v", err)
		}
	})
}

func TestMountedDiscGetPVD(t *testing.T) {
	t.Run("ReturnsCorrectPVD", func(t *testing.T) {
		// Arrange: Open a mounted disc with specific UUID and VolumeID.
		tempDir := createTestDir(t, nil)
		uuid := "test-uuid-getpvd"
		volumeID := "TEST_VOLUME_GETPVD"

		disc, err := iso9660.OpenMountedDisc(tempDir, uuid, volumeID)
		if err != nil {
			t.Fatalf("OpenMountedDisc failed: %v", err)
		}
		defer disc.Close()

		// Act: Get the PVD.
		pvd := disc.GetPVD()
		// Assert: PVD is not nil and contains the expected UUID and VolumeID (LINE 37).
		if pvd == nil {
			t.Fatal("Expected PVD not to be nil")
		}
		if pvd.CreationDateTime != uuid {
			t.Errorf("Expected PVD UUID %q, got %q", uuid, pvd.CreationDateTime)
		}
		if pvd.VolumeID != volumeID {
			t.Errorf("Expected PVD VolumeID %q, got %q", volumeID, pvd.VolumeID)
		}
	})
}
