package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// ZeroBytes Needs comments TODO
func ZeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

// GenerateSecp256k1Key Needs comments TODO
func GenerateSecp256k1Key() (*btcec.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return (*btcec.PrivateKey)(key), nil
}

// ToECDSAPub Needs comments TODO
func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}
}
