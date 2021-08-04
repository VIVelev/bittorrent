package handshake

import (
	"fmt"
	"io"
)

const (
	Len  int    = 0x13
	Pstr string = "BitTorrent protocol"
)

type Handshake struct {
	ReservedBytes [8]byte
	InfoHash      [20]byte
	PeerID        [20]byte
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
	var l [1]byte
	_, err := io.ReadFull(r, l[:])
	if err != nil {
		return nil, err
	}
	if int(l[0]) != Len {
		return nil, fmt.Errorf("invalid Pstr Len: %d, should be: %d", l[0], Len)
	}

	var pstr [Len]byte
	_, err = io.ReadFull(r, pstr[:])
	if err != nil {
		return nil, err
	}
	if string(pstr[:]) != Pstr {
		return nil, fmt.Errorf("invalid Pstr: %s, should be: %s", pstr, Pstr)
	}

	hs := new(Handshake)
	_, err = io.ReadFull(r, hs.ReservedBytes[:])
	if err != nil {
		return nil, err
	}
	_, err = io.ReadFull(r, hs.InfoHash[:])
	if err != nil {
		return nil, err
	}
	_, err = io.ReadFull(r, hs.PeerID[:])
	if err != nil {
		return nil, err
	}

	return hs, nil
}
