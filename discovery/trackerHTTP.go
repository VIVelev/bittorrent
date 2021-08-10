package discovery

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/VIVelev/bittorrent/peer"
	"github.com/jackpal/bencode-go"
)

type bencodeHttpResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

// buildURL builds a HTTP request url.
func buildURL(announce string, left int, infoHash, peerId [20]byte, port uint16) (string, error) {
	base, err := url.Parse(announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash": []string{string(infoHash[:])},
		"peer_id":   []string{string(peerId[:])},
		// "ip":         []string{}, // optional
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{strconv.Itoa(left)},
		// "event":      []string{"started"}, // optional, one of: started, completed, stopped
		"compact": []string{"1"}, // BEP 23
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

// httpRequestPeers asks the tracker over HTTP at announce about peers, introducing itself with peerID and port.
func httpRequestPeers(announce string, left int, infoHash, peerId [20]byte, port uint16) ([]peer.Peer, error) {
	announceURL, err := buildURL(announce, left, infoHash, peerId, port)
	if err != nil {
		return nil, fmt.Errorf("buildURL: %s", err)
	}

	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get(announceURL)
	if err != nil {
		return nil, fmt.Errorf("get: %s", err)
	}

	defer resp.Body.Close()
	trackerResp := bencodeHttpResponse{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, fmt.Errorf("response: %s", err)
	}

	return peer.UnmarshalCompact([]byte(trackerResp.Peers))
}
