package peer

import "math/rand"

const (
	DownloadPort uint16 = 6881
)

func RandID() (id [20]byte) {
	rand.Read(id[:])
	return
}
