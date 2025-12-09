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
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	bin "github.com/ZaparooProject/go-gameid/internal/binary"
)

// Genesis magic words to search for in ROM
var genesisMagicWords = [][]byte{
	[]byte("SEGA GENESIS"),
	[]byte("SEGA MEGA DRIVE"),
	[]byte("SEGA 32X"),
	[]byte("SEGA EVERDRIVE"),
	[]byte("SEGA SSF"),
	[]byte("SEGA MEGAWIFI"),
	[]byte("SEGA PICO"),
	[]byte("SEGA TERA68K"),
	[]byte("SEGA TERA286"),
}

// Genesis device support codes
var genesisDeviceSupport = map[byte]string{
	'J': "3-button Controller",
	'6': "6-button Controller",
	'0': "Master System Controller",
	'A': "Analog Joystick",
	'4': "Multitap",
	'G': "Lightgun",
	'L': "Activator",
	'M': "Mouse",
	'B': "Trackball",
	'T': "Tablet",
	'V': "Paddle",
	'K': "Keyboard or Keypad",
	'R': "RS-232",
	'P': "Printer",
	'C': "CD-ROM (Sega CD)",
	'F': "Floppy Drive",
	'D': "Download",
}

// Genesis region support codes
var genesisRegionSupport = map[byte]string{
	'J': "Japan",
	'U': "Americas",
	'E': "Europe",
}

// Genesis software types
var genesisSoftwareTypes = map[string]string{
	"GM": "Game",
	"AI": "Aid",
	"OS": "Boot ROM (TMSS)",
	"BR": "Boot ROM (Sega CD)",
}

// GenesisIdentifier identifies Sega Genesis / Mega Drive games.
type GenesisIdentifier struct{}

// NewGenesisIdentifier creates a new Genesis identifier.
func NewGenesisIdentifier() *GenesisIdentifier {
	return &GenesisIdentifier{}
}

// Console returns the console type.
func (*GenesisIdentifier) Console() Console {
	return ConsoleGenesis
}

// Identify extracts Genesis game information from the given reader.
func (*GenesisIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	data, magicWordInd, err := genesisReadHeader(reader, size)
	if err != nil {
		return nil, err
	}

	return genesisParseHeader(data, magicWordInd, db)
}

// genesisReadHeader reads the Genesis ROM header and finds the magic word.
func genesisReadHeader(reader io.ReaderAt, size int64) (data []byte, magicWordIndex int, err error) {
	searchSize := int64(0x200)
	if size < searchSize {
		searchSize = size
	}

	data = make([]byte, searchSize)
	if _, readErr := reader.ReadAt(data, 0); readErr != nil && readErr != io.EOF {
		return nil, -1, fmt.Errorf("failed to read Genesis ROM: %w", readErr)
	}

	// Search for magic word in range 0x100-0x200
	magicWordInd := findGenesisMagicWord(data)
	if magicWordInd == -1 {
		return nil, -1, ErrInvalidFormat{Console: ConsoleGenesis, Reason: "magic word not found"}
	}

	// Need to read more data for full header
	headerEnd := magicWordInd + 0x100
	if int64(headerEnd) > size {
		headerEnd = int(size)
	}

	if headerEnd > len(data) {
		fullData := make([]byte, headerEnd)
		if _, readErr := reader.ReadAt(fullData, 0); readErr != nil && readErr != io.EOF {
			return nil, -1, fmt.Errorf("failed to read Genesis header: %w", readErr)
		}
		data = fullData
	}

	return data, magicWordInd, nil
}

// findGenesisMagicWord searches for a Genesis magic word in the data.
func findGenesisMagicWord(data []byte) int {
	for _, magicWord := range genesisMagicWords {
		for i := 0x100; i <= 0x200-len(magicWord); i++ {
			if bin.BytesEqual(data[i:i+len(magicWord)], magicWord) {
				return i
			}
		}
	}
	return -1
}

// genesisParseHeader parses the Genesis header and returns the result.
//
//nolint:funlen,revive // Header parsing requires many field extractions
func genesisParseHeader(data []byte, magicWordInd int, db Database) (*Result, error) {
	extractString := func(offset, length int) string {
		start := magicWordInd + offset
		end := start + length
		if end > len(data) {
			return ""
		}
		return bin.CleanString(data[start:end])
	}

	extractBytes := func(offset, length int) []byte {
		start := magicWordInd + offset
		end := start + length
		if end > len(data) {
			return nil
		}
		return data[start:end]
	}

	systemType := extractString(0x000, 0x010)
	publisher := extractString(0x013, 0x004)
	releaseYear := extractString(0x018, 0x004)
	releaseMonth := extractString(0x01D, 0x003)
	titleDomestic := extractString(0x020, 0x030)
	titleOverseas := extractString(0x050, 0x030)
	softwareType := extractString(0x080, 0x002)
	gameID := extractString(0x082, 0x009)
	revision := extractString(0x08C, 0x002)

	// Checksum is big-endian uint16
	checksumBytes := extractBytes(0x08E, 2)
	var checksum uint16
	if len(checksumBytes) == 2 {
		checksum = binary.BigEndian.Uint16(checksumBytes)
	}

	deviceSupport := parseGenesisDeviceSupport(extractBytes(0x090, 0x010))
	addrs := parseGenesisAddresses(extractBytes(0x0A0, 4),
		extractBytes(0x0A4, 4), extractBytes(0x0A8, 4), extractBytes(0x0AC, 4))
	regionSupport := parseGenesisRegionSupport(extractBytes(0x0F0, 0x003))

	// Normalize serial for database lookup (remove dashes and spaces)
	serial := strings.ReplaceAll(gameID, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleGenesis)
	result.ID = gameID
	result.InternalTitle = titleDomestic
	result.SetMetadata("system_type", systemType)
	result.SetMetadata("publisher", publisher)
	result.SetMetadata("release_year", releaseYear)
	result.SetMetadata("release_month", releaseMonth)
	result.SetMetadata("title_domestic", titleDomestic)
	result.SetMetadata("title_overseas", titleOverseas)
	result.SetMetadata("ID", gameID)
	result.SetMetadata("revision", revision)
	result.SetMetadata("checksum", fmt.Sprintf("0x%04x", checksum))
	result.SetMetadata("rom_start", fmt.Sprintf("0x%08x", addrs.romStart))
	result.SetMetadata("rom_end", fmt.Sprintf("0x%08x", addrs.romEnd))
	result.SetMetadata("ram_start", fmt.Sprintf("0x%08x", addrs.ramStart))
	result.SetMetadata("ram_end", fmt.Sprintf("0x%08x", addrs.ramEnd))

	setGenesisSoftwareType(result, softwareType)
	setGenesisDeviceSupport(result, deviceSupport)
	setGenesisRegionSupport(result, regionSupport)

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleGenesis, serial); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use domestic title
	setGenesisFallbackTitle(result, titleOverseas, titleDomestic)

	return result, nil
}

// setGenesisSoftwareType sets the software type metadata from the raw code.
func setGenesisSoftwareType(result *Result, softwareType string) {
	if softwareType == "" {
		return
	}
	if st, ok := genesisSoftwareTypes[softwareType]; ok {
		result.SetMetadata("software_type", st)
	} else {
		result.SetMetadata("software_type", softwareType)
	}
}

// setGenesisDeviceSupport sets the device support metadata.
func setGenesisDeviceSupport(result *Result, deviceSupport []string) {
	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}
}

// setGenesisRegionSupport sets the region support metadata.
func setGenesisRegionSupport(result *Result, regionSupport []string) {
	if len(regionSupport) > 0 {
		result.SetMetadata("region_support", strings.Join(regionSupport, " / "))
	}
}

// setGenesisFallbackTitle sets the title from internal names if not set from database.
func setGenesisFallbackTitle(result *Result, titleOverseas, titleDomestic string) {
	if result.Title != "" {
		return
	}
	if titleOverseas != "" {
		result.Title = titleOverseas
	} else {
		result.Title = titleDomestic
	}
}

// parseGenesisDeviceSupport parses device support bytes.
func parseGenesisDeviceSupport(deviceSupportBytes []byte) []string {
	var deviceSupport []string
	for _, devByte := range deviceSupportBytes {
		if devByte == 0 || devByte == ' ' {
			continue
		}
		if dev, ok := genesisDeviceSupport[devByte]; ok {
			deviceSupport = append(deviceSupport, dev)
		} else if devByte >= 0x20 && devByte <= 0x7E {
			deviceSupport = append(deviceSupport, string(devByte))
		}
	}
	return deviceSupport
}

// genesisAddresses holds ROM and RAM address information.
type genesisAddresses struct {
	romStart uint32
	romEnd   uint32
	ramStart uint32
	ramEnd   uint32
}

// parseGenesisAddresses parses ROM/RAM address bytes.
func parseGenesisAddresses(romStartBytes, romEndBytes, ramStartBytes, ramEndBytes []byte) genesisAddresses {
	var addrs genesisAddresses
	if len(romStartBytes) == 4 {
		addrs.romStart = binary.BigEndian.Uint32(romStartBytes)
	}
	if len(romEndBytes) == 4 {
		addrs.romEnd = binary.BigEndian.Uint32(romEndBytes)
	}
	if len(ramStartBytes) == 4 {
		addrs.ramStart = binary.BigEndian.Uint32(ramStartBytes)
	}
	if len(ramEndBytes) == 4 {
		addrs.ramEnd = binary.BigEndian.Uint32(ramEndBytes)
	}
	return addrs
}

// parseGenesisRegionSupport parses region support bytes.
func parseGenesisRegionSupport(regionSupportBytes []byte) []string {
	var regionSupport []string
	for _, regByte := range regionSupportBytes {
		if regByte == 0 || regByte == ' ' {
			continue
		}
		if reg, ok := genesisRegionSupport[regByte]; ok {
			regionSupport = append(regionSupport, reg)
		} else if regByte >= 0x20 && regByte <= 0x7E {
			regionSupport = append(regionSupport, string(regByte))
		}
	}
	return regionSupport
}

// ValidateGenesis checks if the given data looks like a valid Genesis ROM.
func ValidateGenesis(data []byte) bool {
	if len(data) < 0x200 {
		return false
	}

	for _, magicWord := range genesisMagicWords {
		for i := 0x100; i <= 0x200-len(magicWord); i++ {
			if bin.BytesEqual(data[i:i+len(magicWord)], magicWord) {
				return true
			}
		}
	}

	return false
}
