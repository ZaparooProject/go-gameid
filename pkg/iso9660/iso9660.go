package iso9660

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	// Standard sector sizes for CD-ROMs
	SectorSize2048 = 2048
	SectorSize2352 = 2352

	// Offsets for 2352-byte sectors
	SectorHeader2352 = 0x18

	// Primary Volume Descriptor location (sector 16)
	PVDSector = 16

	// Volume descriptor types
	VolumeTypePrimary    = 1
	VolumeTypeTerminator = 255
)

// ISO9660 represents an ISO 9660 disc image
type ISO9660 struct {
	reader       io.ReaderAt
	size         int64
	SectorSize   int
	SectorOffset int
	PVD          *PrimaryVolumeDescriptor
	closer       io.Closer // For cleanup when using files
}

// PrimaryVolumeDescriptor contains parsed PVD data
type PrimaryVolumeDescriptor struct {
	SystemID         string
	VolumeID         string
	VolumeSpaceSize  uint32
	PublisherID      string
	DataPreparerID   string
	CreationDateTime string
	RootDirLBA       uint32
	RootDirSize      uint32
}

// FileEntry represents a file in the ISO
type FileEntry struct {
	Name string
	LBA  uint32
	Size uint32
}

// Open opens an ISO 9660 image
func Open(r io.ReaderAt, size int64) (*ISO9660, error) {
	iso := &ISO9660{
		reader: r,
		size:   size,
	}

	// Detect sector size
	sectorSize, err := detectSectorSize(size)
	if err != nil {
		return nil, fmt.Errorf("failed to detect sector size: %w", err)
	}
	iso.SectorSize = sectorSize

	// Set sector offset for 2352-byte sectors
	if sectorSize == SectorSize2352 {
		iso.SectorOffset = SectorHeader2352
	} else {
		iso.SectorOffset = 0
	}

	// Read Primary Volume Descriptor
	pvdData := make([]byte, SectorSize2048)
	pvdOffset := int64(PVDSector*iso.SectorSize + iso.SectorOffset)

	if _, err := r.ReadAt(pvdData, pvdOffset); err != nil {
		return nil, fmt.Errorf("failed to read PVD: %w", err)
	}

	// Verify it's a valid PVD
	if pvdData[0] != VolumeTypePrimary {
		return nil, fmt.Errorf("invalid PVD type: %d", pvdData[0])
	}

	if string(pvdData[1:6]) != "CD001" {
		return nil, fmt.Errorf("invalid PVD signature")
	}

	iso.PVD = parsePrimaryVolumeDescriptor(pvdData)

	return iso, nil
}

// detectSectorSize determines the sector size based on file size
func detectSectorSize(size int64) (int, error) {
	if size == 0 {
		return 0, fmt.Errorf("empty file")
	}

	if size%SectorSize2352 == 0 {
		return SectorSize2352, nil
	} else if size%SectorSize2048 == 0 {
		return SectorSize2048, nil
	}

	return 0, fmt.Errorf("invalid disc image size: %d", size)
}

// parsePrimaryVolumeDescriptor parses PVD data
func parsePrimaryVolumeDescriptor(data []byte) *PrimaryVolumeDescriptor {
	pvd := &PrimaryVolumeDescriptor{}

	// System identifier (offset 8, length 32)
	pvd.SystemID = cleanISOString(data[8:40])

	// Volume identifier (offset 40, length 32)
	pvd.VolumeID = cleanISOString(data[40:72])

	// Volume space size (offset 80, little-endian)
	pvd.VolumeSpaceSize = binary.LittleEndian.Uint32(data[80:84])

	// Publisher identifier (offset 318, length 128)
	pvd.PublisherID = cleanISOString(data[318:446])

	// Data preparer identifier (offset 446, length 128)
	pvd.DataPreparerID = cleanISOString(data[446:574])

	// Creation date/time (usually at offset 813, but we need to search for it)
	pvd.CreationDateTime = extractCreationDateTime(data)

	// Root directory record (offset 156)
	// Location of extent (LBA) - little-endian at offset 2
	pvd.RootDirLBA = binary.LittleEndian.Uint32(data[158:162])

	// Data length - little-endian at offset 10
	pvd.RootDirSize = binary.LittleEndian.Uint32(data[166:170])

	return pvd
}

// cleanISOString cleans up ISO 9660 strings
func cleanISOString(data []byte) string {
	// Convert to string and trim spaces
	s := string(data)
	s = strings.TrimSpace(s)

	// Remove any non-printable characters
	result := ""
	for _, r := range s {
		if r >= 32 && r <= 126 {
			result += string(r)
		}
	}

	return result
}

// extractCreationDateTime extracts the creation date/time from PVD data
func extractCreationDateTime(data []byte) string {
	// ISO 9660 specifies Creation Date/Time at offset 813, length 17 bytes.
	// The format is YYYYMMDDHHMMSSFF (FF = fractional seconds, usually 00)
	// followed by a single byte for timezone offset.
	const uuidOffset = 813
	const uuidLength = 17 // YYYYMMDDHHMMSSFF + 1 byte for timezone

	if uuidOffset+uuidLength > len(data) {
		return "" // Not enough data for UUID
	}

	rawUUIDBytes := data[uuidOffset : uuidOffset+uuidLength]

	// Check if the first 16 bytes (date/time part) are all null bytes
	allNulls := true
	for i := 0; i < 16 && i < len(rawUUIDBytes); i++ {
		if rawUUIDBytes[i] != 0x00 {
			allNulls = false
			break
		}
	}
	if allNulls {
		// Python formats null bytes as well, e.g., "\x00\x00\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00"
		return "\x00\x00\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00-\x00\x00"
	}

	// Clean and format the date/time string (first 16 bytes)
	// cleanISOString removes non-printable characters, including nulls.
	uuidStr := cleanISOString(rawUUIDBytes[:16])

	// If after cleaning, it's too short to be a valid date string,
	// return the cleaned string as is. This handles cases where it's
	// not a valid date string but not all nulls.
	if len(uuidStr) < 14 {
		return uuidStr
	}

	// Format as YYYY-MM-DD-HH-MM-SS
	formatted := uuidStr[:4] + "-" +
		uuidStr[4:6] + "-" +
		uuidStr[6:8] + "-" +
		uuidStr[8:10] + "-" +
		uuidStr[10:12] + "-" +
		uuidStr[12:14]

	// Add fractional seconds if available and valid
	if len(uuidStr) >= 16 {
		formatted += "-" + uuidStr[14:16]
	}

	return formatted
}

// ListFiles lists files in the root directory
func (iso *ISO9660) ListFiles(onlyRootDir bool) ([]FileEntry, error) {
	if iso.PVD == nil {
		return nil, fmt.Errorf("no PVD loaded")
	}

	files := []FileEntry{}

	// Read root directory
	dirData := make([]byte, iso.PVD.RootDirSize)
	dirOffset := int64(iso.PVD.RootDirLBA*uint32(iso.SectorSize) + uint32(iso.SectorOffset))

	if _, err := iso.reader.ReadAt(dirData, dirOffset); err != nil {
		return nil, fmt.Errorf("failed to read root directory: %w", err)
	}

	// Parse directory entries
	i := 0
	for i < len(dirData) {
		// Directory record length
		recLen := dirData[i]
		if recLen == 0 {
			break
		}

		// Extended attribute record length
		// extAttrLen := dirData[i+1]

		// Location of extent (LBA) - little-endian
		lba := binary.LittleEndian.Uint32(dirData[i+2 : i+6])

		// Data length - little-endian
		dataLen := binary.LittleEndian.Uint32(dirData[i+10 : i+14])

		// File flags
		fileFlags := dirData[i+25]
		isDir := (fileFlags & 0x02) != 0

		// File identifier length
		nameLen := dirData[i+32]

		// File identifier
		if i+33+int(nameLen) > len(dirData) {
			break
		}

		name := dirData[i+33 : i+33+int(nameLen)]

		// Skip special entries (. and ..)
		if len(name) == 1 && (name[0] == 0x00 || name[0] == 0x01) {
			i += int(recLen)
			continue
		}

		// Parse filename
		filename := string(name)

		// Remove version suffix (;1)
		if idx := strings.Index(filename, ";"); idx > 0 {
			filename = filename[:idx]
		}

		// Add to files list if not a directory
		if !isDir {
			files = append(files, FileEntry{
				Name: "/" + filename,
				LBA:  lba,
				Size: dataLen,
			})
		} else if !onlyRootDir {
			// TODO: Implement recursive directory listing
			return nil, fmt.Errorf("recursive directory listing not yet implemented")
		}

		i += int(recLen)
	}

	return files, nil
}

// ReadFile reads a file from the ISO by LBA and size
func (iso *ISO9660) ReadFile(lba, size uint32) ([]byte, error) {
	data := make([]byte, size)
	offset := int64(lba*uint32(iso.SectorSize) + uint32(iso.SectorOffset))

	if _, err := iso.reader.ReadAt(data, offset); err != nil {
		return nil, fmt.Errorf("failed to read file at LBA %d: %w", lba, err)
	}

	return data, nil
}

// ReadFileByEntry reads a file using its FileEntry
func (iso *ISO9660) ReadFileByEntry(entry *FileEntry) ([]byte, error) {
	return iso.ReadFile(entry.LBA, entry.Size)
}
