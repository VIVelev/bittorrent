package peers

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

// Peer represents connection information for a peer.
type Peer struct {
	IP   net.IP
	Port uint16
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

// Unmarshal parses bytes to peers.
func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 bytes for IP, 2 for Port
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("peers bin must be a multiple of %d", peerSize)
	}

	peers := make([]Peer, len(peersBin)/peerSize)
	for i := range peers {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16([]byte(peersBin[offset+4 : offset+6]))
	}
	return peers, nil
}
