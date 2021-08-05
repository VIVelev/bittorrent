package p2p

import (
	"crypto/rand"
	"log"

	"github.com/VIVelev/bittorrent/io"
	"github.com/VIVelev/bittorrent/peer"
)

var PeerID [20]byte

const Port uint16 = 6881

func init() {
	rand.Read(PeerID[:])
}

func Download(tf *io.TorrentFile, peers []peer.Peer) error {
	log.Println("Starting download for", tf.Name)
	return nil
}
