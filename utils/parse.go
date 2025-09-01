package utils

import "math/big"

// bytes must be 32 bytes (EVM word)
func ParseInt256(b []byte) *big.Int {
	val := new(big.Int).SetBytes(b)

	// If MSB (sign bit) is set, it's negative
	if b[0]&0x80 != 0 {
		// subtract 2^256
		two256 := new(big.Int).Lsh(big.NewInt(1), 256)
		val.Sub(val, two256)
	}
	return val
}
