package iso9660_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

func TestOpenImage(t *testing.T) {
	t.Run("OpenDirectoryAsMountedDisc", func(t *testing.T) {
		// Arrange: Create a temporary directory with a file inside.
		tempDir := createTestDir(t, map[string]string{"file.txt": "content"})
		uuid := "test-uuid-openimage"
		label := "TEST_LABEL_OPENIMAGE"

		// Act: Call OpenImage with the directory path.
		disc, err := iso9660.OpenImage(tempDir, uuid, label)
		// Assert: No error, and the returned DiscImage is a *MountedDisc (LINE 44).
		if err != nil {
			t.Fatalf("OpenImage failed for directory: %v", err)
		}
		defer disc.Close()

		mountedDisc, ok := disc.(*iso9660.MountedDisc)
		if !ok {
			t.Fatalf("Expected disc to be of type *iso9660.MountedDisc, got %T", disc)
		}

		// Further assert that the PVD details are correctly passed to MountedDisc (LINE 45-46).
		pvd := mountedDisc.GetPVD()
		if pvd == nil {
			t.Fatal("Expected PVD not to be nil")
		}
		if pvd.CreationDateTime != uuid {
			t.Errorf("Expected PVD UUID %q, got %q", uuid, pvd.CreationDateTime)
		}
		if pvd.VolumeID != label {
			t.Errorf("Expected PVD VolumeID %q, got %q", label, pvd.VolumeID)
		}

		// Verify basic functionality (e.g., ListFiles) to ensure it's a working MountedDisc.
		files, err := mountedDisc.ListFiles(true)
		if err != nil {
			t.Fatalf("ListFiles failed on mounted disc: %v", err)
		}
		if len(files) != 1 || files[0].Name != "/file.txt" {
			t.Errorf("Expected 1 file named /file.txt, got %+v", files)
		}
	})

	t.Run("OpenNonExistentPath", func(t *testing.T) {
		// Arrange: A path that does not exist.
		nonExistentPath := filepath.Join(t.TempDir(), "non_existent_path")

		// Act: Call OpenImage with the non-existent path.
		disc, err := iso9660.OpenImage(nonExistentPath, "", "")
		// Assert: An error is returned, and disc is nil. The error can originate from os.Stat (LINE 42)
		// or OpenFile (LINE 49).
		if err == nil {
			t.Fatal("Expected an error for non-existent path, got nil")
		}
		if disc != nil {
			t.Fatal("Expected disc to be nil for non-existent path")
		}
		if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "failed to open file") {
			t.Errorf("Expected 'no such file or directory' or 'failed to open file' error, got: %v", err)
		}
	})

	t.Run("OpenFileAsISO_NotMountedDisc", func(t *testing.T) {
		// Arrange: Create a dummy file (not a directory).
		// This test confirms that OpenImage correctly attempts to open a file as an ISO
		// (via OpenFile, LINE 49) and does NOT treat it as a MountedDisc.
		// We don't need a valid ISO for this test, just a file.
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "dummy.iso")
		if err := os.WriteFile(filePath, []byte("dummy iso content"), 0644); err != nil {
			t.Fatalf("Failed to create dummy ISO file: %v", err)
		}

		// Act: Call OpenImage with the file path.
		disc, err := iso9660.OpenImage(filePath, "", "")
		// Assert: If an error occurs (likely, as it's not a real ISO), ensure it's not
		// an error indicating it tried to open it as a directory. If it succeeds,
		// ensure it's not a MountedDisc.
		if err == nil {
			if _, ok := disc.(*iso9660.MountedDisc); ok {
				t.Fatalf("Expected disc NOT to be of type *iso9660.MountedDisc when opening a file")
			}
			disc.Close() // Close if it succeeded
		} else {
			if strings.Contains(err.Error(), "path must be a directory") {
				t.Fatalf("OpenImage incorrectly tried to open file as directory: %v", err)
			}
		}
	})

	t.Run("OpenFileAsISO_PVDOverride", func(t *testing.T) {
		// Arrange: Create a dummy file. We'll simulate OpenFile returning a valid ISO9660
		// object with some default PVD values.
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "dummy_with_pvd.iso")
		if err := os.WriteFile(filePath, []byte("dummy iso content"), 0644); err != nil {
			t.Fatalf("Failed to create dummy ISO file: %v", err)
		}

		// To properly test the PVD override logic (LINE 55-61), we would need to mock
		// the `iso9660.OpenFile` function to return a controllable `*iso9660.ISO9660` instance.
		// Without a mocking framework or refactoring `OpenFile` to accept an interface,
		// this specific path is hard to test deterministically.
		// For now, we acknowledge this and focus on the `MountedDisc` part of `OpenImage`.
		// If `OpenFile` were refactored (e.g., `OpenImage(path string, openFileFunc func(string) (*ISO9660, error))`),
		// this test would be straightforward.
		t.Skip("Skipping PVD override test for ISO files due to lack of mocking for OpenFile")
	})
}
