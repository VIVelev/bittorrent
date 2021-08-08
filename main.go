package main

import (
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/VIVelev/bittorrent/discovery"
	"github.com/VIVelev/bittorrent/io"
	"github.com/VIVelev/bittorrent/p2p"
	"github.com/VIVelev/bittorrent/peer"
)

func main() {
	tf, err := io.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	peerID := peer.RandID()
	port := peer.DownloadPort
	peers, err := discovery.RequestPeers(tf, peerID, port)
	if err != nil {
		panic(err)
	}
	if len(peers) == 0 {
		panic("0 peers were found")
	}

	data := p2p.Download(tf, peerID, peers)
	err = ioutil.WriteFile(tf.Name, data, fs.ModePerm)
	if err != nil {
		panic(err)
	}
}
