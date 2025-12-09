package identifier

import (
	"fmt"
	"io"

	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// SNES header offsets (relative to header start)
const (
	snesLoROMHeaderStart = 0x7FC0
	snesHiROMHeaderStart = 0xFFC0
	snesHeaderSize       = 32

	snesInternalNameOffset       = 0x00
	snesInternalNameSize         = 21
	snesMapModeOffset            = 0x15 // 21
	snesROMTypeOffset            = 0x16 // 22
	snesDeveloperIDOffset        = 0x1A // 26
	snesROMVersionOffset         = 0x1B // 27
	snesChecksumComplementOffset = 0x1C // 28
	snesChecksumOffset           = 0x1E // 30
)

// SNESIdentifier identifies Super Nintendo games.
type SNESIdentifier struct{}

// NewSNESIdentifier creates a new SNES identifier.
func NewSNESIdentifier() *SNESIdentifier {
	return &SNESIdentifier{}
}

// Console returns the console type.
func (s *SNESIdentifier) Console() Console {
	return ConsoleSNES
}

// Identify extracts SNES game information from the given reader.
func (s *SNESIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	// Read entire ROM for analysis
	data := make([]byte, size)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read SNES ROM: %w", err)
	}

	// Check for and strip 512-byte SMC header
	if len(data)%1024 == 512 {
		data = data[512:]
	}

	// Find valid header by checking checksum complement
	var headerStart int
	var checksum uint16
	var foundHeader bool

	for _, start := range []int{snesLoROMHeaderStart, snesHiROMHeaderStart} {
		if start+snesHeaderSize > len(data) {
			continue
		}

		// Read checksum and complement
		cs := uint16(data[start+snesChecksumOffset+1])<<8 | uint16(data[start+snesChecksumOffset])
		csc := uint16(data[start+snesChecksumComplementOffset+1])<<8 | uint16(data[start+snesChecksumComplementOffset])

		// Valid header if checksum + complement = 0xFFFF
		if cs+csc == 0xFFFF {
			headerStart = start
			checksum = cs
			foundHeader = true
			break
		}
	}

	if !foundHeader {
		return nil, ErrInvalidFormat{Console: ConsoleSNES, Reason: "no valid header found"}
	}

	header := data[headerStart:]

	// Extract internal name (21 bytes)
	internalName := header[snesInternalNameOffset : snesInternalNameOffset+snesInternalNameSize]
	internalNameHex := "0x"
	for _, b := range internalName {
		internalNameHex += fmt.Sprintf("%02x", b)
	}

	// Extract other fields
	mapMode := header[snesMapModeOffset]
	romType := header[snesROMTypeOffset]
	developerID := header[snesDeveloperIDOffset]
	romVersion := header[snesROMVersionOffset]

	// Determine FastROM/SlowROM
	fastSlowROM := "SlowROM"
	if (mapMode & 0x10) != 0 {
		fastSlowROM = "FastROM"
	}

	// Determine ROM type (LoROM/HiROM/ExLoROM/ExHiROM)
	romTypeStr := "LoROM"
	if (mapMode & 0x01) != 0 {
		romTypeStr = "HiROM"
	}
	if (mapMode & 0x04) != 0 {
		romTypeStr = "Ex" + romTypeStr
	}

	// Determine hardware
	var hardware string
	switch {
	case romType == 0:
		hardware = "ROM"
	case romType == 1:
		hardware = "ROM + RAM"
	case romType == 2:
		hardware = "ROM + RAM + Battery"
	case romType >= 3 && romType <= 6:
		hardware = []string{
			"ROM + Coprocessor",
			"ROM + Coprocessor + RAM",
			"ROM + Coprocessor + RAM + Battery",
			"ROM + Coprocessor + Battery",
		}[romType-3]
	}

	// Determine coprocessor if present
	if romType >= 3 {
		coprocessor := ""
		chipByte := (mapMode & 0xF0) >> 4
		switch chipByte {
		case 0:
			coprocessor = "DSP"
		case 1:
			coprocessor = "Super FX"
		case 2:
			coprocessor = "OBC1"
		case 3:
			coprocessor = "SA-1"
		case 4:
			coprocessor = "S-DD1"
		case 5:
			coprocessor = "S-RTC"
		case 0xE:
			coprocessor = "Super Game Boy / Satellaview"
		case 0xF:
			if headerStart > 0 {
				prevByte := data[headerStart-1]
				switch prevByte & 0x0F {
				case 0:
					coprocessor = "SPC7110"
				case 1:
					coprocessor = "ST010 / ST011"
				case 2:
					coprocessor = "ST018"
				case 3:
					coprocessor = "CX4"
				}
			}
		}
		if hardware != "" && coprocessor != "" {
			hardware = hardware[:len(hardware)-1] + " (" + coprocessor + ")"
		}
	}

	// Convert internal name to printable string for title fallback
	internalNameStr := binary.ExtractPrintable(internalName)

	result := NewResult(ConsoleSNES)
	result.InternalTitle = internalNameStr
	result.SetMetadata("internal_title", internalNameHex)
	result.SetMetadata("fast_slow_rom", fastSlowROM)
	result.SetMetadata("rom_type", romTypeStr)
	result.SetMetadata("developer_ID", fmt.Sprintf("0x%02x", developerID))
	result.SetMetadata("rom_version", fmt.Sprintf("%d", romVersion))
	result.SetMetadata("checksum", fmt.Sprintf("0x%04x", checksum))

	if hardware != "" {
		result.SetMetadata("hardware", hardware)
	}

	// Database lookup uses (developer_ID, internal_name_hex, rom_version, checksum) as key
	if db != nil {
		type snesKey struct {
			developerID  int
			internalName string
			romVersion   int
			checksum     int
		}
		key := snesKey{
			developerID:  int(developerID),
			internalName: internalNameHex,
			romVersion:   int(romVersion),
			checksum:     int(checksum),
		}
		if entry, found := db.Lookup(ConsoleSNES, key); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use internal name
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateSNES checks if the given data looks like a valid SNES ROM.
func ValidateSNES(data []byte) bool {
	// Strip SMC header if present
	if len(data)%1024 == 512 && len(data) > 512 {
		data = data[512:]
	}

	for _, start := range []int{snesLoROMHeaderStart, snesHiROMHeaderStart} {
		if start+snesHeaderSize > len(data) {
			continue
		}

		cs := uint16(data[start+snesChecksumOffset+1])<<8 | uint16(data[start+snesChecksumOffset])
		csc := uint16(data[start+snesChecksumComplementOffset+1])<<8 | uint16(data[start+snesChecksumComplementOffset])

		if cs+csc == 0xFFFF {
			return true
		}
	}

	return false
}
