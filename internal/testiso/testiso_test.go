// Copyright (c) 2026 Niema Moshiri and The Zaparoo Project.
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

package testiso

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

func TestCreateMinimalWithFiles(t *testing.T) {
	t.Parallel()

	isoData := CreateMinimal(t, "TESTVOL", "TESTSYS", "TESTPUB", []File{
		{Name: "FIRST.TXT;1", Data: []byte("first file")},
		{Name: "SECOND.TXT;1", Data: []byte("second file")},
	})

	iso, err := iso9660.OpenReader(bytes.NewReader(isoData), int64(len(isoData)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	if got := iso.GetVolumeID(); !strings.HasPrefix(got, "TESTVOL") {
		t.Errorf("GetVolumeID() = %q, want prefix %q", got, "TESTVOL")
	}
	if got := iso.GetSystemID(); !strings.HasPrefix(got, "TESTSYS") {
		t.Errorf("GetSystemID() = %q, want prefix %q", got, "TESTSYS")
	}
	if got := iso.GetPublisherID(); !strings.HasPrefix(got, "TESTPUB") {
		t.Errorf("GetPublisherID() = %q, want prefix %q", got, "TESTPUB")
	}

	files, err := iso.IterFiles(true)
	if err != nil {
		t.Fatalf("IterFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("IterFiles() found %d files, want 2", len(files))
	}
	if files[0].Path != "/FIRST.TXT;1" || files[1].Path != "/SECOND.TXT;1" {
		t.Errorf("IterFiles() paths = %q, %q", files[0].Path, files[1].Path)
	}

	data, err := iso.ReadFileByPath("SECOND.TXT")
	if err != nil {
		t.Fatalf("ReadFileByPath() error = %v", err)
	}
	if string(data) != "second file" {
		t.Errorf("ReadFileByPath() = %q, want %q", data, "second file")
	}
}

func TestCreateMinimalTruncatesIdentifiers(t *testing.T) {
	t.Parallel()

	isoData := CreateMinimal(t,
		"VOLUME-ID-THAT-IS-LONGER-THAN-THIRTY-TWO-BYTES",
		"SYSTEM-ID-THAT-IS-LONGER-THAN-THIRTY-TWO-BYTES",
		"PUBLISHER-ID-THAT-FITS",
		nil,
	)

	iso, err := iso9660.OpenReader(bytes.NewReader(isoData), int64(len(isoData)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer func() { _ = iso.Close() }()

	if len(iso.GetVolumeID()) < 32 || !strings.HasPrefix(iso.GetVolumeID(), "VOLUME-ID") {
		t.Errorf("GetVolumeID() = %q, want truncated volume ID", iso.GetVolumeID())
	}
	if len(iso.GetSystemID()) < 32 || !strings.HasPrefix(iso.GetSystemID(), "SYSTEM-ID") {
		t.Errorf("GetSystemID() = %q, want truncated system ID", iso.GetSystemID())
	}
}

func TestWriteFileRecordExactBuffer(t *testing.T) {
	t.Parallel()

	const fileName = "FILE.BIN;1"
	record := make([]byte, DirectoryRecordLength(fileName))

	WriteFileRecord(t, record, 20, 4, fileName)

	if int(record[0]) != len(record) {
		t.Errorf("record length = %d, want %d", record[0], len(record))
	}
	if record[25] != 0 {
		t.Errorf("file flags = %#x, want 0", record[25])
	}
	if got := string(record[33 : 33+len(fileName)]); got != fileName {
		t.Errorf("file name = %q, want %q", got, fileName)
	}
}

func TestDirectoryRecordLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int
	}{
		{"A", 34},
		{"AB", 36},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DirectoryRecordLength(tt.name); got != tt.want {
				t.Errorf("DirectoryRecordLength(%q) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}
