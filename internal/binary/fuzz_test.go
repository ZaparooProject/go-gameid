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

// FuzzFindBytes fuzzes the byte pattern search function.
func FuzzFindBytes(f *testing.F) {
	// Add corpus seeds
	f.Add([]byte("hello world"), []byte("world"))
	f.Add([]byte("hello world"), []byte("xyz"))
	f.Add([]byte("aaa"), []byte("a"))
	f.Add([]byte{}, []byte{})
	f.Add([]byte("test"), []byte{})
	f.Add([]byte{}, []byte("test"))
	f.Add([]byte{0x00, 0x01, 0x02}, []byte{0x01, 0x02})
	f.Add([]byte("abcabc"), []byte("abc"))

	f.Fuzz(func(t *testing.T, haystack, needle []byte) {
		// FindBytes should never panic
		idx := FindBytes(haystack, needle)

		// Verify correctness: if found, needle must actually be at that position
		if idx >= 0 {
			if idx+len(needle) > len(haystack) {
				t.Errorf("FindBytes returned invalid index %d for haystack len %d, needle len %d",
					idx, len(haystack), len(needle))
				return
			}
			if !bytes.Equal(haystack[idx:idx+len(needle)], needle) {
				t.Errorf("FindBytes returned index %d but needle not found there", idx)
			}
		}

		// If needle is empty, behavior depends on implementation
		// If haystack is shorter than needle, must return -1
		if len(needle) > len(haystack) && idx != -1 {
			t.Error("FindBytes should return -1 when needle is longer than haystack")
		}
	})
}

// FuzzFindBytesInRange fuzzes ranged byte search.
func FuzzFindBytesInRange(f *testing.F) {
	// Add corpus seeds
	f.Add([]byte("hello world"), int64(0), int64(11), []byte("world"))
	f.Add([]byte("hello world"), int64(6), int64(11), []byte("world"))
	f.Add([]byte("hello world"), int64(0), int64(5), []byte("world"))
	f.Add([]byte("test"), int64(0), int64(0), []byte("t"))
	f.Add([]byte("test"), int64(5), int64(3), []byte("t")) // start > end
	f.Add([]byte{}, int64(0), int64(0), []byte{})

	f.Fuzz(func(t *testing.T, data []byte, start, end int64, needle []byte) {
		// Limit input size
		if len(data) > 1024*1024 {
			return
		}

		reader := bytes.NewReader(data)

		// Should never panic
		idx, err := FindBytesInRange(reader, start, end, needle)

		// Basic sanity checks
		if err == nil && idx >= 0 {
			// Index should be >= start if valid
			if idx < start {
				t.Errorf("FindBytesInRange returned index %d which is less than start %d", idx, start)
			}
		}
	})
}

// FuzzCleanString fuzzes string cleaning.
func FuzzCleanString(f *testing.F) {
	// Add corpus seeds
	f.Add([]byte("hello\x00world"))
	f.Add([]byte("  trimmed  "))
	f.Add([]byte{0x00, 0x00, 0x00})
	f.Add([]byte{})
	f.Add([]byte("normal string"))
	f.Add([]byte{0x20, 0x20, 0x00, 0x41, 0x42}) // Spaces then null then data

	f.Fuzz(func(t *testing.T, data []byte) {
		// CleanString should never panic
		result := CleanString(data)

		// Result should not contain null bytes
		for _, c := range result {
			if c == 0 {
				t.Error("CleanString result contains null byte")
			}
		}
	})
}

// FuzzExtractPrintable fuzzes printable character extraction.
func FuzzExtractPrintable(f *testing.F) {
	// Add corpus seeds
	f.Add([]byte("Hello World!"))
	f.Add([]byte{0x00, 0x01, 0x41, 0x42, 0xFF, 0x43})
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x02, 0x03}) // No printable chars
	f.Add([]byte{0x20, 0x7E})             // Space and tilde (bounds)
	f.Add([]byte{0x1F, 0x7F})             // Just outside printable range

	f.Fuzz(func(t *testing.T, data []byte) {
		// ExtractPrintable should never panic
		result := ExtractPrintable(data)

		// Result should only contain printable ASCII (0x20-0x7E)
		for _, c := range result {
			if c < 0x20 || c > 0x7E {
				// After TrimSpace, only printable chars should remain
				// But TrimSpace removes leading/trailing spaces, so interior should be printable
				if c != ' ' {
					t.Errorf("ExtractPrintable result contains non-printable char: 0x%02X", c)
				}
			}
		}
	})
}

// FuzzBytesEqual fuzzes byte slice comparison.
func FuzzBytesEqual(f *testing.F) {
	f.Add([]byte("test"), []byte("test"))
	f.Add([]byte("test"), []byte("tests"))
	f.Add([]byte{}, []byte{})
	f.Add([]byte{0x00}, []byte{0x00})

	f.Fuzz(func(t *testing.T, first, second []byte) {
		// BytesEqual should never panic
		result := BytesEqual(first, second)

		// Verify correctness
		expected := bytes.Equal(first, second)
		if result != expected {
			t.Errorf("BytesEqual(%v, %v) = %v, want %v", first, second, result, expected)
		}
	})
}
