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
	"path/filepath"
	"reflect"
	"testing"
)

// mockNeoGeoCDISO implements the interface needed for NeoGeoCD identification.
type mockNeoGeoCDISO struct {
	uuid     string
	volumeID string
}

func (m *mockNeoGeoCDISO) GetUUID() string     { return m.uuid }
func (m *mockNeoGeoCDISO) GetVolumeID() string { return m.volumeID }

func TestNeoGeoCDIdentifier_Console(t *testing.T) {
	t.Parallel()

	id := NewNeoGeoCDIdentifier()
	if id.Console() != ConsoleNeoGeoCD {
		t.Errorf("Console() = %v, want %v", id.Console(), ConsoleNeoGeoCD)
	}
}

func TestNeoGeoCDIdentifier_Identify_ReturnsNotSupported(t *testing.T) {
	t.Parallel()

	id := NewNeoGeoCDIdentifier()
	_, err := id.Identify(nil, 0, nil)

	var notSupported ErrNotSupported
	if !errors.As(err, &notSupported) {
		t.Errorf("Identify() error = %v, want ErrNotSupported", err)
	}
}

func TestNeoGeoCDIdentifier_IdentifyFromPath_NonExistent(t *testing.T) {
	t.Parallel()

	id := NewNeoGeoCDIdentifier()
	_, err := id.IdentifyFromPath("/nonexistent/path/game.iso", nil)
	if err == nil {
		t.Error("IdentifyFromPath() should error for non-existent file")
	}
}

//nolint:gocognit,revive,funlen // Test covers multiple scenarios in subtests
func TestNeoGeoCDIdentifier_identifyFromISO(t *testing.T) {
	t.Parallel()

	t.Run("with uuid and volumeID", func(t *testing.T) {
		t.Parallel()
		id := NewNeoGeoCDIdentifier()
		mockISO := &mockNeoGeoCDISO{
			uuid:     "2024-01-01-00-00-00-00",
			volumeID: "NEOGEOCD",
		}

		result, err := id.identifyFromISO(mockISO, nil)
		if err != nil {
			t.Fatalf("identifyFromISO() error = %v", err)
		}

		if result.Metadata["uuid"] != "2024-01-01-00-00-00-00" {
			t.Errorf("uuid = %q, want %q", result.Metadata["uuid"], "2024-01-01-00-00-00-00")
		}
		if result.Metadata["volume_ID"] != "NEOGEOCD" {
			t.Errorf("volume_ID = %q, want %q", result.Metadata["volume_ID"], "NEOGEOCD")
		}
		if result.ID != "NEOGEOCD" {
			t.Errorf("ID = %q, want %q", result.ID, "NEOGEOCD")
		}
	})

	t.Run("tuple database lookup", func(t *testing.T) {
		t.Parallel()
		id := NewNeoGeoCDIdentifier()
		mockISO := &mockNeoGeoCDISO{
			uuid:     "test-uuid",
			volumeID: "GAME_VOL",
		}

		// Create mock database that supports Lookup with struct key
		db := &mockNeoGeoCDDatabase{
			tupleEntries: map[string]map[string]string{
				"test-uuid:GAME_VOL": {"title": "Test Game", "ID": "NGCD-001"},
			},
		}

		result, err := id.identifyFromISO(mockISO, db)
		if err != nil {
			t.Fatalf("identifyFromISO() error = %v", err)
		}

		if result.Title != "Test Game" {
			t.Errorf("Title = %q, want %q", result.Title, "Test Game")
		}
	})

	t.Run("volumeID fallback", func(t *testing.T) {
		t.Parallel()
		id := NewNeoGeoCDIdentifier()
		mockISO := &mockNeoGeoCDISO{
			uuid:     "unknown-uuid",
			volumeID: "KNOWN_VOL",
		}

		db := &mockNeoGeoCDDatabase{
			stringEntries: map[string]map[string]string{
				"KNOWN_VOL": {"title": "Fallback Game"},
			},
		}

		result, err := id.identifyFromISO(mockISO, db)
		if err != nil {
			t.Fatalf("identifyFromISO() error = %v", err)
		}

		if result.Title != "Fallback Game" {
			t.Errorf("Title = %q, want %q", result.Title, "Fallback Game")
		}
	})

	t.Run("no database match", func(t *testing.T) {
		t.Parallel()
		id := NewNeoGeoCDIdentifier()
		mockISO := &mockNeoGeoCDISO{
			uuid:     "unknown-uuid",
			volumeID: "UNKNOWN_VOL",
		}

		db := &mockNeoGeoCDDatabase{}

		result, err := id.identifyFromISO(mockISO, db)
		if err != nil {
			t.Fatalf("identifyFromISO() error = %v", err)
		}

		// ID should fall back to volumeID
		if result.ID != "UNKNOWN_VOL" {
			t.Errorf("ID = %q, want %q", result.ID, "UNKNOWN_VOL")
		}
	})
}

// TestNeoGeoCDIdentifier_RealISO tests with real test data if available.
func TestNeoGeoCDIdentifier_RealISO(t *testing.T) {
	t.Parallel()

	isoPath := filepath.Join("..", "testdata", "NeoGeoCD", "240pTestSuite.iso")

	id := NewNeoGeoCDIdentifier()
	result, err := id.IdentifyFromPath(isoPath, nil)
	if err != nil {
		t.Skipf("Skipping real ISO test (file not available or error): %v", err)
	}

	// Just verify it returns some result
	if result == nil {
		t.Fatal("IdentifyFromPath() returned nil result")
	}
	if result.Console != ConsoleNeoGeoCD {
		t.Errorf("Console = %v, want %v", result.Console, ConsoleNeoGeoCD)
	}
}

// mockNeoGeoCDDatabase implements Database for NeoGeoCD testing.
type mockNeoGeoCDDatabase struct {
	tupleEntries  map[string]map[string]string // "uuid:volumeID" -> entry
	stringEntries map[string]map[string]string // volumeID -> entry
}

func (m *mockNeoGeoCDDatabase) Lookup(console Console, key any) (map[string]string, bool) {
	if console != ConsoleNeoGeoCD || m.tupleEntries == nil {
		return nil, false
	}

	// Key is struct{uuid, volumeID string} - use reflection to extract fields
	// since the struct is defined locally in the identifier code
	val := reflect.ValueOf(key)
	if val.Kind() != reflect.Struct {
		return nil, false
	}

	uuidField := val.FieldByName("uuid")
	volumeIDField := val.FieldByName("volumeID")

	if !uuidField.IsValid() || !volumeIDField.IsValid() {
		return nil, false
	}

	lookupKey := uuidField.String() + ":" + volumeIDField.String()
	entry, found := m.tupleEntries[lookupKey]
	return entry, found
}

func (m *mockNeoGeoCDDatabase) LookupByString(console Console, key string) (map[string]string, bool) {
	if console != ConsoleNeoGeoCD || m.stringEntries == nil {
		return nil, false
	}
	entry, found := m.stringEntries[key]
	return entry, found
}

func (*mockNeoGeoCDDatabase) GetIDPrefixes(_ Console) []string {
	return nil
}
