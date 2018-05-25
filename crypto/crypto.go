// TAKEN FROM: https://github.com/ethereum/go-ethereum/blob/master/crypto/crypto.go
package crypto

import (
	"crypto/rand"
	"math/big"

	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

func ZeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

func GenerateSecp256k1Key() (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateSecp256k1Key(rand.Reader)
}
