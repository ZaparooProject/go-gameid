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

package iso9660

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzISO9660Open fuzzes ISO9660 opening and PVD parsing.
// This tests the critical binary parsing paths in init() and parsePathTable().
func FuzzISO9660Open(f *testing.F) {
	// Add corpus seeds
	f.Add(createMinimalISO("VOL", "SYS", "PUB"))
	f.Add(make([]byte, 2048*20)) // Valid size, no PVD
	f.Add(make([]byte, 2352*20)) // CD-ROM raw sector size
	f.Add(make([]byte, 1000))    // Invalid size
	f.Add([]byte{})              // Empty

	// Add seed with corrupted PVD
	corruptedPVD := createMinimalISO("VOL", "SYS", "PUB")
	if len(corruptedPVD) > 16*2048+200 {
		// Corrupt path table size to be very large
		corruptedPVD[16*2048+132] = 0xFF
		corruptedPVD[16*2048+133] = 0xFF
		corruptedPVD[16*2048+134] = 0xFF
		corruptedPVD[16*2048+135] = 0xFF
	}
	f.Add(corruptedPVD)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip extremely large inputs to prevent memory issues in fuzzing
		if len(data) > 10*1024*1024 { // 10MB limit
			return
		}

		// Create temp file
		tmpDir := t.TempDir()
		isoPath := filepath.Join(tmpDir, "fuzz.iso")
		if err := os.WriteFile(isoPath, data, 0o600); err != nil {
			return
		}

		// Try to open - should not panic regardless of input
		iso, err := Open(isoPath)
		if err != nil {
			// Expected for malformed data
			return
		}
		defer func() { _ = iso.Close() }()

		// If Open succeeded, these methods should not panic
		_ = iso.GetVolumeID()
		_ = iso.GetSystemID()
		_ = iso.GetPublisherID()
		_ = iso.GetDataPreparerID()
		_ = iso.GetUUID()
		_ = iso.BlockSize()
		_ = iso.Size()

		// IterFiles should terminate without panicking
		_, _ = iso.IterFiles(true)
		_, _ = iso.IterFiles(false)

		// FileExists should not panic
		_ = iso.FileExists("/NONEXISTENT.TXT")
	})
}

// FuzzParseCue fuzzes CUE sheet parsing.
func FuzzParseCue(f *testing.F) {
	// Add corpus seeds
	f.Add([]byte(`FILE "track.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00`))
	f.Add([]byte(`FILE "game.bin" BINARY
TRACK 01 MODE2/2352
  INDEX 01 00:00:00
FILE "audio.bin" BINARY
TRACK 02 AUDIO
  INDEX 01 00:00:00`))
	f.Add([]byte(`FILE "file with spaces.bin" BINARY
TRACK 01 MODE1/2048
  INDEX 01 00:00:00`))
	f.Add([]byte(``)) // Empty
	f.Add([]byte(`NOT A CUE FILE`))
	f.Add([]byte(`FILE BINARY`)) // Malformed

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create temp CUE file
		tmpDir := t.TempDir()
		cuePath := filepath.Join(tmpDir, "fuzz.cue")
		if err := os.WriteFile(cuePath, data, 0o600); err != nil {
			return
		}

		// ParseCue should not panic regardless of content
		cue, err := ParseCue(cuePath)
		if err != nil {
			return
		}

		// If parsing succeeded, verify fields are accessible
		_ = len(cue.BinFiles)
		for _, binFile := range cue.BinFiles {
			_ = binFile
		}
	})
}
