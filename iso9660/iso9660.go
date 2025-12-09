// Package iso9660 provides parsing for ISO9660 disc images.
package iso9660

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Common errors
var (
	ErrInvalidISO   = errors.New("invalid ISO9660 image")
	ErrPVDNotFound  = errors.New("primary volume descriptor not found")
	ErrInvalidBlock = errors.New("invalid block size")
	ErrFileNotFound = errors.New("file not found")
)

// PVD magic word: 0x01 followed by "CD001"
var pvdMagicWord = []byte{0x01, 'C', 'D', '0', '0', '1'}

// FileInfo contains information about a file in the ISO filesystem.
type FileInfo struct {
	Path string
	LBA  uint32 // Logical Block Address
	Size uint32
}

// PathTableEntry represents an entry in the ISO9660 path table.
type pathTableEntry struct {
	name      string
	lba       uint32
	parentIdx int // -1 for root
}

// ISO9660 represents a parsed ISO9660 disc image.
type ISO9660 struct {
	file        *os.File
	blockSize   int
	blockOffset int64
	pvd         []byte
	pathTable   []pathTableEntry
	size        int64
}

// Open opens an ISO9660 disc image from a file.
func Open(path string) (*ISO9660, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	iso := &ISO9660{
		file: f,
		size: info.Size(),
	}

	if err := iso.init(); err != nil {
		f.Close()
		return nil, err
	}

	return iso, nil
}

// OpenReader creates an ISO9660 from an io.ReaderAt.
// The caller is responsible for closing the underlying reader if needed.
func OpenReader(r io.ReaderAt, size int64) (*ISO9660, error) {
	iso := &ISO9660{
		size: size,
	}

	// Create a wrapper that implements the file interface we need
	if f, ok := r.(*os.File); ok {
		iso.file = f
	} else {
		// For non-file readers, we need a different approach
		return nil, errors.New("OpenReader currently only supports *os.File")
	}

	if err := iso.init(); err != nil {
		return nil, err
	}

	return iso, nil
}

// init initializes the ISO9660 structure by finding PVD and parsing path table.
func (iso *ISO9660) init() error {
	// Determine block size from file size
	if iso.size%2352 == 0 {
		iso.blockSize = 2352
	} else if iso.size%2048 == 0 {
		iso.blockSize = 2048
	} else {
		return ErrInvalidBlock
	}

	// Search for PVD in first ~1MB
	searchSize := int64(1000000)
	if searchSize > iso.size {
		searchSize = iso.size
	}

	header := make([]byte, searchSize)
	if _, err := iso.file.ReadAt(header, 0); err != nil && err != io.EOF {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Find PVD magic word
	pvdOffset := int64(-1)
	for i := 0; i <= len(header)-len(pvdMagicWord); i++ {
		match := true
		for j, b := range pvdMagicWord {
			if header[i+j] != b {
				match = false
				break
			}
		}
		if match {
			pvdOffset = int64(i)
			break
		}
	}

	if pvdOffset == -1 {
		return ErrPVDNotFound
	}

	// Calculate block offset (PVD should be at block 16)
	iso.blockOffset = pvdOffset - int64(16*iso.blockSize)

	// Read PVD (one block)
	iso.pvd = make([]byte, iso.blockSize)
	if _, err := iso.file.ReadAt(iso.pvd, pvdOffset); err != nil {
		return fmt.Errorf("failed to read PVD: %w", err)
	}

	// Parse path table
	if err := iso.parsePathTable(); err != nil {
		return fmt.Errorf("failed to parse path table: %w", err)
	}

	return nil
}

// parsePathTable parses the ISO9660 path table.
func (iso *ISO9660) parsePathTable() error {
	// Path table size at offset 132 (little-endian)
	pathTableSize := binary.LittleEndian.Uint32(iso.pvd[132:136])
	// Path table LBA at offset 140 (little-endian)
	pathTableLBA := binary.LittleEndian.Uint32(iso.pvd[140:144])

	// Read path table
	offset := iso.blockOffset + int64(pathTableLBA)*int64(iso.blockSize)
	pathTableRaw := make([]byte, pathTableSize)
	if _, err := iso.file.ReadAt(pathTableRaw, offset); err != nil {
		return fmt.Errorf("failed to read path table: %w", err)
	}

	// Parse path table entries
	iso.pathTable = nil
	i := 0
	for i < len(pathTableRaw) {
		if i >= len(pathTableRaw) {
			break
		}

		dirNameLen := int(pathTableRaw[i])
		if dirNameLen == 0 {
			break
		}

		// Extended attribute record length at i+1 (skip)
		dirLBA := binary.LittleEndian.Uint32(pathTableRaw[i+2 : i+6])
		dirParentIdx := int(binary.LittleEndian.Uint16(pathTableRaw[i+6:i+8])) - 1

		dirName := string(pathTableRaw[i+8 : i+8+dirNameLen])
		if dirName == "\x00" {
			dirName = ""
			dirParentIdx = -1 // Root
		}

		iso.pathTable = append(iso.pathTable, pathTableEntry{
			name:      dirName + "/",
			lba:       dirLBA,
			parentIdx: dirParentIdx,
		})

		// Move to next entry (8 + name length, padded to even)
		i += 8 + dirNameLen
		if i%2 == 1 {
			i++
		}
	}

	return nil
}

// Close closes the ISO9660 file.
func (iso *ISO9660) Close() error {
	if iso.file != nil {
		return iso.file.Close()
	}
	return nil
}

// GetSystemID returns the system identifier from the PVD.
func (iso *ISO9660) GetSystemID() string {
	if len(iso.pvd) < 40 {
		return ""
	}
	return strings.TrimSpace(string(iso.pvd[8:40]))
}

// GetVolumeID returns the volume identifier from the PVD.
func (iso *ISO9660) GetVolumeID() string {
	if len(iso.pvd) < 72 {
		return ""
	}
	return strings.TrimSpace(string(iso.pvd[40:72]))
}

// GetPublisherID returns the publisher identifier from the PVD.
func (iso *ISO9660) GetPublisherID() string {
	if len(iso.pvd) < 446 {
		return ""
	}
	return strings.TrimSpace(string(iso.pvd[318:446]))
}

// GetDataPreparerID returns the data preparer identifier from the PVD.
func (iso *ISO9660) GetDataPreparerID() string {
	if len(iso.pvd) < 574 {
		return ""
	}
	return strings.TrimSpace(string(iso.pvd[446:574]))
}

// GetUUID returns a unique identifier derived from disc metadata.
func (iso *ISO9660) GetUUID() string {
	if len(iso.pvd) < 829 {
		return ""
	}

	uuid := strings.TrimSpace(string(iso.pvd[813:829]))
	if len(uuid) < 4 {
		return uuid
	}

	// Format as XXXX-XX-XX-XX-XX-XX-XX
	result := uuid[:4]
	for i := 4; i < len(uuid); i += 2 {
		end := i + 2
		if end > len(uuid) {
			end = len(uuid)
		}
		result += "-" + uuid[i:end]
	}
	return result
}

// IterFiles returns a list of files in the filesystem.
// If onlyRootDir is true, only files in the root directory are returned.
func (iso *ISO9660) IterFiles(onlyRootDir bool) ([]FileInfo, error) {
	var files []FileInfo

	for idx, entry := range iso.pathTable {
		// Build full directory path
		dirPath := entry.name
		tmpIdx := entry.parentIdx
		for tmpIdx >= 0 && tmpIdx < len(iso.pathTable) {
			dirPath = iso.pathTable[tmpIdx].name + dirPath
			tmpIdx = iso.pathTable[tmpIdx].parentIdx
		}

		// Read directory entries
		offset := iso.blockOffset + int64(entry.lba)*int64(iso.blockSize)

		for {
			// Read record length
			lenBuf := make([]byte, 1)
			if _, err := iso.file.ReadAt(lenBuf, offset); err != nil {
				break
			}
			recLen := int(lenBuf[0])
			if recLen == 0 {
				break
			}

			// Read record
			recBuf := make([]byte, recLen-1)
			if _, err := iso.file.ReadAt(recBuf, offset+1); err != nil {
				break
			}

			// Check flags (offset 24 in record, which is 25 from start)
			flags := recBuf[24]

			// Skip directories (bit 1 set)
			if (flags & 0x02) == 0 {
				// File entry
				fileLBA := binary.LittleEndian.Uint32(recBuf[1:5])
				fileSize := binary.LittleEndian.Uint32(recBuf[9:13])
				fileNameLen := int(recBuf[31])

				if fileNameLen > 0 && 32+fileNameLen <= len(recBuf) {
					fileName := string(recBuf[32 : 32+fileNameLen])
					filePath := dirPath + fileName

					// Only include if in root dir (when requested)
					if !onlyRootDir || strings.Count(filePath, "/") == 1 {
						files = append(files, FileInfo{
							Path: filePath,
							LBA:  fileLBA,
							Size: fileSize,
						})
					}
				}
			}

			offset += int64(recLen)
		}

		// If only root dir and this is not root, skip other directories
		if onlyRootDir && idx > 0 {
			break
		}
	}

	return files, nil
}

// ReadFile reads the contents of a file by its FileInfo.
func (iso *ISO9660) ReadFile(info FileInfo) ([]byte, error) {
	offset := iso.blockOffset + int64(info.LBA)*int64(iso.blockSize)
	data := make([]byte, info.Size)
	if _, err := iso.file.ReadAt(data, offset); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file %s: %w", info.Path, err)
	}
	return data, nil
}

// ReadFileByPath reads a file by its path.
func (iso *ISO9660) ReadFileByPath(path string) ([]byte, error) {
	files, err := iso.IterFiles(false)
	if err != nil {
		return nil, err
	}

	// Normalize path
	path = strings.ToUpper(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	for _, f := range files {
		// ISO9660 filenames often have version suffix (;1)
		fpath := strings.ToUpper(f.Path)
		fpath = strings.Split(fpath, ";")[0]

		if fpath == path || f.Path == path {
			return iso.ReadFile(f)
		}
	}

	return nil, ErrFileNotFound
}

// FileExists checks if a file exists at the given path.
func (iso *ISO9660) FileExists(path string) bool {
	_, err := iso.ReadFileByPath(path)
	return err == nil
}

// BlockSize returns the block size of the disc image.
func (iso *ISO9660) BlockSize() int {
	return iso.blockSize
}

// Size returns the total size of the disc image.
func (iso *ISO9660) Size() int64 {
	return iso.size
}
