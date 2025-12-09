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
	"errors"
	"testing"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// mockPlayStationISO implements playstationISO for testing.
type mockPlayStationISO struct {
	filesErr error
	uuid     string
	volumeID string
	files    []iso9660.FileInfo
}

func (m *mockPlayStationISO) GetUUID() string     { return m.uuid }
func (m *mockPlayStationISO) GetVolumeID() string { return m.volumeID }
func (m *mockPlayStationISO) IterFiles(_ bool) ([]iso9660.FileInfo, error) {
	return m.files, m.filesErr
}
func (*mockPlayStationISO) Close() error { return nil }

// mockDatabase implements Database for testing.
type mockDatabase struct {
	stringEntries map[Console]map[string]map[string]string
	idPrefixes    map[Console][]string
}

func newMockDatabase() *mockDatabase {
	return &mockDatabase{
		stringEntries: make(map[Console]map[string]map[string]string),
		idPrefixes:    make(map[Console][]string),
	}
}

func (*mockDatabase) Lookup(_ Console, _ any) (map[string]string, bool) {
	return nil, false
}

func (m *mockDatabase) LookupByString(console Console, key string) (map[string]string, bool) {
	if m.stringEntries == nil {
		return nil, false
	}
	consoleEntries, ok := m.stringEntries[console]
	if !ok {
		return nil, false
	}
	entry, found := consoleEntries[key]
	return entry, found
}

func (m *mockDatabase) GetIDPrefixes(console Console) []string {
	if m.idPrefixes == nil {
		return nil
	}
	return m.idPrefixes[console]
}

func (m *mockDatabase) addEntry(console Console, key string, entry map[string]string) {
	if m.stringEntries[console] == nil {
		m.stringEntries[console] = make(map[string]map[string]string)
	}
	m.stringEntries[console][key] = entry
}

func (m *mockDatabase) setPrefixes(console Console, prefixes []string) {
	m.idPrefixes[console] = prefixes
}

// Tests for serialFromVolumeID
func TestSerialFromVolumeID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		volumeID string
		want     string
	}{
		{"empty", "", ""},
		{"simple", "SLUS-00001", "SLUS_00001"},
		{"no dashes", "SLUS00001", "SLUS00001"},
		{"multiple dashes", "SLUS-000-01-02", "SLUS_000"},
		{"single part", "GAME", "GAME"},
		{"two parts", "SLUS_12345", "SLUS_12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := serialFromVolumeID(tt.volumeID)
			if got != tt.want {
				t.Errorf("serialFromVolumeID(%q) = %q, want %q", tt.volumeID, got, tt.want)
			}
		})
	}
}

// Tests for serialFromFilename
func TestSerialFromFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourcePath string
		want       string
	}{
		{"iso file", "/path/to/GAME.iso", "GAME"},
		{"cue file", "/path/to/GAME.cue", "GAME"},
		// Note: function trims extension first (.gz), then tries to trim ".gz" again (no effect)
		// This is existing behavior - GAME.iso.gz → GAME.iso → GAME.iso
		{"gz compressed", "/path/to/GAME.iso.gz", "GAME.iso"},
		{"bin file", "/path/to/SLUS_123.45.bin", "SLUS_123.45"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := serialFromFilename(tt.sourcePath)
			if got != tt.want {
				t.Errorf("serialFromFilename(%q) = %q, want %q", tt.sourcePath, got, tt.want)
			}
		})
	}
}

// Tests for findPlayStationSerial
func TestFindPlayStationSerial(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()
		got := findPlayStationSerial([]string{"SLUS_123.45"}, ConsolePSX, nil)
		if got != "" {
			t.Errorf("findPlayStationSerial with nil db = %q, want empty", got)
		}
	})

	t.Run("no prefixes", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		// No prefixes set
		got := findPlayStationSerial([]string{"SLUS_123.45"}, ConsolePSX, db)
		if got != "" {
			t.Errorf("findPlayStationSerial with no prefixes = %q, want empty", got)
		}
	})

	t.Run("matching prefix with database entry", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		db.setPrefixes(ConsolePSX, []string{"SLUS"})
		db.addEntry(ConsolePSX, "SLUS_12345", map[string]string{"title": "Test Game"})

		got := findPlayStationSerial([]string{"SLUS_123.45"}, ConsolePSX, db)
		if got != "SLUS_12345" {
			t.Errorf("findPlayStationSerial = %q, want %q", got, "SLUS_12345")
		}
	})

	t.Run("no matching files", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		db.setPrefixes(ConsolePSX, []string{"SLUS"})

		got := findPlayStationSerial([]string{"README.TXT", "ICON.ICO"}, ConsolePSX, db)
		if got != "" {
			t.Errorf("findPlayStationSerial with no matching files = %q, want empty", got)
		}
	})
}

// Tests for trySerialLookup
func TestTrySerialLookup(t *testing.T) {
	t.Parallel()

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		db.addEntry(ConsolePSX, "SLUS_12345", map[string]string{"title": "Test"})

		got := trySerialLookup("SLUS_12345", "SLUS", ConsolePSX, db)
		if got != "SLUS_12345" {
			t.Errorf("trySerialLookup = %q, want %q", got, "SLUS_12345")
		}
	})

	t.Run("normalized dots and dashes", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		db.addEntry(ConsolePSX, "SLUS_12345", map[string]string{"title": "Test"})

		// Input has dots and dashes that get normalized
		got := trySerialLookup("SLUS-123.45", "SLUS", ConsolePSX, db)
		if got != "SLUS_12345" {
			t.Errorf("trySerialLookup = %q, want %q", got, "SLUS_12345")
		}
	})

	t.Run("underscore variant", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		// After normalization: SLUS12345, altSerial = SLUS + _ + 2345 = SLUS_2345
		// (skips char at index len(prefix))
		db.addEntry(ConsolePSX, "SLUS_2345", map[string]string{"title": "Test"})

		// After normalization: SLUS12345, try with underscore after prefix
		got := trySerialLookup("SLUS12345", "SLUS", ConsolePSX, db)
		if got != "SLUS_2345" {
			t.Errorf("trySerialLookup = %q, want %q", got, "SLUS_2345")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()

		got := trySerialLookup("SLUS_99999", "SLUS", ConsolePSX, db)
		if got != "" {
			t.Errorf("trySerialLookup not found = %q, want empty", got)
		}
	})
}

// Tests for identifyPlayStation
//
//nolint:gocognit,revive,funlen // Test covers multiple scenarios in subtests
func TestIdentifyPlayStation(t *testing.T) {
	t.Parallel()

	t.Run("serial from root file", func(t *testing.T) {
		t.Parallel()
		db := newMockDatabase()
		db.setPrefixes(ConsolePSX, []string{"SLUS"})
		db.addEntry(ConsolePSX, "SLUS_12345", map[string]string{
			"title": "Test Game",
			"ID":    "SLUS-12345",
		})

		mockISO := &mockPlayStationISO{
			uuid:     "test-uuid",
			volumeID: "SLUS-12345",
			files: []iso9660.FileInfo{
				{Path: "/SLUS_123.45;1"},
			},
		}

		result, err := identifyPlayStation(mockISO, ConsolePSX, db, "/games/test.iso")
		if err != nil {
			t.Fatalf("identifyPlayStation() error = %v", err)
		}

		if result.ID != "SLUS-12345" {
			t.Errorf("result.ID = %q, want %q", result.ID, "SLUS-12345")
		}
		if result.Title != "Test Game" {
			t.Errorf("result.Title = %q, want %q", result.Title, "Test Game")
		}
	})

	t.Run("volume ID fallback", func(t *testing.T) {
		t.Parallel()
		mockISO := &mockPlayStationISO{
			uuid:     "test-uuid",
			volumeID: "SCUS-12345",
			files:    []iso9660.FileInfo{},
		}

		result, err := identifyPlayStation(mockISO, ConsolePSX, nil, "")
		if err != nil {
			t.Fatalf("identifyPlayStation() error = %v", err)
		}

		// Serial comes from volume ID, ID gets dashes replaced with underscores then back
		if result.ID != "SCUS-12345" {
			t.Errorf("result.ID = %q, want %q", result.ID, "SCUS-12345")
		}
	})

	t.Run("filename fallback", func(t *testing.T) {
		t.Parallel()
		mockISO := &mockPlayStationISO{
			uuid:     "",
			volumeID: "",
			files:    []iso9660.FileInfo{},
		}

		result, err := identifyPlayStation(mockISO, ConsolePSX, nil, "/games/SLPM-12345.iso")
		if err != nil {
			t.Fatalf("identifyPlayStation() error = %v", err)
		}

		if result.ID != "SLPM-12345" {
			t.Errorf("result.ID = %q, want %q", result.ID, "SLPM-12345")
		}
	})

	t.Run("IterFiles error", func(t *testing.T) {
		t.Parallel()
		mockISO := &mockPlayStationISO{
			filesErr: errors.New("read error"),
		}

		_, err := identifyPlayStation(mockISO, ConsolePSX, nil, "")
		if err == nil {
			t.Error("identifyPlayStation() should error on IterFiles error")
		}
	})

	t.Run("metadata populated", func(t *testing.T) {
		t.Parallel()
		mockISO := &mockPlayStationISO{
			uuid:     "2024-01-01",
			volumeID: "TESTVOL",
			files: []iso9660.FileInfo{
				{Path: "/README.TXT"},
				{Path: "/ICON.ICO"},
			},
		}

		result, err := identifyPlayStation(mockISO, ConsolePSX, nil, "")
		if err != nil {
			t.Fatalf("identifyPlayStation() error = %v", err)
		}

		if result.Metadata["uuid"] != "2024-01-01" {
			t.Errorf("uuid metadata = %q, want %q", result.Metadata["uuid"], "2024-01-01")
		}
		if result.Metadata["volume_ID"] != "TESTVOL" {
			t.Errorf("volume_ID metadata = %q, want %q", result.Metadata["volume_ID"], "TESTVOL")
		}
		if result.Metadata["root_files"] == "" {
			t.Error("root_files metadata should not be empty")
		}
	})
}

// Tests for PSXIdentifier
func TestPSXIdentifier_Console(t *testing.T) {
	t.Parallel()

	id := NewPSXIdentifier()
	if id.Console() != ConsolePSX {
		t.Errorf("Console() = %v, want %v", id.Console(), ConsolePSX)
	}
}

func TestPSXIdentifier_Identify_ReturnsNotSupported(t *testing.T) {
	t.Parallel()

	id := NewPSXIdentifier()
	_, err := id.Identify(nil, 0, nil)

	var notSupported ErrNotSupported
	if !errors.As(err, &notSupported) {
		t.Errorf("Identify() error = %v, want ErrNotSupported", err)
	}
}

// Tests for version suffix stripping
func TestIdentifyPlayStation_VersionSuffix(t *testing.T) {
	t.Parallel()

	db := newMockDatabase()
	db.setPrefixes(ConsolePSX, []string{"SLUS"})
	db.addEntry(ConsolePSX, "SLUS_12345", map[string]string{"title": "Test"})

	mockISO := &mockPlayStationISO{
		files: []iso9660.FileInfo{
			{Path: "/SLUS_123.45;1"}, // With version suffix
		},
	}

	result, err := identifyPlayStation(mockISO, ConsolePSX, db, "")
	if err != nil {
		t.Fatalf("identifyPlayStation() error = %v", err)
	}

	// Check that root_files has the version suffix stripped
	rootFiles := result.Metadata["root_files"]
	if rootFiles != "SLUS_123.45" {
		t.Errorf("root_files = %q, want %q", rootFiles, "SLUS_123.45")
	}
}
