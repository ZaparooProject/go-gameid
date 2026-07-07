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

// Package testiso builds small ISO9660 images for tests.
package testiso

import (
	"encoding/binary"
	"testing"
)

// BlockSize is the logical sector size used by generated test ISOs.
const BlockSize = 2048

// File describes a root-level file to add to a generated test ISO.
type File struct {
	Name string
	Data []byte
}

// CreateMinimal returns a minimal ISO9660 image with optional root-level files.
func CreateMinimal(tb testing.TB, volumeID, systemID, publisherID string, files []File) []byte {
	tb.Helper()

	totalBlocks := 20 + len(files)
	data := make([]byte, totalBlocks*BlockSize)
	pvdOffset := 16 * BlockSize

	data[pvdOffset] = 0x01
	copy(data[pvdOffset+1:], "CD001")
	data[pvdOffset+6] = 0x01
	copyBounded(data[pvdOffset+8:], systemID, 32)
	copyBounded(data[pvdOffset+40:], volumeID, 32)
	binary.LittleEndian.PutUint32(data[pvdOffset+80:], mustUint32(tb, totalBlocks))
	binary.BigEndian.PutUint32(data[pvdOffset+84:], mustUint32(tb, totalBlocks))
	binary.LittleEndian.PutUint16(data[pvdOffset+120:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+122:], 1)
	binary.LittleEndian.PutUint16(data[pvdOffset+124:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+126:], 1)
	binary.LittleEndian.PutUint16(data[pvdOffset+128:], BlockSize)
	binary.BigEndian.PutUint16(data[pvdOffset+130:], BlockSize)
	binary.LittleEndian.PutUint32(data[pvdOffset+132:], 10)
	binary.BigEndian.PutUint32(data[pvdOffset+136:], 10)
	binary.LittleEndian.PutUint32(data[pvdOffset+140:], 18)
	copyBounded(data[pvdOffset+318:], publisherID, 128)
	copy(data[pvdOffset+813:], "2024010112000000")

	WriteDirectoryRecord(tb, data[pvdOffset+156:], 19, BlockSize, "\x00")
	writePathTable(data)
	writeRootDirectory(tb, data, files)

	return data
}

func copyBounded(dst []byte, value string, maxLen int) {
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	copy(dst, value)
}

func writePathTable(data []byte) {
	pathTableOffset := 18 * BlockSize
	data[pathTableOffset] = 1
	binary.LittleEndian.PutUint32(data[pathTableOffset+2:], 19)
	binary.LittleEndian.PutUint16(data[pathTableOffset+6:], 1)
	data[pathTableOffset+8] = 0x00
}

func writeRootDirectory(tb testing.TB, data []byte, files []File) {
	tb.Helper()

	rootOffset := 19 * BlockSize
	WriteDirectoryRecord(tb, data[rootOffset:], 19, BlockSize, "\x00")
	WriteDirectoryRecord(tb, data[rootOffset+34:], 19, BlockSize, "\x01")

	recordOffset := rootOffset + 68
	for idx, file := range files {
		recordLen := DirectoryRecordLength(file.Name)
		if recordOffset+recordLen > rootOffset+BlockSize {
			tb.Fatalf("test ISO root directory records exceed one block at file %s", file.Name)
		}
		if len(file.Data) > BlockSize {
			tb.Fatalf("test ISO file %s is %d bytes, max one block (%d)", file.Name, len(file.Data), BlockSize)
		}
		fileLBA := 20 + idx
		WriteFileRecord(tb, data[recordOffset:], fileLBA, len(file.Data), file.Name)
		copy(data[fileLBA*BlockSize:fileLBA*BlockSize+len(file.Data)], file.Data)
		recordOffset += recordLen
	}
}

// WriteDirectoryRecord writes an ISO9660 directory record into record.
func WriteDirectoryRecord(tb testing.TB, record []byte, lba, size int, name string) {
	tb.Helper()

	writeRecord(tb, record, lba, size, name)
	record[25] = 0x02
}

// WriteFileRecord writes an ISO9660 file record into record.
func WriteFileRecord(tb testing.TB, record []byte, lba, size int, name string) {
	tb.Helper()

	writeRecord(tb, record, lba, size, name)
}

func writeRecord(tb testing.TB, record []byte, lba, size int, name string) {
	tb.Helper()

	recLen := DirectoryRecordLength(name)
	if len(record) < recLen {
		tb.Fatalf("test ISO directory record buffer for %s is %d bytes, need %d", name, len(record), recLen)
	}
	record[0] = mustByte(tb, recLen)
	binary.LittleEndian.PutUint32(record[2:], mustUint32(tb, lba))
	binary.BigEndian.PutUint32(record[6:], mustUint32(tb, lba))
	binary.LittleEndian.PutUint32(record[10:], mustUint32(tb, size))
	binary.BigEndian.PutUint32(record[14:], mustUint32(tb, size))
	binary.LittleEndian.PutUint16(record[28:], 1)
	binary.BigEndian.PutUint16(record[30:], 1)
	record[32] = mustByte(tb, len(name))
	copy(record[33:], name)
}

// DirectoryRecordLength returns an even-padded ISO9660 directory record length.
func DirectoryRecordLength(name string) int {
	recLen := 33 + len(name)
	if recLen%2 == 1 {
		recLen++
	}
	return recLen
}

func mustUint32(tb testing.TB, value int) uint32 {
	tb.Helper()

	if value < 0 || value > 1<<32-1 {
		tb.Fatalf("test ISO value %d exceeds uint32", value)
	}
	return uint32(value) //nolint:gosec // Bounds checked above.
}

func mustByte(tb testing.TB, value int) byte {
	tb.Helper()

	if value < 0 || value > 1<<8-1 {
		tb.Fatalf("test ISO value %d exceeds byte", value)
	}
	return byte(value) //nolint:gosec // Bounds checked above.
}
