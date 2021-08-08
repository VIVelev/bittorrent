package io

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

const hashLen int = 20

type bencodeFile struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeInfo struct {
	Name        string        `bencode:"name"`   // name of the file or directory
	Length      int           `bencode:"length"` // present in the single-file case
	Files       []bencodeFile `bencode:"files"`  // present in the multi-file case
	PieceLength int           `bencode:"piece length"`
	Pieces      string        `bencode:"pieces"` // sha1 checksums of the pieces
}

func (i bencodeInfo) hash() ([hashLen]byte, error) {
	buf := new(bytes.Buffer)
	var val interface{}
	if i.Files != nil {
		val = struct {
			Name        string        `bencode:"name"`
			Files       []bencodeFile `bencode:"files"`
			PieceLength int           `bencode:"piece length"`
			Pieces      string        `bencode:"pieces"`
		}{i.Name, i.Files, i.PieceLength, i.Pieces}
	} else {
		val = struct {
			Name        string `bencode:"name"`
			Length      int    `bencode:"length"`
			PieceLength int    `bencode:"piece length"`
			Pieces      string `bencode:"pieces"`
		}{i.Name, i.Length, i.PieceLength, i.Pieces}
	}

	if err := bencode.Marshal(buf, val); err != nil {
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
		if _, err := io.ReadFull(r, pieceHashes[i][:]); err != nil {
			return nil, err
		}
	}

	if r.Len() != 0 {
		return nil, errors.New("something went wrong")
	}

	return
}

type bencodeTorrent struct {
	Announce     string      `bencode:"announce"`
	AnnounceList [][]string  `bencode:"announce-list"` // BEP 12
	Info         bencodeInfo `bencode:"info"`
}

func (bto bencodeTorrent) toTorrentFile() (*TorrentFile, error) {
	h, err := bto.Info.hash()
	if err != nil {
		return &TorrentFile{}, err
	}

	hashes, err := bto.Info.splitPieces()
	if err != nil {
		return &TorrentFile{}, err
	}

	var length int
	if bto.Info.Files != nil {
		for _, f := range bto.Info.Files {
			length += f.Length
		}
	} else {
		length = bto.Info.Length
	}

	return &TorrentFile{
		Announce:     bto.Announce,
		AnnounceList: bto.AnnounceList,
		InfoHash:     h,
		Name:         bto.Info.Name,
		IsMultiFile:  bto.Info.Files != nil,
		Length:       length,
		Files:        bto.Info.Files,
		PieceLength:  bto.Info.PieceLength,
		PieceHashes:  hashes,
	}, nil
}

// TorrentFile represents the metadata from the .torrent file.
type TorrentFile struct {
	Announce     string
	AnnounceList [][]string
	InfoHash     [20]byte
	Name         string
	IsMultiFile  bool
	Length       int
	Files        []bencodeFile
	PieceLength  int
	PieceHashes  [][hashLen]byte
}

// Open parses a torrent file.
func Open(path string) (*TorrentFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return &TorrentFile{}, err
	}
	defer f.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(f, &bto)
	if err != nil {
		return &TorrentFile{}, err
	}

	return bto.toTorrentFile()
}
