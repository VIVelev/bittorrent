package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHave(t *testing.T) {
	msg := Have(4)
	expected := &Message{
		ID:      MsgHave,
		Payload: []byte{0, 0, 0, 4},
	}

	assert.Equal(t, msg, expected)
}

func TestRequest(t *testing.T) {
	msg := Request(4, 567, 4321)
	expected := &Message{
		ID: MsgRequest,
		Payload: []byte{
			0x00, 0x00, 0x00, 0x04, // index
			0x00, 0x00, 0x02, 0x37, // begin
			0x00, 0x00, 0x10, 0xe1, // length
		},
	}

	assert.Equal(t, msg, expected)
}

func TestParseHave(t *testing.T) {
	tests := map[string]struct {
		input  *Message
		output uint32
		fails  bool
	}{
		"parse valid message": {
			input:  &Message{ID: MsgHave, Payload: []byte{0x00, 0x00, 0x00, 0x04}},
			output: 4,
			fails:  false,
		},
		"wrong message type": {
			input:  &Message{ID: MsgPiece, Payload: []byte{0x00, 0x00, 0x00, 0x04}},
			output: 0,
			fails:  true,
		},
		"payload too short": {
			input:  &Message{ID: MsgHave, Payload: []byte{0x00, 0x00, 0x04}},
			output: 0,
			fails:  true,
		},
		"payload too long": {
			input:  &Message{ID: MsgHave, Payload: []byte{0x00, 0x00, 0x00, 0x00, 0x04}},
			output: 0,
			fails:  true,
		},
	}

	for _, test := range tests {
		index, err := ParseHave(test.input)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, index, test.output)
	}

}

func TestParsePiece(t *testing.T) {
	tests := map[string]struct {
		msg       *Message
		index     uint32
		buf       []byte
		outputN   int
		targetBuf []byte
		fails     bool
	}{
		"parse valid piece": {
			msg: &Message{
				ID: MsgPiece,
				Payload: []byte{
					0x00, 0x00, 0x00, 0x04, // index
					0x00, 0x00, 0x00, 0x02, // begin
					0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // data
				},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   6,
			targetBuf: []byte{0x00, 0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x00},
			fails:     false,
		},
		"wrong message type": {
			msg: &Message{
				ID:      MsgChoke,
				Payload: []byte{},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   0,
			targetBuf: make([]byte, 10),
			fails:     true,
		},
		"payload too short": {
			msg: &Message{
				ID: MsgPiece,
				Payload: []byte{
					0x00, 0x00, 0x00, 0x04, // index
					0x00, 0x00, 0x00, // malformed offset
				},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   0,
			targetBuf: make([]byte, 10),
			fails:     true,
		},
		"wrong index": {
			msg: &Message{
				ID: MsgPiece,
				Payload: []byte{
					0x00, 0x00, 0x00, 0x06, // index is 6, not 4
					0x00, 0x00, 0x00, 0x02, // begin
					0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // data
				},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   0,
			targetBuf: make([]byte, 10),
			fails:     true,
		},
		"offset too high": {
			msg: &Message{
				ID: MsgPiece,
				Payload: []byte{
					0x00, 0x00, 0x00, 0x04, // index
					0x00, 0x00, 0x00, 0x0c, // begin is 12 > 10
					0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // data
				},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   0,
			targetBuf: make([]byte, 10),
			fails:     true,
		},
		"offset ok but data too long": {
			msg: &Message{
				ID: MsgPiece,
				Payload: []byte{
					0x00, 0x00, 0x00, 0x04, // index
					0x00, 0x00, 0x00, 0x02, // begin is ok
					// data is 10 bytes but begin=2; too long for buf
					0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x0a, 0x0b, 0x0c, 0x0d,
				},
			},
			index:     4,
			buf:       make([]byte, 10),
			outputN:   0,
			targetBuf: make([]byte, 10),
			fails:     true,
		},
	}

	for _, test := range tests {
		n, err := ParsePiece(test.msg, test.index, test.buf)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, n, test.outputN)
		assert.Equal(t, test.buf, test.targetBuf)
	}
}
