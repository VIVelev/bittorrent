package client

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/VIVelev/bittorrent/bitfield"
	"github.com/VIVelev/bittorrent/handshake"
	"github.com/VIVelev/bittorrent/message"
	"github.com/VIVelev/bittorrent/peer"
)

// Client is a TCP connection with one peer.
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield bitfield.Bitfield
}

func completeHandshake(conn net.Conn, infoHash, peerID [20]byte) (*handshake.Handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // disable the deadline

	hs := &handshake.Handshake{
		InfoHash: infoHash,
		PeerID:   peerID,
	}
	req := handshake.Marshal(hs)
	_, err := conn.Write(req[:])
	if err != nil {
		return nil, err
	}

	res, err := handshake.Unmarshal(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("expected InfoHash: %x, got: %x", infoHash, res.InfoHash)
	}

	return res, nil
}

func recvBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // disable the deadline

	msg, err := message.Unmarshal(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, fmt.Errorf("expected bitfield message, but got %s", msg)
	}
	if msg.ID != message.MsgBitfield {
		return nil, fmt.Errorf("expected bitfield message, but got %s", msg)
	}

	return msg.Payload, nil
}

// New connects to a peer, completes a handshake, and receives a bitfield.
func New(peer peer.Peer, infoHash, peerID [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		return nil, err
	}

	bf, err := recvBitfield(conn)
	if err != nil {
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
	}, nil
}

// Read unmarshals a message from the connection.
func (c *Client) Read() (*message.Message, error) {
	return message.Unmarshal(c.Conn)
}

// WriteUnchoke sends an UnchokeMsg to the peer.
func (c *Client) WriteUnchoke() error {
	m := &message.Message{ID: message.MsgUnchoke}
	_, err := c.Conn.Write(message.Marshal(m))
	return err
}

func (c *Client) WriteInterested() error {
	m := &message.Message{ID: message.MsgInterested}
	_, err := c.Conn.Write(message.Marshal(m))
	return err
}

func (c *Client) WriteNotInterested() error {
	m := &message.Message{ID: message.MsgNotInterested}
	_, err := c.Conn.Write(message.Marshal(m))
	return err
}

func (c *Client) WriteHave(index uint32) error {
	m := message.Have(index)
	_, err := c.Conn.Write(message.Marshal(m))
	return err
}

func (c *Client) WriteRequest(index, begin, length uint32) error {
	m := message.Request(index, begin, length)
	_, err := c.Conn.Write(message.Marshal(m))
	return err
}
