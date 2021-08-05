package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

type messageID uint8

const (
	// MsgChoke chokes the receiver.
	MsgChoke messageID = iota
	// MsgUnchoke unchokes the receiver.
	MsgUnchoke
	// MsgInterested expresses interest in receiving data.
	MsgInterested
	// MsgNotInterested expresses disinterest in receiving data.
	MsgNotInterested
	// MsgHave alerts the receiver that the sender has downloaded a piece.
	MsgHave
	// MsgBitfield encodes which pieces the sender has downloaded.
	MsgBitfield
	// MsgRequest requests a block of data from the receiver.
	MsgRequest
	// MsgPiece delivers a block of data to fulfill a request.
	MsgPiece
	// MsgCancel cancels a request.
	MsgCancel
)

// Message stores ID and payload of a message.
type Message struct {
	ID      messageID
	Payload []byte
}

// Marshal serializes a message to bytes of the form:
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message.
func Marshal(m *Message) (ret []byte) {
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(1 + len(m.Payload))
	ret = make([]byte, 4+length)
	binary.BigEndian.PutUint32(ret[0:4], length)
	ret[4] = byte(m.ID)
	copy(ret[5:], m.Payload)
	return
}

// Unmarshal parses a message from a stream.
// Returns `nil` on keep-alive message.
func Unmarshal(r io.Reader) (*Message, error) {
	var length uint32
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	msgBuf := make([]byte, length)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      messageID(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil

}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	switch m.ID {
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgNotInterested:
		return "NotInterested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgRequest:
		return "Request"
	case MsgPiece:
		return "Piece"
	case MsgCancel:
		return "Cancel"
	default:
		return fmt.Sprintf("Unknown#%d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}

	return fmt.Sprintf("%s (payload len. %d)", m.name(), len(m.Payload))
}
