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

package iso9660

import (
	"fmt"

	"github.com/ZaparooProject/go-gameid/chd"
)

// OpenCHD opens an ISO9660 filesystem from a CHD disc image file.
// The CHD file's DataTrackSectorReader provides 2048-byte logical sectors
// starting at the first data track, suitable for ISO9660 parsing.
// This handles multi-track CDs like Neo Geo CD that have audio tracks first.
func OpenCHD(path string) (*ISO9660, error) {
	chdFile, err := chd.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open CHD: %w", err)
	}

	// Use the data track sector reader which provides 2048-byte logical sectors
	// starting at the first data track (essential for multi-track CDs)
	reader := chdFile.DataTrackSectorReader()
	size := chdFile.DataTrackSize()

	// Create ISO9660 with the CHD as the underlying closer
	iso, err := OpenReaderWithCloser(reader, size, chdFile)
	if err != nil {
		_ = chdFile.Close()
		return nil, fmt.Errorf("parse ISO9660 from CHD: %w", err)
	}

	return iso, nil
}
