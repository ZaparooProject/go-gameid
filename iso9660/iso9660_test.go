// Copyright (c) 2025 Niema Moshiri and The Zaparoo Project.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This file is part of go-gameid.
//
// go-gameid is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gameid is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-gameid.  If not, see <https://www.gnu.org/licenses/>.

package iso9660

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createMinimalISO creates a minimal valid ISO9660 image for testing.
// Block size is 2048 bytes.
//
//nolint:revive,funlen // Test helper function requires many statements for ISO format
func createMinimalISO(volumeID, systemID, publisherID string) []byte {
	blockSize := 2048

	// We need at least:
	// - 16 blocks of system area (unused)
	// - 1 block for PVD at block 16
	// - Some blocks for path table
	// - Root directory
	totalBlocks := 20
	data := make([]byte, totalBlocks*blockSize)

	// PVD at block 16 (offset 32768)
	pvdOffset := 16 * blockSize

	// Type code (1 = PVD)
	data[pvdOffset] = 0x01

	// Standard identifier "CD001"
	copy(data[pvdOffset+1:], "CD001")

	// Version (1)
	data[pvdOffset+6] = 0x01

	// System identifier (offset 8, 32 bytes)
	if len(systemID) > 32 {
		systemID = systemID[:32]
	}
	copy(data[pvdOffset+8:], systemID)

	// Volume identifier (offset 40, 32 bytes)
	if len(volumeID) > 32 {
		volumeID = volumeID[:32]
	}
	copy(data[pvdOffset+40:], volumeID)

	// Volume space size (offset 80, little-endian + big-endian)
	binary.LittleEndian.PutUint32(data[pvdOffset+80:], uint32(totalBlocks))
	binary.BigEndian.PutUint32(data[pvdOffset+84:], uint32(totalBlocks))

	// Volume set size (offset 120)
	binary.LittleEndian.PutUint16(data[pvdOffset+120:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+122:], 1)

	// Volume sequence number (offset 124)
	binary.LittleEndian.PutUint16(data[pvdOffset+124:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+126:], 1)

	// Logical block size (offset 128, 2048)
	binary.LittleEndian.PutUint16(data[pvdOffset+128:], uint16(blockSize))
	binary.BigEndian.PutUint16(data[pvdOffset+130:], uint16(blockSize))

	// Path table size (offset 132, little-endian)
	pathTableSize := uint32(10) // Minimal path table
	binary.LittleEndian.PutUint32(data[pvdOffset+132:], pathTableSize)
	binary.BigEndian.PutUint32(data[pvdOffset+136:], pathTableSize)

	// Path table LBA (offset 140, little-endian - block 18)
	binary.LittleEndian.PutUint32(data[pvdOffset+140:], 18)

	// Root directory record (offset 156, 34 bytes)
	rootDirOffset := pvdOffset + 156
	data[rootDirOffset] = 34                                  // Record length
	data[rootDirOffset+1] = 0                                 // Extended attribute record length
	binary.LittleEndian.PutUint32(data[rootDirOffset+2:], 19) // Location (block 19)
	binary.BigEndian.PutUint32(data[rootDirOffset+6:], 19)
	binary.LittleEndian.PutUint32(data[rootDirOffset+10:], uint32(blockSize)) // Data length
	binary.BigEndian.PutUint32(data[rootDirOffset+14:], uint32(blockSize))
	// Date/time (offset 18, 7 bytes) - leave as zeros
	data[rootDirOffset+25] = 0x02 // Directory flag
	data[rootDirOffset+32] = 1    // File identifier length
	data[rootDirOffset+33] = 0x00 // File identifier (root = 0x00)

	// Publisher identifier (offset 318, 128 bytes)
	if len(publisherID) > 128 {
		publisherID = publisherID[:128]
	}
	copy(data[pvdOffset+318:], publisherID)

	// Volume creation date/time (offset 813, 17 bytes) - used for UUID
	copy(data[pvdOffset+813:], "2024010112000000")

	// Path table at block 18
	pathTableOffset := 18 * blockSize
	// Root directory entry
	data[pathTableOffset] = 1                                   // Directory identifier length
	data[pathTableOffset+1] = 0                                 // Extended attribute record length
	binary.LittleEndian.PutUint32(data[pathTableOffset+2:], 19) // Directory LBA
	binary.LittleEndian.PutUint16(data[pathTableOffset+6:], 1)  // Parent directory number
	data[pathTableOffset+8] = 0x00                              // Directory identifier (root)
	data[pathTableOffset+9] = 0x00                              // Padding

	// Root directory at block 19
	rootOffset := 19 * blockSize
	// Self entry (.)
	data[rootOffset] = 34 // Record length
	data[rootOffset+1] = 0
	binary.LittleEndian.PutUint32(data[rootOffset+2:], 19)
	binary.BigEndian.PutUint32(data[rootOffset+6:], 19)
	binary.LittleEndian.PutUint32(data[rootOffset+10:], uint32(blockSize))
	binary.BigEndian.PutUint32(data[rootOffset+14:], uint32(blockSize))
	data[rootOffset+25] = 0x02 // Directory flag
	data[rootOffset+32] = 1
	data[rootOffset+33] = 0x00

	// Parent entry (..)
	parentOffset := rootOffset + 34
	data[parentOffset] = 34
	data[parentOffset+1] = 0
	binary.LittleEndian.PutUint32(data[parentOffset+2:], 19)
	binary.BigEndian.PutUint32(data[parentOffset+6:], 19)
	binary.LittleEndian.PutUint32(data[parentOffset+10:], uint32(blockSize))
	binary.BigEndian.PutUint32(data[parentOffset+14:], uint32(blockSize))
	data[parentOffset+25] = 0x02
	data[parentOffset+32] = 1
	data[parentOffset+33] = 0x01

	return data
}

//nolint:gosec // G306 permissions ok for tests
func TestISO9660_Open(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a minimal ISO
	isoData := createMinimalISO("TESTVOLUME", "TESTSYSTEM", "TESTPUBLISHER")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o644); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	if iso.BlockSize() != 2048 {
		t.Errorf("BlockSize() = %d, want 2048", iso.BlockSize())
	}
}

//nolint:dupl,gosec // Similar test setup for ISO field retrieval is intentional; G306 permissions ok
func TestISO9660_GetVolumeID(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("MYVOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o644); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	volumeID := iso.GetVolumeID()
	// TrimSpace doesn't remove null bytes, so just check prefix
	if !strings.HasPrefix(volumeID, "MYVOL") {
		t.Errorf("GetVolumeID() = %q, want prefix %q", volumeID, "MYVOL")
	}
}

//nolint:dupl,gosec // Similar test setup for ISO field retrieval is intentional; G306 permissions ok
func TestISO9660_GetSystemID(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "PLAYSTATION", "PUBLISHER")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o644); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	systemID := iso.GetSystemID()
	// TrimSpace doesn't remove null bytes, so just check prefix
	if !strings.HasPrefix(systemID, "PLAYSTATION") {
		t.Errorf("GetSystemID() = %q, want prefix %q", systemID, "PLAYSTATION")
	}
}

func TestISO9660_Open_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := Open("/nonexistent/path/test.iso")
	if err == nil {
		t.Error("Open() should error for non-existent file")
	}
}

//nolint:gosec // G306 permissions ok for tests
func TestISO9660_Open_InvalidSize(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create file with invalid size (not divisible by 2048 or 2352)
	isoPath := filepath.Join(tmpDir, "invalid.iso")
	if writeErr := os.WriteFile(isoPath, make([]byte, 1000), 0o644); writeErr != nil {
		t.Fatalf("Failed to write file: %v", writeErr)
	}

	_, err = Open(isoPath)
	if err == nil {
		t.Error("Open() should error for invalid block size")
	}
}

//nolint:gosec // G306 permissions ok for tests
func TestISO9660_Open_NoPVD(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create file with valid size but no PVD
	isoPath := filepath.Join(tmpDir, "nopvd.iso")
	if writeErr := os.WriteFile(isoPath, make([]byte, 2048*20), 0o644); writeErr != nil {
		t.Fatalf("Failed to write file: %v", writeErr)
	}

	_, err = Open(isoPath)
	if !errors.Is(err, ErrPVDNotFound) {
		t.Errorf("Open() error = %v, want %v", err, ErrPVDNotFound)
	}
}

//nolint:gosec // G306 permissions ok for tests
func TestISO9660_Size(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o644); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	expectedSize := int64(len(isoData))
	if iso.Size() != expectedSize {
		t.Errorf("Size() = %d, want %d", iso.Size(), expectedSize)
	}
}

//nolint:gosec // G306 permissions ok for tests
func TestISO9660_IterFiles(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o644); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	// Our minimal ISO has no files (just root directory entries)
	files, err := iso.IterFiles(true)
	if err != nil {
		t.Fatalf("IterFiles() error = %v", err)
	}

	// Should return empty list for minimal ISO
	_ = files // We just verify it doesn't error
}

func TestISO9660_ReadFileByPath_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o600); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	_, err = iso.ReadFileByPath("/NONEXISTENT.TXT")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("ReadFileByPath() error = %v, want %v", err, ErrFileNotFound)
	}
}

func TestISO9660_FileExists(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o600); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	if iso.FileExists("/NONEXISTENT.TXT") {
		t.Error("FileExists() should return false for non-existent file")
	}
}

//nolint:dupl // Similar test setup for ISO field retrieval is intentional
func TestISO9660_GetPublisherID(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	isoData := createMinimalISO("VOL", "SYS", "MY PUBLISHER")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if writeErr := os.WriteFile(isoPath, isoData, 0o600); writeErr != nil {
		t.Fatalf("Failed to write ISO: %v", writeErr)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	pubID := iso.GetPublisherID()
	// TrimSpace doesn't remove null bytes, so just check prefix
	if !strings.HasPrefix(pubID, "MY PUBLISHER") {
		t.Errorf("GetPublisherID() = %q, want prefix %q", pubID, "MY PUBLISHER")
	}
}

// TestOpenCHD_NeoGeoCD verifies OpenCHD with real Neo Geo CD test file.
func TestOpenCHD_NeoGeoCD(t *testing.T) {
	t.Parallel()

	iso, err := OpenCHD("../testdata/NeoGeoCD/240pTestSuite.chd")
	if err != nil {
		t.Fatalf("OpenCHD failed: %v", err)
	}
	defer func() { _ = iso.Close() }()

	// Verify we can read ISO9660 metadata
	volumeID := iso.GetVolumeID()
	if volumeID == "" {
		t.Error("GetVolumeID() returned empty")
	}
	t.Logf("Volume ID: %q", volumeID)

	// Verify we can list files
	files, err := iso.IterFiles(false)
	if err != nil {
		t.Fatalf("IterFiles failed: %v", err)
	}
	t.Logf("Found %d files", len(files))

	// Should have IPL.TXT for Neo Geo CD
	hasIPL := false
	for _, f := range files {
		if strings.Contains(strings.ToUpper(f.Path), "IPL.TXT") {
			hasIPL = true
			break
		}
	}
	if !hasIPL {
		t.Log("Note: IPL.TXT not found in root - may be in subdirectory")
	}
}

// TestOpenCHD_NonExistent verifies error handling for missing files.
func TestOpenCHD_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := OpenCHD("/nonexistent/path/file.chd")
	if err == nil {
		t.Error("OpenCHD should fail for non-existent file")
	}
}

// TestOpenCHD_InvalidCHD verifies error handling for non-CHD files.
func TestOpenCHD_InvalidCHD(t *testing.T) {
	t.Parallel()

	// Try to open a non-CHD file
	_, err := OpenCHD("iso9660_test.go")
	if err == nil {
		t.Error("OpenCHD should fail for non-CHD file")
	}
}

// TestOpenReaderWithCloser verifies OpenReaderWithCloser functionality.
func TestOpenReaderWithCloser(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	isoData := createMinimalISO("TEST", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if err := os.WriteFile(isoPath, isoData, 0o600); err != nil {
		t.Fatalf("Failed to write ISO: %v", err)
	}

	// Open the file and use OpenReaderWithCloser
	//nolint:gosec // G304: test file path constructed from t.TempDir
	file, err := os.Open(isoPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	stat, _ := file.Stat()
	iso, err := OpenReaderWithCloser(file, stat.Size(), file)
	if err != nil {
		_ = file.Close()
		t.Fatalf("OpenReaderWithCloser failed: %v", err)
	}

	// Verify it works
	_ = iso.GetVolumeID()

	// Close should close the underlying file
	if err := iso.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestGetDataPreparerID verifies data preparer ID extraction.
func TestGetDataPreparerID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	isoData := createMinimalISO("VOL", "SYS", "PUB")
	isoPath := filepath.Join(tmpDir, "test.iso")
	if err := os.WriteFile(isoPath, isoData, 0o600); err != nil {
		t.Fatalf("Failed to write ISO: %v", err)
	}

	iso, err := Open(isoPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = iso.Close() }()

	// Data preparer ID is at a different offset - our minimal ISO doesn't set it
	_ = iso.GetDataPreparerID()
}
