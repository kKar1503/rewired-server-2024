package utils

func SplitByteInto2FourBits(b byte) (uint8, uint8) {
	b1 := (b >> 4) & 0x0F
	b2 := b & 0x0F
	return b1, b2
}

func Join2FourBitsIntoByte(i, j uint8) byte {
	return (i << 4) | j
}
