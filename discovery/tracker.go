// package discovery implements peer discovery
package discovery

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/VIVelev/bittorrent/io"
	"github.com/VIVelev/bittorrent/peer"
	"github.com/jackpal/bencode-go"
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
		"info_hash": []string{string(tf.InfoHash[:])},
		"peer_id":   []string{string(peerID[:])},
		// "ip":         []string{}, // optional
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{strconv.Itoa(int(tf.Length))},
		// "event":      []string{"started"}, // optional, one of: started, completed, stopped
		"compact": []string{"1"}, // BEP 23
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

// RequestPeers asks the tracker from tf about peers, introducing itself with peerID and port.
func RequestPeers(tf *io.TorrentFile, peerID [20]byte, port uint16) ([]peer.Peer, error) {
	url, err := buildURL(tf, peerID, port)
	if err != nil {
		return nil, err
	}

	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	trackerResp := bencodeResponse{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	return peer.UnmarshalCompact([]byte(trackerResp.Peers))
}
