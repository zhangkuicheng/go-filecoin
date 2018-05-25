package crypto

import (
	"crypto/rand"

	ci "github.com/libp2p/go-libp2p-crypto"
)

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

func GenerateSecp256k1Key() (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateSecp256k1Key(rand.Reader)
}
