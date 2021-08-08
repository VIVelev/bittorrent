package bitfield

// Bitfield represents the pieces that a peer has.
type Bitfield []byte

// HasPiece checks if the bitfield has a particular index set.
func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	bitIndex := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}
	return bf[byteIndex]<<byte(bitIndex)&128 != 0
}

// SetPiece sets the bit at index in the bitfield.
func (bf Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	bitIndex := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] |= 128 >> bitIndex
}
