package handshake

import (
	"fmt"
	"io"
)

const (
	Len  int    = 0x13
	Pstr string = "BitTorrent protocol"
)

// Handshake is the first message two peers exchange to establish a connetion.
// A handshake consists of:
// <length of protocol id><protocol id><ReservedBytes><InfoHash><PeerID>
type Handshake struct {
	ReservedBytes [8]byte  // used to indicate support for certain extensions
	InfoHash      [20]byte // identifies which file we want
	PeerID        [20]byte // idetifies ourselves
}

func Marshal(hs *Handshake) (ret [1 + Len + 48]byte) {
	ret[0] = byte(Len)
	curr := 1
	curr += copy(ret[curr:], []byte(Pstr))
	curr += copy(ret[curr:], hs.ReservedBytes[:])
	curr += copy(ret[curr:], hs.InfoHash[:])
	curr += copy(ret[curr:], hs.PeerID[:])
	return
}

func Unmarshal(r io.Reader) (*Handshake, error) {
	var buf [1 + Len + 8 + 20 + 20]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, fmt.Errorf("read: %s", err)
	}

	if int(buf[0]) != Len {
		return nil, fmt.Errorf("invalid Pstr Len: %d, should be: %d", buf[0], Len)
	}
	if string(buf[1:1+Len]) != Pstr {
		return nil, fmt.Errorf("invalid Pstr: %s, should be: %s", buf[1:1+Len], Pstr)
	}

	hs := new(Handshake)
	copy(hs.ReservedBytes[:], buf[1+Len:1+Len+8])
	copy(hs.InfoHash[:], buf[1+Len+8:1+Len+8+20])
	copy(hs.PeerID[:], buf[1+Len+8+20:1+Len+8+20+20])
	return hs, nil
}
