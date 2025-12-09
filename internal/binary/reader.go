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

// Package binary provides utilities for reading binary data from ROM and disc images.
package binary

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// ReadAt reads len(buf) bytes from r at offset.
func ReadAt(r io.ReaderAt, offset int64, buf []byte) error {
	_, err := r.ReadAt(buf, offset)
	if err != nil {
		return fmt.Errorf("read at offset %d: %w", offset, err)
	}
	return nil
}

// ReadBytesAt reads n bytes from r at offset.
func ReadBytesAt(r io.ReaderAt, offset int64, n int) ([]byte, error) {
	buf := make([]byte, n)
	if err := ReadAt(r, offset, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadUint8At reads a single byte from r at offset.
func ReadUint8At(r io.ReaderAt, offset int64) (uint8, error) {
	buf := make([]byte, 1)
	if err := ReadAt(r, offset, buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// ReadUint16LEAt reads a little-endian uint16 from r at offset.
func ReadUint16LEAt(r io.ReaderAt, offset int64) (uint16, error) {
	buf := make([]byte, 2)
	if err := ReadAt(r, offset, buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}

// ReadUint16BEAt reads a big-endian uint16 from r at offset.
func ReadUint16BEAt(r io.ReaderAt, offset int64) (uint16, error) {
	buf := make([]byte, 2)
	if err := ReadAt(r, offset, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf), nil
}

// ReadUint32LEAt reads a little-endian uint32 from r at offset.
func ReadUint32LEAt(r io.ReaderAt, offset int64) (uint32, error) {
	buf := make([]byte, 4)
	if err := ReadAt(r, offset, buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

// ReadUint32BEAt reads a big-endian uint32 from r at offset.
func ReadUint32BEAt(r io.ReaderAt, offset int64) (uint32, error) {
	buf := make([]byte, 4)
	if err := ReadAt(r, offset, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf), nil
}

// ReadStringAt reads a string of n bytes from r at offset, trimming null bytes and spaces.
func ReadStringAt(r io.ReaderAt, offset int64, n int) (string, error) {
	buf, err := ReadBytesAt(r, offset, n)
	if err != nil {
		return "", err
	}
	return CleanString(buf), nil
}

// ReadPrintableStringAt reads a string keeping only printable ASCII characters.
func ReadPrintableStringAt(r io.ReaderAt, offset int64, n int) (string, error) {
	buf, err := ReadBytesAt(r, offset, n)
	if err != nil {
		return "", err
	}
	return ExtractPrintable(buf), nil
}

// CleanString converts bytes to a string, trimming null bytes and whitespace.
func CleanString(data []byte) string {
	// Find the first null byte
	end := len(data)
	for i, c := range data {
		if c == 0 {
			end = i
			break
		}
	}
	return strings.TrimSpace(string(data[:end]))
}

// ExtractPrintable extracts only printable ASCII characters (0x20-0x7E) from bytes.
func ExtractPrintable(b []byte) string {
	var result strings.Builder
	for _, c := range b {
		if c >= 0x20 && c <= 0x7E {
			_ = result.WriteByte(c)
		}
	}
	return strings.TrimSpace(result.String())
}

// BytesEqual compares two byte slices for equality.
func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FindBytes searches for needle in haystack and returns the offset, or -1 if not found.
func FindBytes(haystack, needle []byte) int {
	if len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if BytesEqual(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

// FindBytesInRange searches for needle in reader between start and end offsets.
// Returns the absolute offset or -1 if not found.
func FindBytesInRange(reader io.ReaderAt, start, end int64, needle []byte) (int64, error) {
	size := end - start
	if size <= 0 {
		return -1, nil
	}
	buf := make([]byte, size)
	if _, err := reader.ReadAt(buf, start); err != nil && err != io.EOF {
		return -1, fmt.Errorf("read bytes in range %d-%d: %w", start, end, err)
	}
	idx := FindBytes(buf, needle)
	if idx == -1 {
		return -1, nil
	}
	return start + int64(idx), nil
}
