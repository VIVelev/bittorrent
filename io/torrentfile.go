package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackpal/bencode-go"
)

const hashLen = 20

type bencodeInfo struct {
	Name        string `bencode:"name"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

func (i bencodeInfo) hash() ([hashLen]byte, error) {
	buf := new(bytes.Buffer)
	if err := bencode.Marshal(buf, i); err != nil {
		return [hashLen]byte{}, err
	}
	return sha1.Sum(buf.Bytes()), nil
}

func (i bencodeInfo) splitPieces() (pieceHashes [][hashLen]byte, err error) {
	r := strings.NewReader(i.Pieces)
	if r.Len()%hashLen != 0 {
		return nil, fmt.Errorf("the length of pieces must be a multiple of %d", hashLen)
	}

	pieceHashes = make([][hashLen]byte, r.Len()/hashLen)
	for i := range pieceHashes {
		io.ReadFull(r, pieceHashes[i][:])
	}

	if r.Len() != 0 {
		return nil, errors.New("something went wrong")
	}

	return
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func (bto bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	h, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}

	hashes, err := bto.Info.splitPieces()
	if err != nil {
		return TorrentFile{}, err
	}

	return TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    h,
		Name:        bto.Info.Name,
		Length:      bto.Info.Length,
		PieceLength: bto.Info.PieceLength,
		PieceHashes: hashes,
	}, nil
}

// TorrentFile represents the metadata from the .torrent file.
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	Name        string
	Length      int
	PieceLength int
	PieceHashes [][hashLen]byte
}

// Open parses a torrent file.
func Open(path string) (TorrentFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer f.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(f, &bto)
	if err != nil {
		return TorrentFile{}, err
	}

	return bto.toTorrentFile()
}
