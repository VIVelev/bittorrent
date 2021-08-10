package discovery

// This file implements BEP 15: UDP Tracker Protocol for BitTorrent
// reference: https://www.bittorrent.org/beps/bep_0015.html

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/VIVelev/bittorrent/peer"
)

type udpAction uint32

const (
	protocolId       uint64 = 0x0000041727101980 // magic constant
	maxUDPPacketSize int    = 20000
)

const (
	connectAction = udpAction(iota)
	announceAction
	scrapeAction
	errorAction
)

// Do not expect packets to be exactly of a certain size.

type connectRequest struct {
	transactionId uint32
}

func (creq *connectRequest) marshal(conn net.PacketConn, raddr net.Addr) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, protocolId)
	binary.Write(buf, binary.BigEndian, connectAction)
	binary.Write(buf, binary.BigEndian, creq.transactionId)

	if _, err := conn.WriteTo(buf.Bytes(), raddr); err != nil {
		return err
	}
	return nil
}

type connectResponse struct {
	action        udpAction
	transactionId uint32
	connectionId  uint64
}

func (cres *connectResponse) unmarshal(conn net.PacketConn) (*connectResponse, error) {
	var buf [16]byte
	_, _, err := conn.ReadFrom(buf[:])
	if err != nil {
		return nil, err
	}

	cres.action = udpAction(binary.BigEndian.Uint32(buf[0:4]))
	cres.transactionId = binary.BigEndian.Uint32(buf[4:8])
	cres.connectionId = binary.BigEndian.Uint64(buf[8:16])
	return cres, nil
}

func (cres *connectResponse) validate(req *connectRequest) bool {
	if cres.action != connectAction {
		return false
	}

	if cres.transactionId != req.transactionId {
		return false
	}

	return true
}

type announceEvent uint32

const (
	none = announceEvent(iota)
	completed
	started
	stopped
)

type announceRequest struct {
	connectionId  uint64
	transactionId uint32
	infoHash      [20]byte
	peerId        [20]byte
	downloaded    uint64
	left          uint64
	uploaded      uint64
	event         announceEvent
	ip            uint32
	key           uint32 // used for statistics made by the tracker
	numWant       uint32
	port          uint16
}

func (areq *announceRequest) marshal(conn net.PacketConn, raddr net.Addr) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, areq.connectionId)
	binary.Write(buf, binary.BigEndian, announceAction)
	binary.Write(buf, binary.BigEndian, areq.transactionId)
	buf.Write(areq.infoHash[:])
	buf.Write(areq.peerId[:])
	binary.Write(buf, binary.BigEndian, areq.downloaded)
	binary.Write(buf, binary.BigEndian, areq.left)
	binary.Write(buf, binary.BigEndian, areq.uploaded)
	binary.Write(buf, binary.BigEndian, areq.event)
	binary.Write(buf, binary.BigEndian, areq.ip)
	binary.Write(buf, binary.BigEndian, areq.key)
	binary.Write(buf, binary.BigEndian, areq.numWant)
	binary.Write(buf, binary.BigEndian, areq.port)

	n, err := conn.WriteTo(buf.Bytes(), raddr)
	if err != nil {
		return err
	}
	if n != 98 {
		return fmt.Errorf("somethig went wrong, wrote only %d bytes, should have written 98", n)
	}

	return nil
}

type announceResponse struct {
	action        udpAction
	transactionId uint32
	interval      uint32
	leechers      uint32
	seeders       uint32
	peers         []byte // <IPv4><Port> pairs (6 bytes in-total)
}

func (ares *announceResponse) String() string {
	return fmt.Sprintf("Leechers: %d\nSeeders: %d\nPeers: %v\n", ares.leechers, ares.seeders, ares.peers)
}

func (ares *announceResponse) unmarshal(conn net.PacketConn) (*announceResponse, error) {
	var buf [maxUDPPacketSize]byte
	n, _, err := conn.ReadFrom(buf[:])
	if err != nil {
		return nil, err
	}

	if n < 20 {
		return nil, fmt.Errorf("expected at least 20 bytes, instead read only %d", n)
	}

	ares.action = udpAction(binary.BigEndian.Uint32(buf[0:4]))
	ares.transactionId = binary.BigEndian.Uint32(buf[4:8])
	ares.interval = binary.BigEndian.Uint32(buf[8:12])
	ares.leechers = binary.BigEndian.Uint32(buf[12:16])
	ares.seeders = binary.BigEndian.Uint32(buf[16:20])
	ares.peers = buf[20:n]

	return ares, nil
}

func (ares *announceResponse) validate(req *announceRequest) bool {
	if ares.action != announceAction {
		return false
	}

	if ares.transactionId != req.transactionId {
		return false
	}

	return true
}

const (
	maxTimeout            time.Duration = 3840 * time.Second // 15 * 2 ^ 8
	connectionIdValidTime time.Duration = 2 * time.Minute
)

func timeoutSetter(conn *net.UDPConn) (func() time.Duration, func() time.Duration) {
	calcTimeout := func(n int) time.Duration {
		return time.Duration(15*math.Pow(2, float64(n))) * time.Second
	}

	n := 0
	setTimeout := func() time.Duration {
		t := calcTimeout(n)
		(*conn).SetDeadline(time.Now().Add(t))
		n++
		return t
	}
	resetTimeout := func() time.Duration {
		n = 0
		return setTimeout()
	}
	return setTimeout, resetTimeout

}

func UdpRequestPeers(announce string, left int, infoHash, peerId [20]byte, port uint16) ([]peer.Peer, error) {
	// TODO: take into account connectionIdValidTime
	// TODO: take into account possible error responses

	// dial the announce url
	u, err := url.Parse(announce)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "udp" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	log.Printf("Dialing tracker %s.\n", u.Host)
	raddr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		return nil, err
	}

	setTimeout, resetTimeout := timeoutSetter(conn)
	t := setTimeout()

	// make a connection request
	log.Println("Make a connection request to the tracker.")
	connReq := &connectRequest{
		transactionId: rand.Uint32(),
	}
	var connRes *connectResponse
	for t <= maxTimeout {
		log.Println("Current timeout:", t)
		log.Println("Requesting...")
		err := connReq.marshal(conn, raddr)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				t = setTimeout()
				continue
			}
			return nil, fmt.Errorf("connect request: %s", err)
		}

		log.Println("Waiting for a connection response...")
		connRes, err = new(connectResponse).unmarshal(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				t = setTimeout()
				continue
			}
			return nil, fmt.Errorf("connect response: %s", err)
		}

		if !connRes.validate(connReq) {
			return nil, errors.New("the connect response is invalid")
		}
		break
	}

	if connRes == nil {
		return nil, errors.New("connect: timeout exceeded")
	}

	t = resetTimeout()

	log.Println("Connection Id:", connRes.connectionId)

	// make an announce request
	log.Println("Make an announce request to the tracker.")
	announceReq := &announceRequest{
		connectionId:  connRes.connectionId,
		transactionId: rand.Uint32(),
		infoHash:      infoHash,
		peerId:        peerId,
		downloaded:    0,
		left:          uint64(left),
		uploaded:      0,
		event:         none,
		ip:            0,
		key:           42, // TODO: more robust implementation
		numWant:       ^uint32(0),
		port:          port,
	}
	var announceRes *announceResponse
	for t <= maxTimeout {
		log.Println("Current timeout:", t)
		log.Println("Requesting...")
		err := announceReq.marshal(conn, raddr)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				t = setTimeout()
				continue
			}
			return nil, fmt.Errorf("announnce request: %s", err)
		}

		log.Println("Waiting for an announce response...")
		announceRes, err = new(announceResponse).unmarshal(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				t = setTimeout()
				continue
			}
			return nil, fmt.Errorf("announce response: %s", err)
		}

		if !announceRes.validate(announceReq) {
			return nil, errors.New("the announce response is invalid")
		}
		break
	}

	if announceRes == nil {
		return nil, errors.New("announce: timeout exceeded")
	}

	log.Printf("Got an announce response:\n%s", announceRes)

	return peer.UnmarshalCompact(announceRes.peers)
}
