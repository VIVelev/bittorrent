package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/VIVelev/bittorrent/client"
	"github.com/VIVelev/bittorrent/io"
	"github.com/VIVelev/bittorrent/message"
	"github.com/VIVelev/bittorrent/peer"
)

const (
	// MaxBacklog is the max number of unfulfilled requests a client can have in its pipeline.
	MaxBacklog int = 5
	// MaxBlockSize is the largest number of bytes a request can ask for.
	MaxBlockSize int = 16384 // 16KiB
)

type pieceWork struct {
	index    int
	length   int
	checksum [20]byte
}

type downloadedPiece struct {
	index int
	data  []byte
}

func attemptDownloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	// backloged requests to peer
	// requested bytes from peer
	// downloaded bytes from peer
	var backloged, requested, downloaded int
	// store the bytes in-memory
	buf := make([]byte, pw.length)

	// setting a deadline helps get unresponsive peers unstuck
	// 30 seconds is more than enough to download a 262kB piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // disable the deadline

	for downloaded < pw.length {
		if !c.Choked {
			// make at most MaxBacklog requests
			for backloged < MaxBacklog && requested < pw.length {
				blockSize := MaxBlockSize
				// the last block may have less than MaxBlockSize bytes
				if val := pw.length - requested; val < MaxBlockSize {
					blockSize = val
				}

				if err := c.WriteRequest(pw.index, requested, blockSize); err != nil {
					return nil, fmt.Errorf("write request: %s", err)
				}

				backloged++
				requested += blockSize
			}
		}

		msg, err := c.Read()
		if err != nil {
			return nil, fmt.Errorf("read message: %s", err)
		}

		if msg == nil {
			// keep-alive message
			continue
		}

		switch msg.ID {
		case message.MsgChoke:
			c.Choked = true
		case message.MsgUnchoke:
			c.Choked = false
		case message.MsgHave:
			index, err := message.ParseHave(msg)
			if err != nil {
				return nil, err
			}
			c.Bitfield.SetPiece(index)
		case message.MsgPiece:
			n, err := message.ParsePiece(msg, pw.index, buf)
			if err != nil {
				return nil, fmt.Errorf("parse piece: %s", err)
			}
			backloged--
			downloaded += n
		}
	}

	return buf, nil
}

func checkIntegrity(pw *pieceWork, buf []byte) bool {
	hash := sha1.Sum(buf)
	return bytes.Equal(hash[:], pw.checksum[:])
}

func startDownloadWorker(p peer.Peer, infoHash, peerID [20]byte, workQ chan *pieceWork, piecesQ chan *downloadedPiece) {
	c, err := client.New(p, infoHash, peerID)
	if err != nil {
		log.Printf("Could not handshake with %s. Error: %s. Disconnecting.\n", p, err)
		return
	}
	log.Printf("Completed handshake with %s.\n", p)
	defer c.Conn.Close()

	c.WriteUnchoke()
	c.WriteInterested()

	for pw := range workQ {
		if !c.Bitfield.HasPiece(pw.index) {
			workQ <- pw // put back in the queue
			continue
		}

		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			// this peer does not want to talk ;(
			log.Println("Exiting.", err)
			workQ <- pw // put back in the queue
			return
		}

		if !checkIntegrity(pw, buf) {
			log.Printf("Piece %d failed integrity check.\n", pw.index)
			workQ <- pw // put back in the queue
			continue
		}

		c.WriteHave(pw.index)
		piecesQ <- &downloadedPiece{index: pw.index, data: buf}
	}
}

func Download(tf *io.TorrentFile, peerID [20]byte, peers []peer.Peer) []byte {
	log.Println("Starting download for", tf.Name)
	totalPieces := len(tf.PieceHashes)

	// init work and pieces queues, and sigDead channel
	workQ := make(chan *pieceWork, totalPieces)
	piecesQ := make(chan *downloadedPiece)

	// fill in the work q
	for i, hash := range tf.PieceHashes {
		length := tf.PieceLength
		if i == totalPieces-1 {
			length = tf.Length - tf.PieceLength*i
			fmt.Printf("Piece length:%d\n", tf.PieceLength)
			fmt.Printf("Last piece length:%d\n", length)
		}

		workQ <- &pieceWork{index: i, length: length, checksum: hash}
	}

	// start download workers
	log.Printf("Starting a download worker for each peer (%d in total).\n", len(peers))
	for _, p := range peers {
		go startDownloadWorker(p, tf.InfoHash, peerID, workQ, piecesQ)
	}

	// collect download pieces in a buffer until full
	buf := make([]byte, tf.Length)
	numDownloaded := 0
	for numDownloaded < totalPieces {
		piece := <-piecesQ
		if piece == nil {
			return nil
		}
		begin := piece.index * tf.PieceLength
		end := begin + tf.PieceLength
		if end > tf.Length {
			end = tf.Length
		}
		copy(buf[begin:end], piece.data)
		numDownloaded++

		percent := float64(numDownloaded) / float64(totalPieces) * 100
		numWorkers := runtime.NumGoroutine() - 1 // substract the main thread
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, piece.index, numWorkers)
	}
	close(workQ)

	return buf
}
