package message

import (
	"encoding/binary"
	"fmt"
)

// Have creates a Have message.
func Have(index int) *Message {
	var payload [4]byte
	binary.BigEndian.PutUint32(payload[:], uint32(index))
	return &Message{ID: MsgHave, Payload: payload[:]}
}

// Request creates a Request message.
func Request(index, begin, length int) *Message {
	var payload [12]byte
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload[:]}
}

// Piece creates a Piece message.
func Piece(index, begin int, data []byte) *Message {
	// TODO:
	return nil
}

// ParseHave converts a Have message to the index from the payload.
func ParseHave(msg *Message) (int, error) {
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("expected a Have message (ID %d), got ID %d", MsgHave, msg.ID)
	}
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload with length 4, got length %d", len(msg.Payload))
	}
	return int(binary.BigEndian.Uint32(msg.Payload)), nil
}

// ParseRequest converts a Request message to the index, begin, length from the payload.
func ParseRequest(msg *Message) (int, int, int, error) {
	// TODO:
	return 0, 0, 0, nil
}

// ParsePiece converts a Piece message to index, begin, data and writes data to buf.
func ParsePiece(msg *Message, index int, buf []byte) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("expected a Piece message (ID %d), got ID %d", MsgPiece, msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("payload too short, expected 8+ bytes, got %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("expected index %d, got %d", index, parsedIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("offset begin too high")
	}

	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("not enough space in buf (%d) to write data (%d) from offset begin (%d)", len(buf), len(data), begin)
	}

	return copy(buf[begin:], data), nil
}
