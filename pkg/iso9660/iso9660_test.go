package iso9660

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSectorSize(t *testing.T) {
	tests := []struct {
		name         string
		size         int64
		expectedSize int
		expectError  bool
	}{
		{"2048 byte sectors", 2048 * 100, 2048, false},
		{"2352 byte sectors", 2352 * 100, 2352, false},
		{"Invalid size", 2049 * 100, 0, true},
		{"Zero size", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := detectSectorSize(tt.size)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if size != tt.expectedSize {
					t.Errorf("Expected sector size %d, got %d", tt.expectedSize, size)
				}
			}
		})
	}
}

func TestPrimaryVolumeDescriptor(t *testing.T) {
	// Create a minimal valid PVD
	pvd := make([]byte, 2048)

	// Type code (1 = Primary Volume Descriptor)
	pvd[0] = 1

	// Standard identifier "CD001"
	copy(pvd[1:6], []byte("CD001"))

	// Version
	pvd[6] = 1

	// System identifier (offset 8, length 32)
	copy(pvd[8:40], []byte("TEST_SYSTEM_ID"))

	// Volume identifier (offset 40, length 32)
	copy(pvd[40:72], []byte("TEST_VOLUME"))

	// Volume space size (offset 80, both-endian 32-bit)
	// Little-endian at 80
	pvd[80] = 0x00
	pvd[81] = 0x10
	pvd[82] = 0x00
	pvd[83] = 0x00
	// Big-endian at 84
	pvd[84] = 0x00
	pvd[85] = 0x00
	pvd[86] = 0x10
	pvd[87] = 0x00

	// Publisher identifier (offset 318, length 128)
	copy(pvd[318:446], []byte("TEST_PUBLISHER"))

	// Data preparer identifier (offset 446, length 128)
	copy(pvd[446:574], []byte("TEST_PREPARER"))

	// Root directory record (offset 156, length 34)
	// Directory record length
	pvd[156] = 34
	// Extended attribute record length
	pvd[157] = 0
	// Location of extent (LBA) - little-endian
	pvd[158] = 0x15
	pvd[159] = 0x00
	pvd[160] = 0x00
	pvd[161] = 0x00
	// Location of extent (LBA) - big-endian
	pvd[162] = 0x00
	pvd[163] = 0x00
	pvd[164] = 0x00
	pvd[165] = 0x15
	// Data length - little-endian
	pvd[166] = 0x00
	pvd[167] = 0x08
	pvd[168] = 0x00
	pvd[169] = 0x00
	// Data length - big-endian
	pvd[170] = 0x00
	pvd[171] = 0x00
	pvd[172] = 0x08
	pvd[173] = 0x00

	// Creation date/time (offset 813, length 17)
	copy(pvd[813:830], []byte("2024010112000000"))

	descriptor := parsePrimaryVolumeDescriptor(pvd)

	tests := []struct {
		field    string
		expected string
	}{
		{"SystemID", "TEST_SYSTEM_ID"},
		{"VolumeID", "TEST_VOLUME"},
		{"PublisherID", "TEST_PUBLISHER"},
		{"DataPreparerID", "TEST_PREPARER"},
		{"UUID", "2024-01-01-12-00-00-00"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			var actual string
			switch tt.field {
			case "SystemID":
				actual = descriptor.SystemID
			case "VolumeID":
				actual = descriptor.VolumeID
			case "PublisherID":
				actual = descriptor.PublisherID
			case "DataPreparerID":
				actual = descriptor.DataPreparerID
			case "UUID":
				actual = descriptor.CreationDateTime
			}

			if actual != tt.expected {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.field, tt.expected, actual)
			}
		})
	}

	// Test root directory info
	if descriptor.RootDirLBA != 0x15 {
		t.Errorf("Expected RootDirLBA to be 0x15, got 0x%x", descriptor.RootDirLBA)
	}
	if descriptor.RootDirSize != 0x800 {
		t.Errorf("Expected RootDirSize to be 0x800, got 0x%x", descriptor.RootDirSize)
	}
}

func TestISO9660_Open(t *testing.T) {
	// Create a minimal ISO image in memory
	// For 2048-byte sectors
	isoData := createMinimalISO(2048)

	reader := bytes.NewReader(isoData)

	iso, err := Open(reader, int64(len(isoData)))
	if err != nil {
		t.Fatalf("Failed to open ISO: %v", err)
	}

	if iso.SectorSize != 2048 {
		t.Errorf("Expected sector size 2048, got %d", iso.SectorSize)
	}

	if iso.PVD.VolumeID != "TEST_ISO" {
		t.Errorf("Expected volume ID 'TEST_ISO', got '%s'", iso.PVD.VolumeID)
	}
}

func TestISO9660_ListFiles(t *testing.T) {
	// This test will be expanded once we implement file listing
	isoData := createMinimalISO(2048)
	reader := bytes.NewReader(isoData)

	iso, err := Open(reader, int64(len(isoData)))
	if err != nil {
		t.Fatalf("Failed to open ISO: %v", err)
	}

	files, err := iso.ListFiles(true)
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	// For now, just check that we can call the method
	if files == nil {
		t.Error("Expected files list to be non-nil")
	}
}

// comprehensive comparison test with original python script
func TestComparisonWithOriginalScript(t *testing.T) {
	// skip if no sample files available
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		t.Skip("testdata directory not found")
	}

	// define test cases with sample files
	testCases := []struct {
		console  string
		filepath string
		expected map[string]string // expected outputs from python script
	}{
		// add test cases here once we have sample files
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.console, filepath.Base(tc.filepath)), func(t *testing.T) {
			// test go implementation
			goResult := runGoGameID(t, tc.console, tc.filepath)

			// test python implementation
			pythonResult := runPythonGameID(t, tc.console, tc.filepath)

			// compare results
			compareResults(t, goResult, pythonResult)
		})
	}
}

// helper to run go implementation
func runGoGameID(t *testing.T, console, filepath string) map[string]string {
	// implementation would call our go gameid
	return nil
}

// helper to run python implementation
func runPythonGameID(t *testing.T, console, filepath string) map[string]string {
	// implementation would call original python script
	return nil
}

// helper to compare results
func compareResults(t *testing.T, goResult, pythonResult map[string]string) {
	// compare the two results and report differences
}

// Helper function to create a minimal valid ISO image
func createMinimalISO(sectorSize int) []byte {
	// Calculate total size: need at least 19 sectors
	// (16 for system area + 1 for PVD + 1 for terminator + 1 for root dir)
	totalSize := 19 * sectorSize
	data := make([]byte, totalSize)

	// Create Primary Volume Descriptor at sector 16
	pvdOffset := 16 * sectorSize

	// If sector size is 2352, we need to account for the header
	if sectorSize == 2352 {
		pvdOffset += 0x18
	}

	// Type code
	data[pvdOffset] = 1

	// Standard identifier
	copy(data[pvdOffset+1:pvdOffset+6], []byte("CD001"))

	// Version
	data[pvdOffset+6] = 1

	// Volume ID
	copy(data[pvdOffset+40:pvdOffset+72], []byte("TEST_ISO"))

	// Add root directory record at sector 18
	rootDirLBA := uint32(18)
	rootDirSize := uint32(sectorSize) // One sector for root dir

	// Root directory record in PVD (offset 156)
	data[pvdOffset+156] = 34 // Directory record length

	// Root directory LBA (little-endian then big-endian)
	binary.LittleEndian.PutUint32(data[pvdOffset+158:], rootDirLBA)
	binary.BigEndian.PutUint32(data[pvdOffset+162:], rootDirLBA)

	// Root directory size (little-endian then big-endian)
	binary.LittleEndian.PutUint32(data[pvdOffset+166:], rootDirSize)
	binary.BigEndian.PutUint32(data[pvdOffset+170:], rootDirSize)

	// Create actual root directory at sector 18
	rootDirOffset := 18 * sectorSize
	if sectorSize == 2352 {
		rootDirOffset += 0x18
	}

	// First entry: current directory (.)
	data[rootDirOffset] = 34                   // Record length
	data[rootDirOffset+2] = byte(rootDirLBA)   // LBA (simplified)
	data[rootDirOffset+10] = byte(rootDirSize) // Size (simplified)
	data[rootDirOffset+25] = 0x02              // Directory flag
	data[rootDirOffset+32] = 1                 // Name length
	data[rootDirOffset+33] = 0x00              // Name: current dir

	// Second entry: parent directory (..)
	secondOffset := rootDirOffset + 34
	data[secondOffset] = 34                   // Record length
	data[secondOffset+2] = byte(rootDirLBA)   // LBA
	data[secondOffset+10] = byte(rootDirSize) // Size
	data[secondOffset+25] = 0x02              // Directory flag
	data[secondOffset+32] = 1                 // Name length
	data[secondOffset+33] = 0x01              // Name: parent dir

	// Add a test file entry
	fileOffset := secondOffset + 34
	data[fileOffset] = 40      // Record length
	data[fileOffset+2] = 19    // File LBA (sector 19)
	data[fileOffset+10] = 100  // File size (100 bytes)
	data[fileOffset+25] = 0x00 // File flag (not a directory)
	data[fileOffset+32] = 8    // Name length
	data[fileOffset+33] = 'T'
	data[fileOffset+34] = 'E'
	data[fileOffset+35] = 'S'
	data[fileOffset+36] = 'T'
	data[fileOffset+37] = '.'
	data[fileOffset+38] = 'T'
	data[fileOffset+39] = 'X'
	data[fileOffset+40] = 'T'

	// Volume Descriptor Set Terminator at sector 17
	termOffset := 17 * sectorSize
	if sectorSize == 2352 {
		termOffset += 0x18
	}

	if termOffset+6 <= len(data) {
		data[termOffset] = 0xFF
		copy(data[termOffset+1:termOffset+6], []byte("CD001"))
		data[termOffset+6] = 1
	}

	return data
}
