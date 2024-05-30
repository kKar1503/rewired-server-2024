package utils

import (
	"testing"
)

func TestSplitByteInto2FourBits(t *testing.T) {
	tests := []struct {
		input     byte
		expected1 uint8
		expected2 uint8
	}{
		{0xAB, 0xA, 0xB},
		{0xFF, 0xF, 0xF},
		{0x00, 0x0, 0x0},
		{0x10, 0x1, 0x0},
		{0x01, 0x0, 0x1},
		{0x4C, 0x4, 0xC},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			b1, b2 := SplitByteInto2FourBits(tt.input)
			if b1 != tt.expected1 || b2 != tt.expected2 {
				t.Errorf(
					"SplitByteInto2FourBits(%X) = (%X, %X), expected (%X, %X)",
					tt.input,
					b1,
					b2,
					tt.expected1,
					tt.expected2,
				)
			}
		})
	}
}

func TestJoin2FourBitsIntoByte(t *testing.T) {
	tests := []struct {
		input1   uint8
		input2   uint8
		expected byte
	}{
		{0xA, 0xB, 0xAB},
		{0xF, 0xF, 0xFF},
		{0x0, 0x0, 0x00},
		{0x1, 0x0, 0x10},
		{0x0, 0x1, 0x01},
		{0x4, 0xC, 0x4C},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := Join2FourBitsIntoByte(tt.input1, tt.input2)
			if result != tt.expected {
				t.Errorf("Join2FourBitsIntoByte(%X, %X) = %X, expected %X", tt.input1, tt.input2, result, tt.expected)
			}
		})
	}
}

func TestSplitAndJoin(t *testing.T) {
	tests := []byte{0xAB, 0xFF, 0x00, 0x10, 0x01, 0x4C}

	for _, input := range tests {
		t.Run("", func(t *testing.T) {
			b1, b2 := SplitByteInto2FourBits(input)
			result := Join2FourBitsIntoByte(b1, b2)
			if result != input {
				t.Errorf("Join2FourBitsIntoByte(SplitByteInto2FourBits(%X)) = %X, expected %X", input, result, input)
			}
		})
	}
}
