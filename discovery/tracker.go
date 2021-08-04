// package discovery implements peer discovery
package discovery

import (
	"net/url"
	"strconv"

	"github.com/VIVelev/bittorrent/io"
)

type bencodeResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func buildURL(tf *io.TorrentFile, peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(tf.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(tf.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(tf.Length)},
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

func requestPeers(tf *io.TorrentFile, peerID [20]byte, port uint16) (string, error) {
	return "", nil
}
