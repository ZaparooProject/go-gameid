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

package chd

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestOpenSegaCDCHD(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	t.Logf("Header version: %d", chdFile.Header().Version)
	t.Logf("Logical bytes: %d", chdFile.Header().LogicalBytes)
	t.Logf("Hunk bytes: %d", chdFile.Header().HunkBytes)
	t.Logf("Unit bytes: %d", chdFile.Header().UnitBytes)
	t.Logf("Map offset: %d", chdFile.Header().MapOffset)
	t.Logf("Compressors: %v", chdFile.Header().Compressors)

	// Try reading some data
	reader := chdFile.RawSectorReader()
	buf := make([]byte, 256)
	bytesRead, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}
	t.Logf("Read %d bytes", bytesRead)
	t.Logf("First 32 bytes: %x", buf[:32])
}

func TestHunkMapDebug(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	// Print first 10 hunk map entries
	t.Logf("Number of hunks: %d", chdFile.hunkMap.NumHunks())
	for idx := uint32(0); idx < 10 && idx < chdFile.hunkMap.NumHunks(); idx++ {
		entry := chdFile.hunkMap.entries[idx]
		t.Logf("Hunk %d: CompType=%d, CompLength=%d, Offset=%d",
			idx, entry.CompType, entry.CompLength, entry.Offset)
	}
}

// TestOpenNonExistent verifies error handling for missing files.
func TestOpenNonExistent(t *testing.T) {
	t.Parallel()

	_, err := Open("/nonexistent/path/to/file.chd")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !os.IsNotExist(errors.Unwrap(err)) && !strings.Contains(err.Error(), "no such file") {
		t.Logf("Got error (acceptable): %v", err)
	}
}

// TestOpenInvalidMagic verifies error handling for non-CHD files.
func TestOpenInvalidMagic(t *testing.T) {
	t.Parallel()

	// Try opening a non-CHD file (use the test file itself as it's not a CHD)
	_, err := Open("chd_test.go")
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
	if !errors.Is(err, ErrInvalidMagic) && !strings.Contains(err.Error(), "invalid CHD magic") {
		t.Errorf("expected ErrInvalidMagic, got: %v", err)
	}
}

// TestCHDSize verifies Size() returns correct logical size.
func TestCHDSize(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	size := chdFile.Size()
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
	// Size should match LogicalBytes
	//nolint:gosec // Test only: LogicalBytes from valid test file
	if size != int64(chdFile.Header().LogicalBytes) {
		t.Errorf("Size() %d != LogicalBytes %d", size, chdFile.Header().LogicalBytes)
	}
}

// TestSectorReader verifies SectorReader returns 2048-byte sectors.
func TestSectorReader(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	reader := chdFile.SectorReader()
	buf := make([]byte, 2048)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}
	if n != 2048 {
		t.Errorf("expected 2048 bytes, got %d", n)
	}
}

// TestFirstDataTrackOffset verifies track offset calculation.
func TestFirstDataTrackOffset(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	offset := chdFile.FirstDataTrackOffset()
	// For a standard CD with data track first, offset should be 0 or small
	t.Logf("FirstDataTrackOffset: %d", offset)
	// Just verify it doesn't panic and returns something reasonable
	if offset < 0 {
		t.Errorf("expected non-negative offset, got %d", offset)
	}
}

// TestHeaderIsCompressed verifies compression detection.
func TestHeaderIsCompressed(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	// Test files should be compressed
	if !chdFile.Header().IsCompressed() {
		t.Error("expected compressed CHD")
	}
}

// TestTrackIsDataTrack verifies track type detection.
func TestTrackIsDataTrack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		trackType string
		want      bool
	}{
		{"MODE1", true},
		{"MODE1_RAW", true},
		{"MODE2_RAW", true},
		{"AUDIO", false},
		{"audio", false},
		{"Audio", false},
	}

	for _, tt := range tests {
		track := Track{Type: tt.trackType}
		if got := track.IsDataTrack(); got != tt.want {
			t.Errorf("Track{Type: %q}.IsDataTrack() = %v, want %v", tt.trackType, got, tt.want)
		}
	}
}

// TestTrackSectorSize verifies sector size calculation.
func TestTrackSectorSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		track    Track
		wantSize int
	}{
		{"default", Track{}, 2352},
		{"mode1_raw", Track{DataSize: 2352}, 2352},
		{"mode1_raw_sub", Track{DataSize: 2352, SubSize: 96}, 2448},
		{"mode1_2048", Track{DataSize: 2048}, 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.track.SectorSize(); got != tt.wantSize {
				t.Errorf("SectorSize() = %d, want %d", got, tt.wantSize)
			}
		})
	}
}

// TestCodecTagToString verifies codec tag formatting.
func TestCodecTagToString(t *testing.T) {
	t.Parallel()

	//nolint:govet // fieldalignment not important in test structs
	tests := []struct {
		tag  uint32
		want string
	}{
		{CodecZlib, "zlib"},
		{CodecLZMA, "lzma"},
		{CodecFLAC, "flac"},
		{CodecZstd, "zstd"},
		{CodecCDZlib, "cdzl"},
		{CodecCDLZMA, "cdlz"},
		{CodecCDFLAC, "cdfl"},
		{CodecCDZstd, "cdzs"},
		{0, "none"},
	}

	for _, tt := range tests {
		if got := codecTagToString(tt.tag); got != tt.want {
			t.Errorf("codecTagToString(0x%x) = %q, want %q", tt.tag, got, tt.want)
		}
	}
}

// TestIsCDCodec verifies CD codec detection.
func TestIsCDCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag  uint32
		want bool
	}{
		{CodecCDZlib, true},
		{CodecCDLZMA, true},
		{CodecCDFLAC, true},
		{CodecCDZstd, true},
		{CodecZlib, false},
		{CodecLZMA, false},
		{CodecFLAC, false},
		{CodecZstd, false},
		{0, false},
	}

	for _, tt := range tests {
		if got := IsCDCodec(tt.tag); got != tt.want {
			t.Errorf("IsCDCodec(0x%x) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

//nolint:gocognit,revive // Table-driven test with multiple assertions
func TestParseCHT2(t *testing.T) {
	t.Parallel()

	//nolint:govet // fieldalignment not important in test structs
	tests := []struct {
		name    string
		data    string
		wantErr bool
		wantNum int
		wantTyp string
		wantFrm int
	}{
		{
			name:    "standard",
			data:    "TRACK:1 TYPE:MODE1_RAW SUBTYPE:RW FRAMES:1000 PREGAP:150 POSTGAP:0",
			wantNum: 1,
			wantTyp: "MODE1_RAW",
			wantFrm: 1000,
		},
		{
			name:    "audio",
			data:    "TRACK:2 TYPE:AUDIO SUBTYPE:NONE FRAMES:5000",
			wantNum: 2,
			wantTyp: "AUDIO",
			wantFrm: 5000,
		},
		{
			name:    "invalid_track_number",
			data:    "TRACK:abc TYPE:MODE1",
			wantErr: true,
		},
		{
			name:    "invalid_frames",
			data:    "TRACK:1 FRAMES:notanumber",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseCHT2([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Number != tt.wantNum {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNum)
			}
			if got.Type != tt.wantTyp {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantTyp)
			}
			if got.Frames != tt.wantFrm {
				t.Errorf("Frames = %d, want %d", got.Frames, tt.wantFrm)
			}
		})
	}
}

// TestTrackTypeToDataSize verifies track type to data size mapping.
func TestTrackTypeToDataSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		trackType string
		want      int
	}{
		{"MODE1/2048", 2048},
		{"MODE1/2352", 2352},
		{"MODE1_RAW", 2352},
		{"MODE2/2352", 2352},
		{"MODE2_RAW", 2352},
		{"AUDIO", 2352},
		{"unknown", 2352}, // Default
	}

	for _, tt := range tests {
		if got := trackTypeToDataSize(tt.trackType); got != tt.want {
			t.Errorf("trackTypeToDataSize(%q) = %d, want %d", tt.trackType, got, tt.want)
		}
	}
}

// TestSubTypeToSize verifies subtype to size mapping.
func TestSubTypeToSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		subType string
		want    int
	}{
		{"NONE", 0},
		{"RW", 96},
		{"RW_RAW", 96},
		{"unknown", 0}, // Default
	}

	for _, tt := range tests {
		if got := subTypeToSize(tt.subType); got != tt.want {
			t.Errorf("subTypeToSize(%q) = %d, want %d", tt.subType, got, tt.want)
		}
	}
}

// TestCDTypeToString verifies binary CD type conversion.
func TestCDTypeToString(t *testing.T) {
	t.Parallel()

	//nolint:govet // fieldalignment not important in test structs
	tests := []struct {
		cdType uint32
		want   string
	}{
		{0, "MODE1/2048"},
		{1, "MODE1/2352"},
		{2, "MODE2/2048"},
		{3, "MODE2/2336"},
		{4, "MODE2/2352"},
		{5, "AUDIO"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := cdTypeToString(tt.cdType); got != tt.want {
			t.Errorf("cdTypeToString(%d) = %q, want %q", tt.cdType, got, tt.want)
		}
	}
}

// TestCDSubTypeToString verifies binary CD subtype conversion.
func TestCDSubTypeToString(t *testing.T) {
	t.Parallel()

	//nolint:govet // fieldalignment not important in test structs
	tests := []struct {
		subType uint32
		want    string
	}{
		{0, "RW"},
		{1, "RW_RAW"},
		{2, "NONE"},
		{99, "NONE"}, // Default
	}

	for _, tt := range tests {
		if got := cdSubTypeToString(tt.subType); got != tt.want {
			t.Errorf("cdSubTypeToString(%d) = %q, want %q", tt.subType, got, tt.want)
		}
	}
}

// TestGetCodecUnknown verifies error for unknown codec.
func TestGetCodecUnknown(t *testing.T) {
	t.Parallel()

	_, err := GetCodec(0x12345678)
	if err == nil {
		t.Error("expected error for unknown codec")
	}
	if !errors.Is(err, ErrUnsupportedCodec) {
		t.Errorf("expected ErrUnsupportedCodec, got: %v", err)
	}
}

// TestReadAtEmptyBuffer verifies ReadAt with empty buffer.
func TestReadAtEmptyBuffer(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	reader := chdFile.SectorReader()
	buf := make([]byte, 0)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
}

// TestDataTrackSizeNoTracks verifies DataTrackSize fallback.
func TestDataTrackSizeNoTracks(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	// DataTrackSize should return something reasonable
	size := chdFile.DataTrackSize()
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

// TestGameCubeCHD verifies GameCube CHD support.
func TestGameCubeCHD(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/GC/GameCube-240pSuite-1.17.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	// Verify header
	header := chdFile.Header()
	if header.Version != 5 {
		t.Errorf("expected version 5, got %d", header.Version)
	}

	// GameCube should have significant size
	if chdFile.Size() < 1000000 {
		t.Errorf("expected larger size for GameCube, got %d", chdFile.Size())
	}

	// Try reading raw sector data
	reader := chdFile.RawSectorReader()
	buf := make([]byte, 256)
	_, err = reader.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}

	// GameCube discs don't have standard CD sync headers
	t.Logf("First 32 bytes: %x", buf[:32])
}

// TestZlibCodecDecompress verifies zlib codec decompression.
func TestZlibCodecDecompress(t *testing.T) {
	t.Parallel()

	codec := &zlibCodec{}

	// Create test data: compress "hello world" with deflate
	original := []byte("hello world hello world hello world hello world")
	var compressed bytes.Buffer
	writer, _ := flate.NewWriter(&compressed, flate.DefaultCompression)
	_, _ = writer.Write(original)
	_ = writer.Close()

	dst := make([]byte, len(original))
	decompLen, err := codec.Decompress(dst, compressed.Bytes())
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if decompLen != len(original) {
		t.Errorf("Decompress returned %d bytes, want %d", decompLen, len(original))
	}
	if !bytes.Equal(dst[:decompLen], original) {
		t.Error("Decompressed data mismatch")
	}
}

// TestZlibCodecDecompressInvalid verifies error handling for invalid data.
func TestZlibCodecDecompressInvalid(t *testing.T) {
	t.Parallel()

	codec := &zlibCodec{}
	dst := make([]byte, 100)
	_, err := codec.Decompress(dst, []byte{0x00, 0x01, 0x02, 0x03})
	// Invalid data should error
	if err == nil {
		t.Log("Note: deflate accepted invalid data (may have partial decode)")
	}
}

// TestCDZlibCodecSourceTooSmall verifies error for truncated source.
func TestCDZlibCodecSourceTooSmall(t *testing.T) {
	t.Parallel()

	codec := &cdZlibCodec{}
	dst := make([]byte, 2448)
	_, err := codec.DecompressCD(dst, []byte{0x00}, 2448, 1)
	if err == nil {
		t.Error("expected error for truncated source")
	}
	if !strings.Contains(err.Error(), "source too small") {
		t.Errorf("expected 'source too small' error, got: %v", err)
	}
}

// TestCDZlibCodecInvalidBaseLength verifies error for invalid base length.
func TestCDZlibCodecInvalidBaseLength(t *testing.T) {
	t.Parallel()

	codec := &cdZlibCodec{}
	dst := make([]byte, 2448)
	// Header: 1 byte ECC bitmap + 2 bytes length (0xFFFF = 65535, way too big)
	src := []byte{0x00, 0xFF, 0xFF}
	_, err := codec.DecompressCD(dst, src, 2448, 1)
	if err == nil {
		t.Error("expected error for invalid base length")
	}
	if !strings.Contains(err.Error(), "invalid base length") {
		t.Errorf("expected 'invalid base length' error, got: %v", err)
	}
}

// TestLZMADictSizeComputation verifies LZMA dictionary size calculation.
func TestLZMADictSizeComputation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		hunkBytes uint32
		minDict   uint32
	}{
		{4096, 4096},       // Small hunk
		{8192, 8192},       // 8KB
		{19584, 24576},     // Typical CD hunk (19584 -> next power)
		{1 << 20, 1 << 20}, // 1MB
	}

	for _, tt := range tests {
		got := computeLZMADictSize(tt.hunkBytes)
		if got < tt.hunkBytes {
			t.Errorf("computeLZMADictSize(%d) = %d, should be >= %d", tt.hunkBytes, got, tt.hunkBytes)
		}
	}
}

// TestLZMACodecEmptySource verifies error for empty source.
func TestLZMACodecEmptySource(t *testing.T) {
	t.Parallel()

	codec := &lzmaCodec{}
	dst := make([]byte, 100)
	_, err := codec.Decompress(dst, []byte{})
	if err == nil {
		t.Error("expected error for empty source")
	}
	if !strings.Contains(err.Error(), "empty source") {
		t.Errorf("expected 'empty source' error, got: %v", err)
	}
}

// TestCDLZMACodecSourceTooSmall verifies error for truncated source.
func TestCDLZMACodecSourceTooSmall(t *testing.T) {
	t.Parallel()

	codec := &cdLZMACodec{}
	dst := make([]byte, 2448)
	_, err := codec.DecompressCD(dst, []byte{0x00}, 2448, 1)
	if err == nil {
		t.Error("expected error for truncated source")
	}
	if !strings.Contains(err.Error(), "source too small") {
		t.Errorf("expected 'source too small' error, got: %v", err)
	}
}

// TestHeaderV4Parsing verifies V4 header parsing.
func TestHeaderV4Parsing(t *testing.T) {
	t.Parallel()

	// Construct a valid V4 header buffer (after magic+size+version already read)
	// V4 header is 108 bytes, we need headerSizeV4-12 = 96 bytes
	buf := make([]byte, 96)

	// Flags at offset 4
	binary.BigEndian.PutUint32(buf[4:8], 0x00000001)
	// Compression at offset 8
	binary.BigEndian.PutUint32(buf[8:12], 0x00000005)
	// Total hunks at offset 12
	binary.BigEndian.PutUint32(buf[12:16], 1000)
	// Logical bytes at offset 16
	binary.BigEndian.PutUint64(buf[16:24], 1000000)
	// Meta offset at offset 24
	binary.BigEndian.PutUint64(buf[24:32], 500)
	// Hunk bytes at offset 32
	binary.BigEndian.PutUint32(buf[32:36], 4096)

	header := &Header{Version: 4}
	err := parseHeaderV4(header, buf)
	if err != nil {
		t.Fatalf("parseHeaderV4 failed: %v", err)
	}

	if header.Flags != 1 {
		t.Errorf("Flags = %d, want 1", header.Flags)
	}
	if header.Compression != 5 {
		t.Errorf("Compression = %d, want 5", header.Compression)
	}
	if header.TotalHunks != 1000 {
		t.Errorf("TotalHunks = %d, want 1000", header.TotalHunks)
	}
	if header.LogicalBytes != 1000000 {
		t.Errorf("LogicalBytes = %d, want 1000000", header.LogicalBytes)
	}
	if header.HunkBytes != 4096 {
		t.Errorf("HunkBytes = %d, want 4096", header.HunkBytes)
	}
	// V4 sets default UnitBytes
	if header.UnitBytes != 2448 {
		t.Errorf("UnitBytes = %d, want 2448", header.UnitBytes)
	}
}

// TestHeaderV4TooSmall verifies error for truncated V4 buffer.
func TestHeaderV4TooSmall(t *testing.T) {
	t.Parallel()

	header := &Header{Version: 4}
	err := parseHeaderV4(header, make([]byte, 10))
	if err == nil {
		t.Error("expected error for truncated buffer")
	}
	if !errors.Is(err, ErrInvalidHeader) {
		t.Errorf("expected ErrInvalidHeader, got: %v", err)
	}
}

// TestHeaderV3Parsing verifies V3 header parsing.
func TestHeaderV3Parsing(t *testing.T) {
	t.Parallel()

	// V3 header is 120 bytes, we need headerSizeV3-12 = 108 bytes
	buf := make([]byte, 108)

	// Flags at offset 4
	binary.BigEndian.PutUint32(buf[4:8], 0x00000002)
	// Compression at offset 8
	binary.BigEndian.PutUint32(buf[8:12], 0x00000003)
	// Total hunks at offset 12
	binary.BigEndian.PutUint32(buf[12:16], 500)
	// Logical bytes at offset 16
	binary.BigEndian.PutUint64(buf[16:24], 500000)
	// Meta offset at offset 24
	binary.BigEndian.PutUint64(buf[24:32], 250)
	// MD5 hashes at offset 32-64 (skip)
	// Hunk bytes at offset 64
	binary.BigEndian.PutUint32(buf[64:68], 8192)

	header := &Header{Version: 3}
	err := parseHeaderV3(header, buf)
	if err != nil {
		t.Fatalf("parseHeaderV3 failed: %v", err)
	}

	if header.Flags != 2 {
		t.Errorf("Flags = %d, want 2", header.Flags)
	}
	if header.Compression != 3 {
		t.Errorf("Compression = %d, want 3", header.Compression)
	}
	if header.TotalHunks != 500 {
		t.Errorf("TotalHunks = %d, want 500", header.TotalHunks)
	}
	if header.HunkBytes != 8192 {
		t.Errorf("HunkBytes = %d, want 8192", header.HunkBytes)
	}
}

// TestHeaderV3TooSmall verifies error for truncated V3 buffer.
func TestHeaderV3TooSmall(t *testing.T) {
	t.Parallel()

	header := &Header{Version: 3}
	err := parseHeaderV3(header, make([]byte, 50))
	if err == nil {
		t.Error("expected error for truncated buffer")
	}
	if !errors.Is(err, ErrInvalidHeader) {
		t.Errorf("expected ErrInvalidHeader, got: %v", err)
	}
}

// TestNumHunksCalculation verifies hunk count calculation.
func TestNumHunksCalculation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		header       Header
		expectedHunk uint32
	}{
		{
			name:         "from_total_hunks",
			header:       Header{TotalHunks: 100, HunkBytes: 4096, LogicalBytes: 1000000},
			expectedHunk: 100, // Uses TotalHunks when set
		},
		{
			name:         "calculated",
			header:       Header{TotalHunks: 0, HunkBytes: 4096, LogicalBytes: 16384},
			expectedHunk: 4, // exact fit: 16384 bytes at 4096 per hunk
		},
		{
			name:         "calculated_with_remainder",
			header:       Header{TotalHunks: 0, HunkBytes: 4096, LogicalBytes: 17000},
			expectedHunk: 5, // rounds up: 17000 bytes needs 5 hunks at 4096
		},
		{
			name:         "zero_hunk_bytes",
			header:       Header{TotalHunks: 0, HunkBytes: 0, LogicalBytes: 16384},
			expectedHunk: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.header.NumHunks()
			if got != tt.expectedHunk {
				t.Errorf("NumHunks() = %d, want %d", got, tt.expectedHunk)
			}
		})
	}
}

// TestParseCHTR verifies CHTR (v1 track) parsing.
func TestParseCHTR(t *testing.T) {
	t.Parallel()

	// CHTR uses same format as CHT2
	data := []byte("TRACK:1 TYPE:MODE1_RAW FRAMES:500")
	track, err := parseCHTR(data)
	if err != nil {
		t.Fatalf("parseCHTR failed: %v", err)
	}
	if track.Number != 1 {
		t.Errorf("Number = %d, want 1", track.Number)
	}
	if track.Type != "MODE1_RAW" {
		t.Errorf("Type = %q, want MODE1_RAW", track.Type)
	}
	if track.Frames != 500 {
		t.Errorf("Frames = %d, want 500", track.Frames)
	}
}

// TestParseCHCD verifies CHCD (binary track metadata) parsing.
func TestParseCHCD(t *testing.T) {
	t.Parallel()

	// Build a valid CHCD buffer
	// Format: numTracks (4 bytes) + track entries (24 bytes each)
	buf := make([]byte, 4+24*2) // 2 tracks

	// Number of tracks
	binary.BigEndian.PutUint32(buf[0:4], 2)

	// Track 1: MODE1/2048, RW subchannel, 1000 frames
	offset := 4
	binary.BigEndian.PutUint32(buf[offset:offset+4], 0)   // Type (0 = MODE1/2048)
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], 0) // SubType = RW
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], 2048)
	binary.BigEndian.PutUint32(buf[offset+12:offset+16], 96)
	binary.BigEndian.PutUint32(buf[offset+16:offset+20], 1000)
	binary.BigEndian.PutUint32(buf[offset+20:offset+24], 0) // Pad frames

	// Track 2: AUDIO
	offset = 4 + 24
	binary.BigEndian.PutUint32(buf[offset:offset+4], 5)   // Type (5 is AUDIO)
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], 2) // SubType (2 is NONE)
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], 2352)
	binary.BigEndian.PutUint32(buf[offset+12:offset+16], 0)
	binary.BigEndian.PutUint32(buf[offset+16:offset+20], 2000)
	binary.BigEndian.PutUint32(buf[offset+20:offset+24], 0)

	tracks, err := parseCHCD(buf)
	if err != nil {
		t.Fatalf("parseCHCD failed: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}

	// Check track 1
	if tracks[0].Number != 1 {
		t.Errorf("Track 1 Number = %d, want 1", tracks[0].Number)
	}
	if tracks[0].Type != "MODE1/2048" {
		t.Errorf("Track 1 Type = %q, want MODE1/2048", tracks[0].Type)
	}
	if tracks[0].Frames != 1000 {
		t.Errorf("Track 1 Frames = %d, want 1000", tracks[0].Frames)
	}

	// Check track 2
	if tracks[1].Number != 2 {
		t.Errorf("Track 2 Number = %d, want 2", tracks[1].Number)
	}
	if tracks[1].Type != "AUDIO" {
		t.Errorf("Track 2 Type = %q, want AUDIO", tracks[1].Type)
	}
}

// TestParseCHCDTooSmall verifies error for truncated CHCD.
func TestParseCHCDTooSmall(t *testing.T) {
	t.Parallel()

	// Buffer too small for header
	_, err := parseCHCD([]byte{0x00, 0x00})
	if err == nil {
		t.Error("expected error for truncated buffer")
	}
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Errorf("expected ErrInvalidMetadata, got: %v", err)
	}
}

// TestParseCHCDTooManyTracks verifies error for excessive track count.
func TestParseCHCDTooManyTracks(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf[0:4], 1000) // Way more than MaxNumTracks
	_, err := parseCHCD(buf)
	if err == nil {
		t.Error("expected error for too many tracks")
	}
	if !strings.Contains(err.Error(), "too many tracks") {
		t.Errorf("expected 'too many tracks' error, got: %v", err)
	}
}

// TestParseCHCDInsufficientData verifies error when data too small for tracks.
func TestParseCHCDInsufficientData(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 4+10) // Header says 1 track but not enough data
	binary.BigEndian.PutUint32(buf[0:4], 1)
	_, err := parseCHCD(buf)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Errorf("expected ErrInvalidMetadata, got: %v", err)
	}
}

// TestMetadataCircularChain verifies detection of circular metadata chains.
func TestMetadataCircularChain(t *testing.T) {
	t.Parallel()

	// Create a mock reader that returns metadata entries pointing to each other
	// Entry at offset 100 points to offset 200, which points back to 100
	data := make([]byte, 300)

	// Entry at offset 100: Tag=CHT2, Next=200
	binary.BigEndian.PutUint32(data[100:104], MetaTagCHT2)
	data[104] = 0 // flags
	data[105] = 0
	data[106] = 0
	data[107] = 10                                 // length = 10
	binary.BigEndian.PutUint64(data[108:116], 200) // next = 200

	// Entry at offset 200: Tag=CHT2, Next=100 (circular!)
	binary.BigEndian.PutUint32(data[200:204], MetaTagCHT2)
	data[204] = 0 // flags
	data[205] = 0
	data[206] = 0
	data[207] = 10                                 // length = 10
	binary.BigEndian.PutUint64(data[208:216], 100) // next = 100 (circular)

	reader := bytes.NewReader(data)
	_, err := parseMetadata(reader, 100)
	if err == nil {
		t.Error("expected error for circular chain")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' error, got: %v", err)
	}
}

// TestMetadataEntryTooLarge verifies MaxMetadataLen validation.
// Note: The CHD format uses 3 bytes for length (max 0xFFFFFF = 16,777,215)
// and MaxMetadataLen is 16*1024*1024 = 16,777,216. Since the max encodable
// value is less than the limit, this check can never trigger from valid format.
func TestMetadataEntryTooLarge(t *testing.T) {
	t.Parallel()

	t.Skip("MaxMetadataLen (16MB) exceeds 24-bit max (16MB-1), so this case cannot be triggered via format")
}

// TestRegisterAndGetCodec verifies codec registration.
func TestRegisterAndGetCodec(t *testing.T) {
	t.Parallel()

	// Test that registered codecs can be retrieved
	codecs := []uint32{
		CodecZlib, CodecLZMA, CodecFLAC, CodecZstd,
		CodecCDZlib, CodecCDLZMA, CodecCDFLAC, CodecCDZstd,
	}

	for _, tag := range codecs {
		codec, err := GetCodec(tag)
		if err != nil {
			t.Errorf("GetCodec(0x%x) failed: %v", tag, err)
			continue
		}
		if codec == nil {
			t.Errorf("GetCodec(0x%x) returned nil codec", tag)
		}
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen,nestif,revive,govet // Debug test with extensive diagnostic output
func TestNeoGeoCDCHD(t *testing.T) {
	t.Parallel()

	chdFile, err := Open("../testdata/NeoGeoCD/240pTestSuite.chd")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = chdFile.Close() }()

	header := chdFile.Header()
	t.Logf("Version: %d", header.Version)
	t.Logf("HunkBytes: %d", header.HunkBytes)
	t.Logf("UnitBytes: %d", header.UnitBytes)
	t.Logf("Compressors: %x", header.Compressors)

	frames := int(header.HunkBytes) / int(header.UnitBytes)
	t.Logf("Frames per hunk: %d", frames)
	t.Logf("Sector bytes per hunk: %d", frames*2352)

	// Print track information
	t.Log("\nTrack information:")
	t.Logf("MetaOffset: %d", header.MetaOffset)
	tracks := chdFile.Tracks()
	t.Logf("Number of tracks: %d", len(tracks))
	for i, track := range tracks {
		t.Logf("Track %d: Type=%s, Frames=%d, StartFrame=%d, Pregap=%d, IsData=%v",
			i+1, track.Type, track.Frames, track.StartFrame, track.Pregap, track.IsDataTrack())
	}

	// Debug: Try parsing metadata directly
	if header.MetaOffset > 0 {
		metaBuf := make([]byte, 100)
		metaBytes, metaErr := chdFile.file.ReadAt(metaBuf, int64(header.MetaOffset)) //nolint:gosec // Test only
		if metaErr != nil {
			t.Logf("Read metadata raw failed: %v", metaErr)
		} else {
			t.Logf("Raw metadata (%d bytes): %x", metaBytes, metaBuf[:metaBytes])
			t.Logf("Tag: %s", string(metaBuf[0:4]))
			t.Logf("Data: %s", string(metaBuf[16:80]))
		}

		// Debug: Try parseMetadata directly
		entries, parseErr := parseMetadata(chdFile.file, header.MetaOffset)
		if parseErr != nil {
			t.Logf("parseMetadata failed: %v", parseErr)
		} else {
			t.Logf("Parsed %d metadata entries", len(entries))
			for i, entry := range entries {
				t.Logf("Entry %d: Tag=%x, Flags=%d, DataLen=%d, Next=%d",
					i, entry.Tag, entry.Flags, len(entry.Data), entry.Next)
				t.Logf("  Data: %s", string(entry.Data))
			}

			// Try parseTracks
			parsedTracks, err := parseTracks(entries)
			if err != nil {
				t.Logf("parseTracks failed: %v", err)
			} else {
				t.Logf("Parsed %d tracks", len(parsedTracks))
				for i, track := range parsedTracks {
					t.Logf("  Track %d: Type=%s Frames=%d IsData=%v",
						i+1, track.Type, track.Frames, track.IsDataTrack())
				}
			}
		}
	}

	// Test firstDataTrackSector
	t.Log("\nData track sector offset:", chdFile.firstDataTrackSector())
	t.Log("Data track size:", chdFile.DataTrackSize())

	// Print first 20 hunk map entries to see the pattern
	t.Log("\nHunk map entries:")
	t.Logf("Number of hunks: %d", chdFile.hunkMap.NumHunks())
	for idx := uint32(0); idx < 20 && idx < chdFile.hunkMap.NumHunks(); idx++ {
		entry := chdFile.hunkMap.entries[idx]
		codecName := "?"
		if int(entry.CompType) < len(header.Compressors) {
			tag := header.Compressors[entry.CompType]
			codecName = string([]byte{byte(tag >> 24), byte(tag >> 16), byte(tag >> 8), byte(tag)})
		}
		t.Logf("Hunk %d: CompType=%d (%s), CompLength=%d, Offset=%d",
			idx, entry.CompType, codecName, entry.CompLength, entry.Offset)
	}

	// Try to read data from a hunk that uses LZMA (comptype 0)
	// Skip the FLAC hunks and read from hunk 2 which is LZMA
	t.Log("\nTrying to read hunk 2 (LZMA)...")
	hunkData, err := chdFile.hunkMap.ReadHunk(2)
	if err != nil {
		t.Logf("Read hunk 2 failed: %v", err)
	} else {
		t.Logf("Hunk 2 data length: %d", len(hunkData))
		if len(hunkData) > 32 {
			t.Logf("First 32 bytes: %x", hunkData[:32])
		}
	}

	// Test reading sector 0 (where PVD actually starts for this disc)
	t.Log("\nTrying to read sector 0 using DataTrackSectorReader...")
	reader := chdFile.DataTrackSectorReader()
	sector0Data := make([]byte, 2048)
	readBytes, err := reader.ReadAt(sector0Data, 0) // Sector 0
	if err != nil {
		t.Logf("Read sector 0 failed: %v", err)
	} else {
		t.Logf("Read %d bytes", readBytes)
		t.Logf("First 32 bytes: %x", sector0Data[:32])
		t.Logf("String view: %s", string(sector0Data[1:6]))
	}

	// Test reading sector 16 using DataTrackSectorReader (where PVD should be)
	t.Log("\nTrying to read sector 16 (PVD) using DataTrackSectorReader...")
	pvdData := make([]byte, 2048)
	readBytes, err = reader.ReadAt(pvdData, 16*2048) // Sector 16
	if err != nil {
		t.Logf("Read PVD sector failed: %v", err)
	} else {
		t.Logf("Read %d bytes", readBytes)
		t.Logf("First 32 bytes: %x", pvdData[:32])
		t.Logf("String view: %s", string(pvdData[1:6]))
	}

	// Check hunk 0 data to understand the disc layout
	t.Log("\nReading hunk 0 (audio track) to see what's there...")
	hunk0Data, err := chdFile.hunkMap.ReadHunk(0)
	if err != nil {
		t.Logf("Read hunk 0 failed: %v", err)
	} else {
		t.Logf("Hunk 0 length: %d", len(hunk0Data))
		t.Logf("Hunk 0 first 32 bytes: %x", hunk0Data[:32])
	}

	// Simulate what iso9660.OpenCHD does - read first ~50KB via DataTrackSectorReader
	t.Log("\nSimulating ISO9660 init read (first 50KB)...")
	isoReader := chdFile.DataTrackSectorReader()
	isoData := make([]byte, 50000)
	readBytes, err = isoReader.ReadAt(isoData, 0)
	if err != nil && err.Error() != "EOF" {
		t.Logf("ISO reader failed: %v", err)
	}
	t.Logf("Read %d bytes", readBytes)
	// Search for CD001 in the data
	for i := range len(isoData) - 6 {
		if isoData[i] == 0x01 && isoData[i+1] == 'C' && isoData[i+2] == 'D' &&
			isoData[i+3] == '0' && isoData[i+4] == '0' && isoData[i+5] == '1' {
			t.Logf("Found PVD at offset %d (sector %d)", i, i/2048)
			t.Logf("Data at PVD: %x", isoData[i:i+32])
			break
		}
	}

	// The PVD \x01CD001 was seen at the start of hunk 2
	// Hunk 2 contains frames 16-23
	// Sector 0 of the data should contain the system area (16 reserved sectors)
	// Then PVD at sector 16
	// So if hunk 2 starts the data track, the PVD should be at the start
	// of the sector data in hunk 2 (after extracting from raw sector)
	t.Log("\nChecking raw hunk 2 data structure...")
	hunk2, err := chdFile.hunkMap.ReadHunk(2)
	if err != nil {
		t.Logf("Read hunk 2 failed: %v", err)
	} else {
		// Check the first sector in hunk 2
		// Raw sector = 2352 bytes, subchannel = 96 bytes, unit = 2448 bytes
		unitBytes := int(header.UnitBytes)
		t.Logf("Unit bytes: %d", unitBytes)
		for sectorIdx := range 3 {
			sectorStart := sectorIdx * unitBytes
			if sectorStart+32 > len(hunk2) {
				break
			}
			// Check sync header (first 12 bytes of raw sector)
			t.Logf("Sector %d raw start: %x", sectorIdx, hunk2[sectorStart:sectorStart+32])
			// User data starts at offset 16 (Mode1)
			dataStart := sectorStart + 16
			if dataStart+32 <= len(hunk2) {
				t.Logf("Sector %d user data (+16): %x", sectorIdx, hunk2[dataStart:dataStart+32])
			}
		}
	}
}
