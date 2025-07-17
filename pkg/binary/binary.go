package binary

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadUint16BE reads a big-endian uint16 from reader
func ReadUint16BE(r io.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(r, binary.BigEndian, &value)
	if err != nil {
		return 0, fmt.Errorf("failed to read uint16 BE: %w", err)
	}
	return value, nil
}

// ReadUint16LE reads a little-endian uint16 from reader
func ReadUint16LE(r io.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(r, binary.LittleEndian, &value)
	if err != nil {
		return 0, fmt.Errorf("failed to read uint16 LE: %w", err)
	}
	return value, nil
}

// ReadUint32BE reads a big-endian uint32 from reader
func ReadUint32BE(r io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(r, binary.BigEndian, &value)
	if err != nil {
		return 0, fmt.Errorf("failed to read uint32 BE: %w", err)
	}
	return value, nil
}

// ReadUint32LE reads a little-endian uint32 from reader
func ReadUint32LE(r io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(r, binary.LittleEndian, &value)
	if err != nil {
		return 0, fmt.Errorf("failed to read uint32 LE: %w", err)
	}
	return value, nil
}

// ExtractString extracts a string from byte slice at given offset and length
func ExtractString(data []byte, offset, length int) string {
	if offset < 0 || length < 0 || offset+length > len(data) {
		return ""
	}
	return string(data[offset : offset+length])
}

// CleanString removes non-printable characters and null-terminates at first 0x00
func CleanString(data []byte) string {
	result := make([]byte, 0, len(data))

	for _, b := range data {
		if b == 0 {
			// Null termination
			break
		}
		if b >= 0x20 && b <= 0x7E {
			// Printable ASCII
			result = append(result, b)
		} else {
			// Replace non-printable with space
			result = append(result, ' ')
		}
	}

	return string(result)
}

// CalculateChecksum8 calculates 8-bit checksum (sum of bytes)
func CalculateChecksum8(data []byte) uint8 {
	var sum uint8
	for _, b := range data {
		sum += b
	}
	return sum
}

// CalculateChecksum16 calculates 16-bit checksum (sum of bytes)
func CalculateChecksum16(data []byte) uint16 {
	var sum uint16
	for _, b := range data {
		sum += uint16(b)
	}
	return sum
}

// N64EndianSwap swaps every pair of bytes (for N64 ROM endianness conversion)
func N64EndianSwap(data []byte) []byte {
	if len(data)%2 != 0 {
		panic("N64EndianSwap requires even-length data")
	}

	result := make([]byte, len(data))
	for i := 0; i < len(data); i += 2 {
		result[i] = data[i+1]
		result[i+1] = data[i]
	}
	return result
}
