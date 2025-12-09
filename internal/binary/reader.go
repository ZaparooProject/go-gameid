// Package binary provides utilities for reading binary data from ROM and disc images.
package binary

import (
	"encoding/binary"
	"io"
	"strings"
)

// ReadAt reads len(buf) bytes from r at offset.
func ReadAt(r io.ReaderAt, offset int64, buf []byte) error {
	_, err := r.ReadAt(buf, offset)
	return err
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
func CleanString(b []byte) string {
	// Find the first null byte
	end := len(b)
	for i, c := range b {
		if c == 0 {
			end = i
			break
		}
	}
	return strings.TrimSpace(string(b[:end]))
}

// ExtractPrintable extracts only printable ASCII characters (0x20-0x7E) from bytes.
func ExtractPrintable(b []byte) string {
	var result strings.Builder
	for _, c := range b {
		if c >= 0x20 && c <= 0x7E {
			result.WriteByte(c)
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

// FindBytesInRange searches for needle in r between start and end offsets.
// Returns the absolute offset or -1 if not found.
func FindBytesInRange(r io.ReaderAt, start, end int64, needle []byte) (int64, error) {
	size := end - start
	if size <= 0 {
		return -1, nil
	}
	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, start); err != nil && err != io.EOF {
		return -1, err
	}
	idx := FindBytes(buf, needle)
	if idx == -1 {
		return -1, nil
	}
	return start + int64(idx), nil
}
