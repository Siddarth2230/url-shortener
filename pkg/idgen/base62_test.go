package idgen

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{62, "10"},
		{12345, "3D7"},
		{916132831, "zzzzz"}, // 62^5 - 1
	}

	for _, tt := range tests {
		result := Encode(tt.input)
		if result != tt.expected {
			t.Errorf("Encode(%d) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{62, "10"},
		{12345, "3D7"},
		{916132831, "zzzzz"}, // 62^5 - 1
	}

	for _, tt := range tests {
		result := Decode(tt.expected)
		if result != tt.input {
			t.Errorf("Decode(%s) = %d; want %d", tt.expected, result, tt.input)
		}
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	// TODO: Test that Decode(Encode(n)) == n for random numbers
	for i := uint64(0); i < 100000; i += 1234 {
		encoded := Encode(i)
		decoded := Decode(encoded)
		if decoded != i {
			t.Errorf("Decode(Encode(%d)) = %d; want %d", i, decoded, i)
		}
	}
}
