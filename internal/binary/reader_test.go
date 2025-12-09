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

package binary

import (
	"bytes"
	"testing"
)

func TestReadUint8At(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x42, 0xFF, 0x80}
	reader := bytes.NewReader(data)

	tests := []struct {
		name    string
		offset  int64
		want    uint8
		wantErr bool
	}{
		{"first byte (0x00)", 0, 0x00, false},
		{"second byte (0x42)", 1, 0x42, false},
		{"third byte (0xFF)", 2, 0xFF, false},
		{"fourth byte (0x80)", 3, 0x80, false},
		{"past end", 4, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadUint8At(reader, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint8At() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadUint8At() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestReadBytesAt(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	reader := bytes.NewReader(data)

	tests := []struct {
		name    string
		want    []byte
		offset  int64
		length  int
		wantErr bool
	}{
		{name: "read from start", offset: 0, length: 3, want: []byte{0x00, 0x01, 0x02}, wantErr: false},
		{name: "read from middle", offset: 2, length: 3, want: []byte{0x02, 0x03, 0x04}, wantErr: false},
		{name: "read to end", offset: 3, length: 3, want: []byte{0x03, 0x04, 0x05}, wantErr: false},
		{name: "read past end", offset: 4, length: 5, want: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadBytesAt(reader, testCase.offset, testCase.length)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ReadBytesAt() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if !testCase.wantErr && !bytes.Equal(got, testCase.want) {
				t.Errorf("ReadBytesAt() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:dupl // Similar test structure for uint16 reading is intentional
func TestReadUint16LEAt(t *testing.T) {
	t.Parallel()

	data := []byte{0x34, 0x12, 0x78, 0x56}
	reader := bytes.NewReader(data)

	tests := []struct {
		name   string
		offset int64
		want   uint16
	}{
		{"first value", 0, 0x1234},
		{"second value", 2, 0x5678},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadUint16LEAt(reader, testCase.offset)
			if err != nil {
				t.Errorf("ReadUint16LEAt() error = %v", err)
				return
			}
			if got != testCase.want {
				t.Errorf("ReadUint16LEAt() = 0x%04X, want 0x%04X", got, testCase.want)
			}
		})
	}
}

//nolint:dupl // Similar test structure for uint16 reading is intentional
func TestReadUint16BEAt(t *testing.T) {
	t.Parallel()

	data := []byte{0x12, 0x34, 0x56, 0x78}
	reader := bytes.NewReader(data)

	tests := []struct {
		name   string
		offset int64
		want   uint16
	}{
		{"first value", 0, 0x1234},
		{"second value", 2, 0x5678},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadUint16BEAt(reader, testCase.offset)
			if err != nil {
				t.Errorf("ReadUint16BEAt() error = %v", err)
				return
			}
			if got != testCase.want {
				t.Errorf("ReadUint16BEAt() = 0x%04X, want 0x%04X", got, testCase.want)
			}
		})
	}
}

func TestReadUint32LEAt(t *testing.T) {
	t.Parallel()

	data := []byte{0x78, 0x56, 0x34, 0x12}
	reader := bytes.NewReader(data)

	got, err := ReadUint32LEAt(reader, 0)
	if err != nil {
		t.Errorf("ReadUint32LEAt() error = %v", err)
		return
	}
	want := uint32(0x12345678)
	if got != want {
		t.Errorf("ReadUint32LEAt() = 0x%08X, want 0x%08X", got, want)
	}
}

func TestReadUint32BEAt(t *testing.T) {
	t.Parallel()

	data := []byte{0x12, 0x34, 0x56, 0x78}
	reader := bytes.NewReader(data)

	got, err := ReadUint32BEAt(reader, 0)
	if err != nil {
		t.Errorf("ReadUint32BEAt() error = %v", err)
		return
	}
	want := uint32(0x12345678)
	if got != want {
		t.Errorf("ReadUint32BEAt() = 0x%08X, want 0x%08X", got, want)
	}
}

func TestReadStringAt(t *testing.T) {
	t.Parallel()

	data := []byte("Hello, World!\x00More text")
	reader := bytes.NewReader(data)

	tests := []struct {
		name   string
		want   string
		offset int64
		length int
	}{
		{name: "full string", offset: 0, length: 13, want: "Hello, World!"},
		{name: "substring", offset: 0, length: 5, want: "Hello"},
		{name: "from middle", offset: 7, length: 6, want: "World!"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadStringAt(reader, testCase.offset, testCase.length)
			if err != nil {
				t.Errorf("ReadStringAt() error = %v", err)
				return
			}
			if got != testCase.want {
				t.Errorf("ReadStringAt() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReadPrintableStringAt(t *testing.T) {
	t.Parallel()

	// Create test data with mixed printable and non-printable characters
	data := []byte("Hello\x01World\x00Test\x80More")
	reader := bytes.NewReader(data)

	tests := []struct {
		name    string
		want    string
		offset  int64
		length  int
		wantErr bool
	}{
		{name: "all printable", offset: 0, length: 5, want: "Hello", wantErr: false},
		{name: "with control char", offset: 0, length: 11, want: "HelloWorld", wantErr: false},
		{name: "with null", offset: 0, length: 16, want: "HelloWorldTest", wantErr: false},
		{name: "from middle", offset: 6, length: 5, want: "World", wantErr: false},
		{name: "past end", offset: 100, length: 5, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadPrintableStringAt(reader, tt.offset, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadPrintableStringAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadPrintableStringAt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		input []byte
	}{
		{name: "normal string", input: []byte("Hello"), want: "Hello"},
		{name: "with null terminator", input: []byte("Hello\x00World"), want: "Hello"},
		{name: "padded with nulls", input: []byte("Test\x00\x00\x00"), want: "Test"},
		{name: "with trailing spaces", input: []byte("Test   "), want: "Test"},
		{name: "with leading spaces", input: []byte("   Test"), want: "Test"},
		{name: "with both", input: []byte("  Test  \x00"), want: "Test"},
		{name: "empty", input: []byte{}, want: ""},
		{name: "only nulls", input: []byte{0, 0, 0}, want: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := CleanString(testCase.input)
			if got != testCase.want {
				t.Errorf("CleanString() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestExtractPrintable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		input []byte
	}{
		{name: "normal ASCII", input: []byte("Hello"), want: "Hello"},
		{name: "with control chars", input: []byte("Hello\x01World"), want: "HelloWorld"},
		{name: "with high bytes", input: []byte("Test\x80\x90"), want: "Test"},
		{name: "spaces preserved", input: []byte("Hello World"), want: "Hello World"},
		{name: "numbers and symbols", input: []byte("Test123!@#"), want: "Test123!@#"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ExtractPrintable(testCase.input)
			if got != testCase.want {
				t.Errorf("ExtractPrintable() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBytesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"not equal", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"different lengths", []byte{1, 2}, []byte{1, 2, 3}, false},
		{"empty both", []byte{}, []byte{}, true},
		{"empty one", []byte{}, []byte{1}, false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := BytesEqual(testCase.a, testCase.b)
			if got != testCase.want {
				t.Errorf("BytesEqual() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFindBytes(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x01, 0x02}

	tests := []struct {
		name   string
		needle []byte
		want   int
	}{
		{"found at start", []byte{0x00, 0x01}, 0},
		{"found in middle", []byte{0x02, 0x03}, 2},
		{"found at end", []byte{0x01, 0x02}, 1}, // First occurrence
		{"not found", []byte{0xFF, 0xFF}, -1},
		{"single byte", []byte{0x03}, 3},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := FindBytes(data, testCase.needle)
			if got != testCase.want {
				t.Errorf("FindBytes() = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestFindBytesInRange(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x01, 0x02}
	reader := bytes.NewReader(data)

	tests := []struct {
		name   string
		needle []byte
		start  int64
		end    int64
		want   int64
	}{
		{"found in range", []byte{0x01, 0x02}, 0, 4, 1},
		{"not in range", []byte{0x01, 0x02}, 2, 5, -1},
		{"second occurrence", []byte{0x01, 0x02}, 3, 7, 5},
		{"at start of range", []byte{0x02, 0x03}, 2, 6, 2},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := FindBytesInRange(reader, testCase.start, testCase.end, testCase.needle)
			if err != nil {
				t.Errorf("FindBytesInRange() error = %v", err)
				return
			}
			if got != testCase.want {
				t.Errorf("FindBytesInRange() = %d, want %d", got, testCase.want)
			}
		})
	}
}
