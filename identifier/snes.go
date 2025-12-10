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
func (*SNESIdentifier) Console() Console {
	return ConsoleSNES
}

// snesHeaderInfo contains parsed SNES header information.
type snesHeaderInfo struct {
	internalNameHex string
	internalName    []byte
	headerStart     int
	checksum        uint16
	mapMode         byte
	romType         byte
	developerID     byte
	romVersion      byte
}

// snesFindHeader locates and validates the SNES header in ROM data.
func snesFindHeader(data []byte) (snesHeaderInfo, error) {
	for _, start := range []int{snesLoROMHeaderStart, snesHiROMHeaderStart} {
		if start+snesHeaderSize > len(data) {
			continue
		}

		// Read checksum and complement
		cs := uint16(data[start+snesChecksumOffset+1])<<8 | uint16(data[start+snesChecksumOffset])
		csc := uint16(data[start+snesChecksumComplementOffset+1])<<8 | uint16(data[start+snesChecksumComplementOffset])

		// Valid header if checksum + complement = 0xFFFF
		if cs+csc == 0xFFFF {
			header := data[start:]
			internalName := header[snesInternalNameOffset : snesInternalNameOffset+snesInternalNameSize]
			return snesHeaderInfo{
				headerStart:     start,
				checksum:        cs,
				internalName:    internalName,
				internalNameHex: snesFormatInternalNameHex(internalName),
				mapMode:         header[snesMapModeOffset],
				romType:         header[snesROMTypeOffset],
				developerID:     header[snesDeveloperIDOffset],
				romVersion:      header[snesROMVersionOffset],
			}, nil
		}
	}
	return snesHeaderInfo{}, ErrInvalidFormat{Console: ConsoleSNES, Reason: "no valid header found"}
}

// snesFormatInternalNameHex converts internal name bytes to hex string.
func snesFormatInternalNameHex(internalName []byte) string {
	result := "0x"
	for _, b := range internalName {
		result += fmt.Sprintf("%02x", b)
	}
	return result
}

// snesGetFastSlowROM determines if ROM is FastROM or SlowROM.
func snesGetFastSlowROM(mapMode byte) string {
	if (mapMode & 0x10) != 0 {
		return "FastROM"
	}
	return "SlowROM"
}

// snesGetROMTypeStr determines ROM mapping type string.
func snesGetROMTypeStr(mapMode byte) string {
	romTypeStr := "LoROM"
	if (mapMode & 0x01) != 0 {
		romTypeStr = "HiROM"
	}
	if (mapMode & 0x04) != 0 {
		romTypeStr = "Ex" + romTypeStr
	}
	return romTypeStr
}

// snesGetHardware determines hardware configuration string.
func snesGetHardware(romType, mapMode byte, data []byte, headerStart int) string {
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
	if romType >= 3 && hardware != "" {
		coprocessor := snesGetCoprocessor(mapMode, data, headerStart)
		if coprocessor != "" {
			hardware = hardware[:len(hardware)-1] + " (" + coprocessor + ")"
		}
	}
	return hardware
}

// Identify extracts SNES game information from the given reader.
func (*SNESIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	// Determine if file has SMC header (512 bytes) by checking file size
	hasSMCHeader := size%1024 == 512

	// Calculate read offset and size - we only need to read up to the HiROM header location
	// plus header size. SMC header is at offset 0 if present.
	readOffset := int64(0)
	if hasSMCHeader {
		readOffset = 512
	}

	// Only read what we need: up to HiROM header (0xFFC0) + header size (32 bytes)
	maxNeeded := int64(snesHiROMHeaderStart + snesHeaderSize)
	readSize := min(size-readOffset, maxNeeded)

	data := make([]byte, readSize)
	if _, err := reader.ReadAt(data, readOffset); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read SNES ROM: %w", err)
	}

	// Find and parse header
	info, err := snesFindHeader(data)
	if err != nil {
		return nil, err
	}

	// Convert internal name to printable string for title fallback
	internalNameStr := binary.ExtractPrintable(info.internalName)

	result := NewResult(ConsoleSNES)
	result.InternalTitle = internalNameStr
	result.SetMetadata("internal_title", info.internalNameHex)
	result.SetMetadata("fast_slow_rom", snesGetFastSlowROM(info.mapMode))
	result.SetMetadata("rom_type", snesGetROMTypeStr(info.mapMode))
	result.SetMetadata("developer_ID", fmt.Sprintf("0x%02x", info.developerID))
	result.SetMetadata("rom_version", fmt.Sprintf("%d", info.romVersion))
	result.SetMetadata("checksum", fmt.Sprintf("0x%04x", info.checksum))

	if hardware := snesGetHardware(info.romType, info.mapMode, data, info.headerStart); hardware != "" {
		result.SetMetadata("hardware", hardware)
	}

	// Database lookup
	snesLookupDatabase(result, db, info)

	// If no title from database, use internal name
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// snesLookupDatabase performs database lookup for SNES game.
func snesLookupDatabase(result *Result, db Database, info snesHeaderInfo) {
	if db == nil {
		return
	}
	type snesKey struct {
		internalName string
		developerID  int
		romVersion   int
		checksum     int
	}
	key := snesKey{
		developerID:  int(info.developerID),
		internalName: info.internalNameHex,
		romVersion:   int(info.romVersion),
		checksum:     int(info.checksum),
	}
	if entry, found := db.Lookup(ConsoleSNES, key); found {
		result.MergeMetadata(entry)
	}
}

// snesGetCoprocessor determines the coprocessor type from the map mode.
func snesGetCoprocessor(mapMode byte, data []byte, headerStart int) string {
	chipByte := (mapMode & 0xF0) >> 4
	switch chipByte {
	case 0:
		return "DSP"
	case 1:
		return "Super FX"
	case 2:
		return "OBC1"
	case 3:
		return "SA-1"
	case 4:
		return "S-DD1"
	case 5:
		return "S-RTC"
	case 0xE:
		return "Super Game Boy / Satellaview"
	case 0xF:
		return snesGetExtendedCoprocessor(data, headerStart)
	default:
		return ""
	}
}

// snesGetExtendedCoprocessor determines extended coprocessor type (0xF chip byte).
func snesGetExtendedCoprocessor(data []byte, headerStart int) string {
	if headerStart <= 0 {
		return ""
	}
	prevByte := data[headerStart-1]
	switch prevByte & 0x0F {
	case 0:
		return "SPC7110"
	case 1:
		return "ST010 / ST011"
	case 2:
		return "ST018"
	case 3:
		return "CX4"
	default:
		return ""
	}
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
