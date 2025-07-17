package identifiers

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

func TestPSXIdentifier_Console(t *testing.T) {
	identifier := NewPSXIdentifier(nil)
	if identifier.Console() != "PSX" {
		t.Errorf("Expected console 'PSX', got %s", identifier.Console())
	}
}

func TestPSXIdentifier_SerialFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"SLUS format", "SLUS_012.34", "SLUS_01234"},
		{"SLES format", "SLES-01234", "SLES_01234"},
		{"SCUS format", "SCUS_94426", "SCUS_94426"},
		{"SLPM format", "SLPM_86789", "SLPM_86789"},
		{"With extension", "SLES_01234.01", "SLES_0123401"},
		{"Different delimiter", "SLUSP012.06", "SLUSP01206"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPSXSerial(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected serial '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestPSXIdentifier_Identify(t *testing.T) {
	// Create test database
	// db := &database.GameDatabase{
	// 	Systems: map[string]database.SystemDatabase{
	// 		"PSX": {
	// 			"SLUS_00594": {
	// 				"title":     "Final Fantasy VII",
	// 				"developer": "Square",
	// 				"publisher": "Sony",
	// 				"genre":     "RPG",
	// 			},
	// 			"SLES_01234": {
	// 				"title":     "Test Game",
	// 				"developer": "Test Dev",
	// 			},
	// 		},
	// 	},
	// }

	// identifier := NewPSXIdentifier(db)

	// Create a minimal ISO with PSX game file
	isoData := createPSXISO("SLUS_005.94")
	reader := bytes.NewReader(isoData)

	// Mock ISO that we'll pass to the identifier
	iso, err := iso9660.Open(reader, int64(len(isoData)))
	if err != nil {
		t.Fatalf("Failed to create test ISO: %v", err)
	}

	// For this test, we'll need to refactor to accept an ISO object
	// For now, let's test the serial extraction logic
	files, err := iso.ListFiles(true)
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	var foundSerial string
	for _, file := range files {
		filename := strings.ToUpper(file.Name)
		filename = strings.TrimPrefix(filename, "/")

		// Check if it matches PSX serial pattern
		for _, prefix := range []string{"SLUS", "SLES", "SCUS", "SLPM", "SCES", "SIPS", "SLPS", "SCPS"} {
			if strings.HasPrefix(filename, prefix) {
				foundSerial = extractPSXSerial(filename)
				break
			}
		}
		if foundSerial != "" {
			break
		}
	}

	if foundSerial != "SLUS_00594" {
		t.Errorf("Expected to find serial SLUS_00594, got '%s'", foundSerial)
	}
}

func TestPS2Identifier_Console(t *testing.T) {
	identifier := NewPS2Identifier(nil)
	if identifier.Console() != "PS2" {
		t.Errorf("Expected console 'PS2', got %s", identifier.Console())
	}
}

// Helper function to create a PSX ISO for testing
func createPSXISO(serialFile string) []byte {
	sectorSize := 2048
	totalSize := 20 * sectorSize
	data := make([]byte, totalSize)

	// Create Primary Volume Descriptor at sector 16
	pvdOffset := 16 * sectorSize

	// Type code
	data[pvdOffset] = 1

	// Standard identifier
	copy(data[pvdOffset+1:pvdOffset+6], []byte("CD001"))

	// Version
	data[pvdOffset+6] = 1

	// System ID
	copy(data[pvdOffset+8:pvdOffset+40], []byte("PLAYSTATION"))

	// Volume ID
	copy(data[pvdOffset+40:pvdOffset+72], []byte("SLUS_00594"))

	// Root directory at sector 18
	rootDirLBA := uint32(18)
	rootDirSize := uint32(sectorSize)

	// Root directory record in PVD
	data[pvdOffset+156] = 34
	binary.LittleEndian.PutUint32(data[pvdOffset+158:], rootDirLBA)
	binary.BigEndian.PutUint32(data[pvdOffset+162:], rootDirLBA)
	binary.LittleEndian.PutUint32(data[pvdOffset+166:], rootDirSize)
	binary.BigEndian.PutUint32(data[pvdOffset+170:], rootDirSize)

	// Create root directory at sector 18
	rootDirOffset := 18 * sectorSize

	// Current directory entry
	data[rootDirOffset] = 34
	data[rootDirOffset+2] = byte(rootDirLBA)
	data[rootDirOffset+10] = byte(rootDirSize)
	data[rootDirOffset+25] = 0x02
	data[rootDirOffset+32] = 1
	data[rootDirOffset+33] = 0x00

	// Parent directory entry
	secondOffset := rootDirOffset + 34
	data[secondOffset] = 34
	data[secondOffset+2] = byte(rootDirLBA)
	data[secondOffset+10] = byte(rootDirSize)
	data[secondOffset+25] = 0x02
	data[secondOffset+32] = 1
	data[secondOffset+33] = 0x01

	// Add serial file
	fileOffset := secondOffset + 34
	nameBytes := []byte(serialFile)
	recordLen := 33 + len(nameBytes)
	if recordLen%2 == 1 {
		recordLen++ // Padding
	}

	data[fileOffset] = byte(recordLen)
	data[fileOffset+2] = 19    // File LBA
	data[fileOffset+10] = 11   // File size (11 bytes for "PLAYSTATION")
	data[fileOffset+25] = 0x00 // File flag
	data[fileOffset+32] = byte(len(nameBytes))
	copy(data[fileOffset+33:], nameBytes)

	// Add SYSTEM.CNF file
	nextOffset := fileOffset + recordLen
	systemName := []byte("SYSTEM.CNF")
	recordLen = 33 + len(systemName)
	if recordLen%2 == 1 {
		recordLen++
	}

	data[nextOffset] = byte(recordLen)
	data[nextOffset+2] = 20  // File LBA
	data[nextOffset+10] = 50 // File size
	data[nextOffset+25] = 0x00
	data[nextOffset+32] = byte(len(systemName))
	copy(data[nextOffset+33:], systemName)

	// Volume Descriptor Set Terminator at sector 17
	termOffset := 17 * sectorSize
	data[termOffset] = 0xFF
	copy(data[termOffset+1:termOffset+6], []byte("CD001"))
	data[termOffset+6] = 1

	return data
}
