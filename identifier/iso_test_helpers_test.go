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

package identifier

import "encoding/binary"

type testISOFile struct {
	name string
	data []byte
}

func createIdentifierTestISO(volumeID string, files []testISOFile) []byte {
	const blockSize = 2048
	totalBlocks := 20 + len(files)
	data := make([]byte, totalBlocks*blockSize)
	pvdOffset := 16 * blockSize

	data[pvdOffset] = 0x01
	copy(data[pvdOffset+1:], "CD001")
	data[pvdOffset+6] = 0x01
	copy(data[pvdOffset+8:], "PLAYSTATION")
	copy(data[pvdOffset+40:], volumeID)
	binary.LittleEndian.PutUint32(data[pvdOffset+80:], mustUint32(totalBlocks))
	binary.BigEndian.PutUint32(data[pvdOffset+84:], mustUint32(totalBlocks))
	binary.LittleEndian.PutUint16(data[pvdOffset+120:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+122:], 1)
	binary.LittleEndian.PutUint16(data[pvdOffset+124:], 1)
	binary.BigEndian.PutUint16(data[pvdOffset+126:], 1)
	binary.LittleEndian.PutUint16(data[pvdOffset+128:], blockSize)
	binary.BigEndian.PutUint16(data[pvdOffset+130:], blockSize)
	binary.LittleEndian.PutUint32(data[pvdOffset+132:], 10)
	binary.BigEndian.PutUint32(data[pvdOffset+136:], 10)
	binary.LittleEndian.PutUint32(data[pvdOffset+140:], 18)
	copy(data[pvdOffset+813:], "2024010112000000")

	writeIdentifierDirectoryRecord(data[pvdOffset+156:], 19, blockSize, "\x00")

	pathTableOffset := 18 * blockSize
	data[pathTableOffset] = 1
	binary.LittleEndian.PutUint32(data[pathTableOffset+2:], 19)
	binary.LittleEndian.PutUint16(data[pathTableOffset+6:], 1)
	data[pathTableOffset+8] = 0x00

	rootOffset := 19 * blockSize
	writeIdentifierDirectoryRecord(data[rootOffset:], 19, blockSize, "\x00")
	writeIdentifierDirectoryRecord(data[rootOffset+34:], 19, blockSize, "\x01")

	recordOffset := rootOffset + 68
	for idx, file := range files {
		fileLBA := 20 + idx
		writeIdentifierFileRecord(data[recordOffset:], fileLBA, len(file.data), file.name)
		copy(data[fileLBA*blockSize:], file.data)
		recordOffset += directoryRecordLength(file.name)
	}

	return data
}

func writeIdentifierDirectoryRecord(record []byte, lba, size int, name string) {
	writeIdentifierRecord(record, lba, size, name, 0x02)
}

func writeIdentifierFileRecord(record []byte, lba, size int, name string) {
	writeIdentifierRecord(record, lba, size, name, 0x00)
}

func writeIdentifierRecord(record []byte, lba, size int, name string, flags byte) {
	recLen := directoryRecordLength(name)
	record[0] = mustByte(recLen)
	binary.LittleEndian.PutUint32(record[2:], mustUint32(lba))
	binary.BigEndian.PutUint32(record[6:], mustUint32(lba))
	binary.LittleEndian.PutUint32(record[10:], mustUint32(size))
	binary.BigEndian.PutUint32(record[14:], mustUint32(size))
	record[25] = flags
	binary.LittleEndian.PutUint16(record[28:], 1)
	binary.BigEndian.PutUint16(record[30:], 1)
	record[32] = mustByte(len(name))
	copy(record[33:], name)
}

func directoryRecordLength(name string) int {
	recLen := 33 + len(name)
	if recLen%2 == 1 {
		recLen++
	}
	return recLen
}

func mustUint32(value int) uint32 {
	if value < 0 || value > 1<<32-1 {
		panic("test ISO value exceeds uint32")
	}
	return uint32(value)
}

func mustByte(value int) byte {
	if value < 0 || value > 1<<8-1 {
		panic("test ISO value exceeds byte")
	}
	return byte(value)
}
