package identifiers

import (
	"encoding/binary"
	"testing"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
)

func TestPSPIdentifier_Console(t *testing.T) {
	identifier := NewPSPIdentifier(nil)
	if identifier.Console() != "PSP" {
		t.Errorf("Expected console 'PSP', got %s", identifier.Console())
	}
}

func TestPSPIdentifier_Identify(t *testing.T) {
	tests := []struct {
		name        string
		umdData     string
		expected    map[string]string
		expectError bool
		noUMDFile   bool
	}{
		{
			name:    "Valid PSP ISO",
			umdData: "ULUS-10041|12345678901234567890",
			expected: map[string]string{
				"ID":    "ULUS-10041",
				"title": "ULUS-10041", // No database match, so ID becomes title
			},
		},
		{
			name:    "PSP with database match",
			umdData: "UCUS-98612|FF7CC",
			expected: map[string]string{
				"ID":        "UCUS-98612",
				"title":     "Crisis Core Test",
				"developer": "Square Enix",
			},
		},
		{
			name:    "PSP with no pipe delimiter",
			umdData: "NPJH-50148",
			expected: map[string]string{
				"ID":    "NPJH-50148",
				"title": "NPJH-50148",
			},
		},
		{
			name:    "PSP with extra data after pipe",
			umdData: "ULES-00151|0001|MORE|DATA",
			expected: map[string]string{
				"ID":    "ULES-00151",
				"title": "ULES-00151",
			},
		},
		{
			name:        "No UMD_DATA.BIN file",
			noUMDFile:   true,
			expectError: true,
		},
		{
			name:        "Empty UMD_DATA.BIN",
			umdData:     "",
			expectError: true,
		},
	}

	// Create test database
	db := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"PSP": {
				"UCUS-98612": {
					"title":     "Crisis Core Test",
					"developer": "Square Enix",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test ISO with UMD_DATA.BIN
			var isoData []byte
			if !tt.noUMDFile {
				isoData = createPSPISO(tt.umdData)
			} else {
				isoData = createMinimalISO(2048) // ISO without UMD_DATA.BIN
			}

			// Write test data to a temporary file
			tmpFile := t.TempDir() + "/test.iso"
			if err := writeTestFile(tmpFile, isoData); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			identifier := NewPSPIdentifier(db)
			result, err := identifier.Identify(tmpFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check all expected fields
			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("Field %s: expected '%s', got '%s'", key, expectedValue, result[key])
				}
			}
		})
	}
}

// Helper function to create a PSP ISO with UMD_DATA.BIN
func createPSPISO(umdContent string) []byte {
	// We'll reuse the createPSXISO function but modify it for PSP
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
	copy(data[pvdOffset+8:pvdOffset+40], []byte("PSP GAME"))

	// Volume ID
	copy(data[pvdOffset+40:pvdOffset+72], []byte("PSP"))

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

	// Add UMD_DATA.BIN file
	fileOffset := secondOffset + 34
	fileName := "UMD_DATA.BIN"
	nameBytes := []byte(fileName)
	recordLen := 33 + len(nameBytes)
	if recordLen%2 == 1 {
		recordLen++ // Padding
	}

	data[fileOffset] = byte(recordLen)
	data[fileOffset+2] = 19 // File LBA

	// File size is the length of UMD content
	fileSize := uint32(len(umdContent))
	data[fileOffset+10] = byte(fileSize)

	data[fileOffset+25] = 0x00 // File flag
	data[fileOffset+32] = byte(len(nameBytes))
	copy(data[fileOffset+33:], nameBytes)

	// Write UMD_DATA.BIN content at sector 19
	umdOffset := 19 * sectorSize
	copy(data[umdOffset:], []byte(umdContent))

	// Volume Descriptor Set Terminator at sector 17
	termOffset := 17 * sectorSize
	data[termOffset] = 0xFF
	copy(data[termOffset+1:termOffset+6], []byte("CD001"))
	data[termOffset+6] = 1

	return data
}

// Helper function to create a minimal ISO without UMD_DATA.BIN
func createMinimalISO(sectorSize int) []byte {
	totalSize := 19 * sectorSize
	data := make([]byte, totalSize)

	// Create Primary Volume Descriptor at sector 16
	pvdOffset := 16 * sectorSize

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
	rootDirSize := uint32(sectorSize)

	// Root directory record in PVD
	data[pvdOffset+156] = 34
	binary.LittleEndian.PutUint32(data[pvdOffset+158:], rootDirLBA)
	binary.BigEndian.PutUint32(data[pvdOffset+162:], rootDirLBA)
	binary.LittleEndian.PutUint32(data[pvdOffset+166:], rootDirSize)
	binary.BigEndian.PutUint32(data[pvdOffset+170:], rootDirSize)

	// Volume Descriptor Set Terminator at sector 17
	termOffset := 17 * sectorSize
	data[termOffset] = 0xFF
	copy(data[termOffset+1:termOffset+6], []byte("CD001"))
	data[termOffset+6] = 1

	return data
}
