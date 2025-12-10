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

package chd

import (
	"testing"
)

func TestOpenSegaCDCHD(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	t.Logf("Header version: %d", chdFile.Header().Version)
	t.Logf("Logical bytes: %d", chdFile.Header().LogicalBytes)
	t.Logf("Hunk bytes: %d", chdFile.Header().HunkBytes)
	t.Logf("Unit bytes: %d", chdFile.Header().UnitBytes)
	t.Logf("Map offset: %d", chdFile.Header().MapOffset)
	t.Logf("Compressors: %v", chdFile.Header().Compressors)

	// Try reading some data
	reader := chdFile.RawSectorReader()
	buf := make([]byte, 256)
	bytesRead, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}
	t.Logf("Read %d bytes", bytesRead)
	t.Logf("First 32 bytes: %x", buf[:32])
}

func TestHunkMapDebug(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	// Print first 10 hunk map entries
	t.Logf("Number of hunks: %d", chdFile.hunkMap.NumHunks())
	for idx := uint32(0); idx < 10 && idx < chdFile.hunkMap.NumHunks(); idx++ {
		entry := chdFile.hunkMap.entries[idx]
		t.Logf("Hunk %d: CompType=%d, CompLength=%d, Offset=%d",
			idx, entry.CompType, entry.CompLength, entry.Offset)
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen,nestif,revive,govet // Debug test with extensive diagnostic output
func TestNeoGeoCDCHD(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/NeoGeoCD/240pTestSuite.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	header := chdFile.Header()
	t.Logf("Version: %d", header.Version)
	t.Logf("HunkBytes: %d", header.HunkBytes)
	t.Logf("UnitBytes: %d", header.UnitBytes)
	t.Logf("Compressors: %x", header.Compressors)

	frames := int(header.HunkBytes) / int(header.UnitBytes)
	t.Logf("Frames per hunk: %d", frames)
	t.Logf("Sector bytes per hunk: %d", frames*2352)

	// Print track information
	t.Log("\nTrack information:")
	t.Logf("MetaOffset: %d", header.MetaOffset)
	tracks := chdFile.Tracks()
	t.Logf("Number of tracks: %d", len(tracks))
	for i, track := range tracks {
		t.Logf("Track %d: Type=%s, Frames=%d, StartFrame=%d, Pregap=%d, IsData=%v",
			i+1, track.Type, track.Frames, track.StartFrame, track.Pregap, track.IsDataTrack())
	}

	// Debug: Try parsing metadata directly
	if header.MetaOffset > 0 {
		metaBuf := make([]byte, 100)
		metaBytes, metaErr := chdFile.file.ReadAt(metaBuf, int64(header.MetaOffset)) //nolint:gosec // Test only
		if metaErr != nil {
			t.Logf("Read metadata raw failed: %v", metaErr)
		} else {
			t.Logf("Raw metadata (%d bytes): %x", metaBytes, metaBuf[:metaBytes])
			t.Logf("Tag: %s", string(metaBuf[0:4]))
			t.Logf("Data: %s", string(metaBuf[16:80]))
		}

		// Debug: Try parseMetadata directly
		entries, parseErr := parseMetadata(chdFile.file, header.MetaOffset)
		if parseErr != nil {
			t.Logf("parseMetadata failed: %v", parseErr)
		} else {
			t.Logf("Parsed %d metadata entries", len(entries))
			for i, entry := range entries {
				t.Logf("Entry %d: Tag=%x, Flags=%d, DataLen=%d, Next=%d",
					i, entry.Tag, entry.Flags, len(entry.Data), entry.Next)
				t.Logf("  Data: %s", string(entry.Data))
			}

			// Try parseTracks
			parsedTracks, err := parseTracks(entries)
			if err != nil {
				t.Logf("parseTracks failed: %v", err)
			} else {
				t.Logf("Parsed %d tracks", len(parsedTracks))
				for i, track := range parsedTracks {
					t.Logf("  Track %d: Type=%s Frames=%d IsData=%v",
						i+1, track.Type, track.Frames, track.IsDataTrack())
				}
			}
		}
	}

	// Test firstDataTrackSector
	t.Log("\nData track sector offset:", chdFile.firstDataTrackSector())
	t.Log("Data track size:", chdFile.DataTrackSize())

	// Print first 20 hunk map entries to see the pattern
	t.Log("\nHunk map entries:")
	t.Logf("Number of hunks: %d", chdFile.hunkMap.NumHunks())
	for idx := uint32(0); idx < 20 && idx < chdFile.hunkMap.NumHunks(); idx++ {
		entry := chdFile.hunkMap.entries[idx]
		codecName := "?"
		if int(entry.CompType) < len(header.Compressors) {
			tag := header.Compressors[entry.CompType]
			codecName = string([]byte{byte(tag >> 24), byte(tag >> 16), byte(tag >> 8), byte(tag)})
		}
		t.Logf("Hunk %d: CompType=%d (%s), CompLength=%d, Offset=%d",
			idx, entry.CompType, codecName, entry.CompLength, entry.Offset)
	}

	// Try to read data from a hunk that uses LZMA (comptype 0)
	// Skip the FLAC hunks and read from hunk 2 which is LZMA
	t.Log("\nTrying to read hunk 2 (LZMA)...")
	hunkData, err := chdFile.hunkMap.ReadHunk(2)
	if err != nil {
		t.Logf("Read hunk 2 failed: %v", err)
	} else {
		t.Logf("Hunk 2 data length: %d", len(hunkData))
		if len(hunkData) > 32 {
			t.Logf("First 32 bytes: %x", hunkData[:32])
		}
	}

	// Test reading sector 0 (where PVD actually starts for this disc)
	t.Log("\nTrying to read sector 0 using DataTrackSectorReader...")
	reader := chdFile.DataTrackSectorReader()
	sector0Data := make([]byte, 2048)
	readBytes, err := reader.ReadAt(sector0Data, 0) // Sector 0
	if err != nil {
		t.Logf("Read sector 0 failed: %v", err)
	} else {
		t.Logf("Read %d bytes", readBytes)
		t.Logf("First 32 bytes: %x", sector0Data[:32])
		t.Logf("String view: %s", string(sector0Data[1:6]))
	}

	// Test reading sector 16 using DataTrackSectorReader (where PVD should be)
	t.Log("\nTrying to read sector 16 (PVD) using DataTrackSectorReader...")
	pvdData := make([]byte, 2048)
	readBytes, err = reader.ReadAt(pvdData, 16*2048) // Sector 16
	if err != nil {
		t.Logf("Read PVD sector failed: %v", err)
	} else {
		t.Logf("Read %d bytes", readBytes)
		t.Logf("First 32 bytes: %x", pvdData[:32])
		t.Logf("String view: %s", string(pvdData[1:6]))
	}

	// Check hunk 0 data to understand the disc layout
	t.Log("\nReading hunk 0 (audio track) to see what's there...")
	hunk0Data, err := chdFile.hunkMap.ReadHunk(0)
	if err != nil {
		t.Logf("Read hunk 0 failed: %v", err)
	} else {
		t.Logf("Hunk 0 length: %d", len(hunk0Data))
		t.Logf("Hunk 0 first 32 bytes: %x", hunk0Data[:32])
	}

	// Simulate what iso9660.OpenCHD does - read first ~50KB via DataTrackSectorReader
	t.Log("\nSimulating ISO9660 init read (first 50KB)...")
	isoReader := chdFile.DataTrackSectorReader()
	isoData := make([]byte, 50000)
	readBytes, err = isoReader.ReadAt(isoData, 0)
	if err != nil && err.Error() != "EOF" {
		t.Logf("ISO reader failed: %v", err)
	}
	t.Logf("Read %d bytes", readBytes)
	// Search for CD001 in the data
	for i := range len(isoData) - 6 {
		if isoData[i] == 0x01 && isoData[i+1] == 'C' && isoData[i+2] == 'D' &&
			isoData[i+3] == '0' && isoData[i+4] == '0' && isoData[i+5] == '1' {
			t.Logf("Found PVD at offset %d (sector %d)", i, i/2048)
			t.Logf("Data at PVD: %x", isoData[i:i+32])
			break
		}
	}

	// The PVD \x01CD001 was seen at the start of hunk 2
	// Hunk 2 contains frames 16-23
	// Sector 0 of the data should contain the system area (16 reserved sectors)
	// Then PVD at sector 16
	// So if hunk 2 starts the data track, the PVD should be at the start
	// of the sector data in hunk 2 (after extracting from raw sector)
	t.Log("\nChecking raw hunk 2 data structure...")
	hunk2, err := chdFile.hunkMap.ReadHunk(2)
	if err != nil {
		t.Logf("Read hunk 2 failed: %v", err)
	} else {
		// Check the first sector in hunk 2
		// Raw sector = 2352 bytes, subchannel = 96 bytes, unit = 2448 bytes
		unitBytes := int(header.UnitBytes)
		t.Logf("Unit bytes: %d", unitBytes)
		for sectorIdx := range 3 {
			sectorStart := sectorIdx * unitBytes
			if sectorStart+32 > len(hunk2) {
				break
			}
			// Check sync header (first 12 bytes of raw sector)
			t.Logf("Sector %d raw start: %x", sectorIdx, hunk2[sectorStart:sectorStart+32])
			// User data starts at offset 16 (Mode1)
			dataStart := sectorStart + 16
			if dataStart+32 <= len(hunk2) {
				t.Logf("Sector %d user data (+16): %x", sectorIdx, hunk2[dataStart:dataStart+32])
			}
		}
	}
}
