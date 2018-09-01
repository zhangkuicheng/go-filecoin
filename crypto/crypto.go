package crypto

import (
	"crypto/ecdsa"
	"io"

	lCrypto "gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
	logging "gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmZp3eKdYQHHAneECmeK6HhiMwTPufmjC8DuuaGKv3unvx/blake2b-simd"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/crypto/internal"
	cu "github.com/filecoin-project/go-filecoin/crypto/internal/util"
)

var log = logging.Logger("crypto")

// Sign cryptographically signs `data` using the private key `priv`.
func Sign(priv *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	hash := blake2b.Sum256(data)
	// sign the content
	sig, err := internal.Sign(hash[:], priv)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to sign data")
	}

	return sig, nil
}

// Verify cryptographically verifies that 'sig' is the signed hash of 'data' with
// the public key `pk`.
func Verify(pk, data, signature []byte) (bool, error) {
	hash := blake2b.Sum256(data)
	// remove recovery id
	sig := signature[:len(signature)-1]
	return internal.VerifySignature(pk, hash[:], sig), nil
}

// Ecrecover returns an uncompressed public key that could produce the given
// signature from data.
// Note: The returned public key should not be used to verify `data` is valid
// since a public key may have N private key pairs
func Ecrecover(data, signature []byte) ([]byte, error) {
	hash := blake2b.Sum256(data)
	return internal.Ecrecover(hash[:], signature)
}

// Recoverer is an interface for ecrecover.
type Recoverer interface {
	Ecrecover(data []byte, sig Signature) ([]byte, error)
}

// Signer is an interface for SignBytes.
type Signer interface {
	SignBytes(data []byte, addr address.Address) (Signature, error)
}

// -- Reexports

// SerializeUncompressed serializes a public key in a 65-byte uncompressed
// format.
func SerializeUncompressed(p *ecdsa.PublicKey) []byte {
	return cu.SerializeUncompressed(p)
}

// GenerateKey generates an ecdsa private key
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return internal.GenerateKey()
}

// ECDSAToBytes exports a private key into a binary dump.
func ECDSAToBytes(priv *ecdsa.PrivateKey) []byte {
	return internal.ECDSAToBytes(priv)
}

// BytesToECDSA creates a private key with the given D value.
func BytesToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return internal.BytesToECDSA(d)
}

// ECDSAPubToBytes marshals `pub` to a slice of bytes
func ECDSAPubToBytes(pub *ecdsa.PublicKey) []byte {
	return internal.ECDSAPubToBytes(pub)
}

// PubKey represents a public key in memory.
type PubKey = lCrypto.PubKey

// PrivKey represents a private key in memory.
type PrivKey = lCrypto.PrivKey

// UnmarshalPrivateKey deserializes the given bytes into private key.
func UnmarshalPrivateKey(data []byte) (PrivKey, error) {
	return lCrypto.UnmarshalPrivateKey(data)
}

// GenerateRSAKeyPair creates a RSA public private key pair of the given bit size.
func GenerateRSAKeyPair(bits int) (PrivKey, PubKey, error) {
	return lCrypto.GenerateKeyPair(lCrypto.RSA, bits)
}

// GenerateEd25519Key creates a new ED25519 Key.
func GenerateEd25519Key(src io.Reader) (PrivKey, PubKey, error) {
	return lCrypto.GenerateEd25519Key(src)
}
