package binary

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadUint16BE(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  uint16
		wantErr bool
	}{
		{"Valid data", []byte{0x12, 0x34}, 0x1234, false},
		{"All zeros", []byte{0x00, 0x00}, 0x0000, false},
		{"All ones", []byte{0xFF, 0xFF}, 0xFFFF, false},
		{"Too short", []byte{0x12}, 0, true},
		{"Empty", []byte{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			got, err := ReadUint16BE(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint16BE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadUint16BE() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint16LE(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  uint16
		wantErr bool
	}{
		{"Valid data", []byte{0x34, 0x12}, 0x1234, false},
		{"All zeros", []byte{0x00, 0x00}, 0x0000, false},
		{"All ones", []byte{0xFF, 0xFF}, 0xFFFF, false},
		{"Too short", []byte{0x12}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			got, err := ReadUint16LE(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint16LE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadUint16LE() = 0x%04X, want 0x%04X", got, tt.want)
			}
		})
	}
}

func TestReadUint32BE(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  uint32
		wantErr bool
	}{
		{"Valid data", []byte{0x12, 0x34, 0x56, 0x78}, 0x12345678, false},
		{"All zeros", []byte{0x00, 0x00, 0x00, 0x00}, 0x00000000, false},
		{"All ones", []byte{0xFF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFFF, false},
		{"Too short", []byte{0x12, 0x34}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			got, err := ReadUint32BE(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUint32BE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadUint32BE() = 0x%08X, want 0x%08X", got, tt.want)
			}
		})
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		offset    int
		length    int
		want      string
		wantClean string // after cleaning non-printable chars
	}{
		{
			name:      "Valid ASCII string",
			data:      []byte("Hello World!"),
			offset:    0,
			length:    12,
			want:      "Hello World!",
			wantClean: "Hello World!",
		},
		{
			name:      "String with null termination",
			data:      []byte("Hello\x00World"),
			offset:    0,
			length:    11,
			want:      "Hello\x00World",
			wantClean: "Hello",
		},
		{
			name:      "String with non-printable chars",
			data:      []byte("Hello\x01\x02World!"),
			offset:    0,
			length:    13,
			want:      "Hello\x01\x02World!",
			wantClean: "Hello  World!",
		},
		{
			name:      "Extract from offset",
			data:      []byte("PrefixHello"),
			offset:    6,
			length:    5,
			want:      "Hello",
			wantClean: "Hello",
		},
		{
			name:      "Bounds check",
			data:      []byte("Short"),
			offset:    0,
			length:    10,
			want:      "",
			wantClean: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractString(tt.data, tt.offset, tt.length)
			if got != tt.want {
				t.Errorf("ExtractString() = %q, want %q", got, tt.want)
			}

			// Test clean string function
			clean := CleanString([]byte(tt.want))
			if clean != tt.wantClean {
				t.Errorf("CleanString() = %q, want %q", clean, tt.wantClean)
			}
		})
	}
}

func TestCalculateChecksum(t *testing.T) {
	// Test simple checksum calculation
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	
	// Test 8-bit checksum
	checksum8 := CalculateChecksum8(data)
	expected8 := uint8(0x01 + 0x02 + 0x03 + 0x04 + 0x05)
	if checksum8 != expected8 {
		t.Errorf("CalculateChecksum8() = 0x%02X, want 0x%02X", checksum8, expected8)
	}

	// Test 16-bit checksum
	checksum16 := CalculateChecksum16(data)
	expected16 := uint16(0x01 + 0x02 + 0x03 + 0x04 + 0x05)
	if checksum16 != expected16 {
		t.Errorf("CalculateChecksum16() = 0x%04X, want 0x%04X", checksum16, expected16)
	}
}

func TestN64EndianSwap(t *testing.T) {
	// Test N64 endianness conversion (swap every 2 bytes)
	tests := []struct {
		name string
		data []byte
		want []byte
	}{
		{
			name: "4 bytes",
			data: []byte{0x12, 0x34, 0x56, 0x78},
			want: []byte{0x34, 0x12, 0x78, 0x56},
		},
		{
			name: "Empty",
			data: []byte{},
			want: []byte{},
		},
		{
			name: "2 bytes",
			data: []byte{0xAB, 0xCD},
			want: []byte{0xCD, 0xAB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := N64EndianSwap(tt.data)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("N64EndianSwap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestN64EndianSwap_OddLength(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("N64EndianSwap() did not panic on odd length data")
		}
	}()

	// Should panic on odd length
	N64EndianSwap([]byte{0x01, 0x02, 0x03})
}

// Test binary writing for creating test fixtures
func TestWriteBinary(t *testing.T) {
	var buf bytes.Buffer

	// Write different types
	binary.Write(&buf, binary.BigEndian, uint16(0x1234))
	binary.Write(&buf, binary.LittleEndian, uint32(0x567890AB))

	expected := []byte{0x12, 0x34, 0xAB, 0x90, 0x78, 0x56}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Binary write produced %v, want %v", buf.Bytes(), expected)
	}
}