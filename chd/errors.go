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

import "errors"

// Allocation limits to prevent DoS from malicious CHD files.
const (
	// MaxCompMapLen is the maximum compressed map size (100MB).
	MaxCompMapLen = 100 * 1024 * 1024

	// MaxNumHunks is the maximum number of hunks (10M = ~200GB uncompressed).
	MaxNumHunks = 10_000_000

	// MaxMetadataLen is the maximum metadata entry size (16MB, matches 24-bit limit).
	MaxMetadataLen = 16 * 1024 * 1024

	// MaxNumTracks is the maximum number of tracks (200, generous for any disc).
	MaxNumTracks = 200

	// MaxMetadataEntries is the maximum metadata chain entries (prevents loops).
	MaxMetadataEntries = 1000
)

// Common errors for CHD parsing.
var (
	// ErrInvalidMagic indicates the file does not have a valid CHD magic word.
	ErrInvalidMagic = errors.New("invalid CHD magic: expected MComprHD")

	// ErrInvalidHeader indicates the header structure is invalid.
	ErrInvalidHeader = errors.New("invalid CHD header")

	// ErrUnsupportedVersion indicates an unsupported CHD version.
	ErrUnsupportedVersion = errors.New("unsupported CHD version")

	// ErrUnsupportedCodec indicates an unsupported compression codec.
	ErrUnsupportedCodec = errors.New("unsupported compression codec")

	// ErrInvalidHunk indicates an invalid hunk index.
	ErrInvalidHunk = errors.New("invalid hunk index")

	// ErrDecompressFailed indicates decompression failed.
	ErrDecompressFailed = errors.New("decompression failed")

	// ErrCorruptData indicates data corruption was detected.
	ErrCorruptData = errors.New("data corruption detected")

	// ErrNoTracks indicates no track metadata was found.
	ErrNoTracks = errors.New("no track metadata found")

	// ErrInvalidMetadata indicates invalid metadata format.
	ErrInvalidMetadata = errors.New("invalid metadata format")
)
