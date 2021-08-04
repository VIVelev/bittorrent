package handshake

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshal(t *testing.T) {
	tests := map[string]struct {
		input  *Handshake
		output [68]byte
	}{
		"marshal": {
			input: &Handshake{
				InfoHash: [20]byte{134, 212, 200, 0, 36, 164, 105, 190, 76, 80, 188, 90, 16, 44, 247, 23, 128, 49, 0, 116},
				PeerID:   [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			},
			output: [68]byte{19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114, 111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 0, 0, 0, 134, 212, 200, 0, 36, 164, 105, 190, 76, 80, 188, 90, 16, 44, 247, 23, 128, 49, 0, 116, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
	}

	for _, test := range tests {
		assert.Equal(t, Marshal(test.input), test.output)
	}
}

func TestUnmarshal(t *testing.T) {
	tests := map[string]struct {
		input  []byte
		output *Handshake
		fails  bool
	}{
		"unmarshal": {
			input: []byte{19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114, 111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 0, 0, 0, 134, 212, 200, 0, 36, 164, 105, 190, 76, 80, 188, 90, 16, 44, 247, 23, 128, 49, 0, 116, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			output: &Handshake{
				InfoHash: [20]byte{134, 212, 200, 0, 36, 164, 105, 190, 76, 80, 188, 90, 16, 44, 247, 23, 128, 49, 0, 116},
				PeerID:   [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			},
			fails: false,
		},
		"empty": {
			input:  []byte{},
			output: nil,
			fails:  true,
		},
		"not enough bytes": {
			input:  []byte{19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114, 111, 116, 111, 99, 111},
			output: nil,
			fails:  true,
		},
		"pstrlen is 0": {
			input:  []byte{0, 0, 0},
			output: nil,
			fails:  true,
		},
	}

	for _, test := range tests {
		r := bytes.NewReader(test.input)
		hs, err := Unmarshal(r)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, hs, test.output)
	}
}
