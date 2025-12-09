package identifier

import (
	"fmt"
	"io"

	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// N64 header offsets
const (
	n64HeaderSize         = 0x40
	n64FirstWordOffset    = 0x00
	n64InternalNameOffset = 0x20
	n64InternalNameSize   = 20 // 0x20-0x34
	n64CartridgeIDOffset  = 0x3C
	n64CartridgeIDSize    = 2
	n64CountryCodeOffset  = 0x3E
	n64VersionOffset      = 0x3F
)

// N64 first word magic - indicates big-endian format
var n64FirstWord = []byte{0x80, 0x37, 0x12, 0x40}

// N64Identifier identifies Nintendo 64 games.
type N64Identifier struct{}

// NewN64Identifier creates a new N64 identifier.
func NewN64Identifier() *N64Identifier {
	return &N64Identifier{}
}

// Console returns the console type.
func (n *N64Identifier) Console() Console {
	return ConsoleN64
}

// n64ConvertEndianness converts byte-swapped N64 ROM data to big-endian.
// This handles .v64 format ROMs which are byte-swapped.
func n64ConvertEndianness(data []byte) []byte {
	if len(data)%2 != 0 {
		return data
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += 2 {
		out[i] = data[i+1]
		out[i+1] = data[i]
	}
	return out
}

// Identify extracts N64 game information from the given reader.
func (n *N64Identifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < n64HeaderSize {
		return nil, ErrInvalidFormat{Console: ConsoleN64, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(r, 0, n64HeaderSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read N64 header: %w", err)
	}

	// Check first word to determine endianness
	firstWord := header[n64FirstWordOffset : n64FirstWordOffset+4]

	// Check if it's byte-swapped (.v64 format)
	if binary.BytesEqual(n64ConvertEndianness(firstWord), n64FirstWord) {
		header = n64ConvertEndianness(header)
	} else if !binary.BytesEqual(firstWord, n64FirstWord) {
		// Also check for word-swapped format (.n64)
		wordSwapped := []byte{header[3], header[2], header[1], header[0]}
		if binary.BytesEqual(wordSwapped, n64FirstWord) {
			// Word-swapped format - swap every 4 bytes
			for i := 0; i < len(header); i += 4 {
				header[i], header[i+1], header[i+2], header[i+3] =
					header[i+3], header[i+2], header[i+1], header[i]
			}
		} else {
			return nil, ErrInvalidFormat{Console: ConsoleN64, Reason: "invalid first word"}
		}
	}

	// Extract cartridge ID (2 bytes at 0x3C)
	cartridgeID := header[n64CartridgeIDOffset : n64CartridgeIDOffset+n64CartridgeIDSize]

	// Extract country code and version
	countryCode := header[n64CountryCodeOffset]
	version := header[n64VersionOffset]

	// Build serial: 2-char cartridge ID + country code character
	serial := fmt.Sprintf("%c%c%c", cartridgeID[0], cartridgeID[1], countryCode)

	// Extract internal name
	internalName := binary.CleanString(header[n64InternalNameOffset : n64InternalNameOffset+n64InternalNameSize])

	result := NewResult(ConsoleN64)
	result.ID = serial
	result.InternalTitle = internalName
	result.SetMetadata("ID", serial)
	result.SetMetadata("internal_name", internalName)
	result.SetMetadata("version", fmt.Sprintf("%d", version))
	result.SetMetadata("country_code", fmt.Sprintf("%c", countryCode))

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleN64, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use internal name
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateN64 checks if the given data looks like a valid N64 ROM.
func ValidateN64(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	firstWord := header[0:4]

	// Check big-endian format
	if binary.BytesEqual(firstWord, n64FirstWord) {
		return true
	}

	// Check byte-swapped format (.v64)
	if binary.BytesEqual(n64ConvertEndianness(firstWord), n64FirstWord) {
		return true
	}

	// Check word-swapped format (.n64)
	wordSwapped := []byte{firstWord[3], firstWord[2], firstWord[1], firstWord[0]}
	return binary.BytesEqual(wordSwapped, n64FirstWord)
}
