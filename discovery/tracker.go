// package discovery implements peer discovery
package discovery

import (
	"errors"
	"log"
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

func buildURL(announce string, tf *io.TorrentFile, peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(announce)
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

func buildURLs(tf *io.TorrentFile, peerID [20]byte, port uint16) ([]string, error) {
	// try using the announce-list first
	if tf.AnnounceList != nil {
		var urls []string
		for _, tier := range tf.AnnounceList {
			for _, announce := range tier {
				if announce[:4] == "http" {
					url, err := buildURL(announce, tf, peerID, port)
					if err != nil {
						return nil, err
					}
					urls = append(urls, url)
				}
			}
		}

		return urls, nil
	}

	url, err := buildURL(tf.Announce, tf, peerID, port)
	return []string{url}, err
}

// RequestPeers asks the tracker from tf about peers, introducing itself with peerID and port.
func RequestPeers(tf *io.TorrentFile, peerID [20]byte, port uint16) ([]peer.Peer, error) {
	urls, err := buildURLs(tf, peerID, port)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	c := &http.Client{Timeout: 15 * time.Second}
	for _, announceURL := range urls {
		base, _ := url.Parse(announceURL)
		log.Println("Contacting tracker at:", base.Host)

		resp, err = c.Get(announceURL)
		if err != nil {
			log.Println("Failed:", err.Error())
			continue
		}

		defer resp.Body.Close()
		trackerResp := bencodeResponse{}
		err = bencode.Unmarshal(resp.Body, &trackerResp)
		if err != nil {
			return nil, err
		}
		return peer.UnmarshalCompact([]byte(trackerResp.Peers))
	}

	return nil, errors.New("couldn't connect to a tracker")
}
