package binary

import (
	"bytes"
	"testing"
)

func TestReadBytesAt(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	r := bytes.NewReader(data)

	tests := []struct {
		name    string
		offset  int64
		length  int
		want    []byte
		wantErr bool
	}{
		{"read from start", 0, 3, []byte{0x00, 0x01, 0x02}, false},
		{"read from middle", 2, 3, []byte{0x02, 0x03, 0x04}, false},
		{"read to end", 3, 3, []byte{0x03, 0x04, 0x05}, false},
		{"read past end", 4, 5, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadBytesAt(r, tt.offset, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadBytesAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("ReadBytesAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadUint16LEAt(t *testing.T) {
	data := []byte{0x34, 0x12, 0x78, 0x56}
	r := bytes.NewReader(data)

	tests := []struct {
		name   string
		offset int64
		want   uint16
	}{
		{"first value", 0, 0x1234},
		{"second value", 2, 0x5678},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadUint16LEAt(r, tt.offset)
			if err != nil {
				t.Errorf("ReadUint16LEAt() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ReadUint16LEAt() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint16BEAt(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	r := bytes.NewReader(data)

	tests := []struct {
		name   string
		offset int64
		want   uint16
	}{
		{"first value", 0, 0x1234},
		{"second value", 2, 0x5678},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadUint16BEAt(r, tt.offset)
			if err != nil {
				t.Errorf("ReadUint16BEAt() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ReadUint16BEAt() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint32LEAt(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12}
	r := bytes.NewReader(data)

	got, err := ReadUint32LEAt(r, 0)
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
	data := []byte{0x12, 0x34, 0x56, 0x78}
	r := bytes.NewReader(data)

	got, err := ReadUint32BEAt(r, 0)
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
	data := []byte("Hello, World!\x00More text")
	r := bytes.NewReader(data)

	tests := []struct {
		name   string
		offset int64
		length int
		want   string
	}{
		{"full string", 0, 13, "Hello, World!"},
		{"substring", 0, 5, "Hello"},
		{"from middle", 7, 6, "World!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadStringAt(r, tt.offset, tt.length)
			if err != nil {
				t.Errorf("ReadStringAt() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ReadStringAt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"normal string", []byte("Hello"), "Hello"},
		{"with null terminator", []byte("Hello\x00World"), "Hello"},
		{"padded with nulls", []byte("Test\x00\x00\x00"), "Test"},
		{"with trailing spaces", []byte("Test   "), "Test"},
		{"with leading spaces", []byte("   Test"), "Test"},
		{"with both", []byte("  Test  \x00"), "Test"},
		{"empty", []byte{}, ""},
		{"only nulls", []byte{0, 0, 0}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanString(tt.input)
			if got != tt.want {
				t.Errorf("CleanString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPrintable(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"normal ASCII", []byte("Hello"), "Hello"},
		{"with control chars", []byte("Hello\x01World"), "HelloWorld"},
		{"with high bytes", []byte("Test\x80\x90"), "Test"},
		{"spaces preserved", []byte("Hello World"), "Hello World"},
		{"numbers and symbols", []byte("Test123!@#"), "Test123!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPrintable(tt.input)
			if got != tt.want {
				t.Errorf("ExtractPrintable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBytesEqual(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("BytesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindBytes(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindBytes(data, tt.needle)
			if got != tt.want {
				t.Errorf("FindBytes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindBytesInRange(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x01, 0x02}
	r := bytes.NewReader(data)

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindBytesInRange(r, tt.start, tt.end, tt.needle)
			if err != nil {
				t.Errorf("FindBytesInRange() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("FindBytesInRange() = %d, want %d", got, tt.want)
			}
		})
	}
}
