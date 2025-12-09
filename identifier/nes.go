package identifier

import (
	"fmt"
	"hash/crc32"
	"io"
)

// NESIdentifier identifies Nintendo Entertainment System games.
// NES identification relies on CRC32 checksum of the entire file.
type NESIdentifier struct{}

// NewNESIdentifier creates a new NES identifier.
func NewNESIdentifier() *NESIdentifier {
	return &NESIdentifier{}
}

// Console returns the console type.
func (n *NESIdentifier) Console() Console {
	return ConsoleNES
}

// Identify extracts NES game information from the given reader.
func (n *NESIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	// Read entire file for CRC32 calculation
	data := make([]byte, size)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read NES ROM: %w", err)
	}

	// Calculate CRC32 checksum
	checksum := crc32.ChecksumIEEE(data)

	result := NewResult(ConsoleNES)
	result.SetMetadata("crc32", fmt.Sprintf("%08x", checksum))

	// Database lookup uses CRC32 as integer key
	if db != nil {
		if entry, found := db.Lookup(ConsoleNES, int(checksum)); found {
			result.MergeMetadata(entry, false)
		}
	}

	// NES ROMs don't have internal title, so ID and title come from database
	if result.ID == "" {
		result.ID = fmt.Sprintf("%08x", checksum)
	}

	return result, nil
}
