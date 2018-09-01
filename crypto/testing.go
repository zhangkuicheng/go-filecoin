package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"io"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/crypto/internal"
)

// MockRecoverer implements the Recoverer interface
type MockRecoverer struct{}

// Ecrecover returns an uncompressed public key that could produce the given
// signature from data.
// Note: The returned public key should not be used to verify `data` is valid
// since a public key may have N private key pairs
func (mr *MockRecoverer) Ecrecover(data []byte, sig Signature) ([]byte, error) {
	return Ecrecover(data, sig)
}

// MockSigner implements the Signer interface
type MockSigner struct {
	AddrKeyInfo map[address.Address]KeyInfo
	Addresses   []address.Address
}

// NewMockSigner returns a new mock signer, capable of signing data with
// keys (addresses derived from) in keyinfo
func NewMockSigner(kis []KeyInfo) MockSigner {
	var ms MockSigner
	ms.AddrKeyInfo = make(map[address.Address]KeyInfo)
	for _, k := range kis {
		// get the secret key
		sk, err := internal.BytesToECDSA(k.PrivateKey)
		if err != nil {
			panic(err)
		}
		// extract public key
		pub, ok := sk.Public().(*ecdsa.PublicKey)
		if !ok {
			panic("unknown public key type")
		}
		addrHash := address.Hash(SerializeUncompressed(pub))
		newAddr := address.NewMainnet(addrHash)
		ms.Addresses = append(ms.Addresses, newAddr)
		ms.AddrKeyInfo[newAddr] = k

	}
	return ms
}

// SignBytes cryptographically signs `data` using the Address `addr`.
func (ms MockSigner) SignBytes(data []byte, addr address.Address) (Signature, error) {
	ki, ok := ms.AddrKeyInfo[addr]
	if !ok {
		panic("unknown address")
	}

	sk, err := internal.BytesToECDSA(ki.PrivateKey)
	if err != nil {
		return Signature{}, err
	}

	return Sign(sk, data)
}

const (
	// SECP256K1 is a curve used to compute private keys
	SECP256K1 = "secp256k1"
)

// MustGenerateKeyInfo generates a slice of KeyInfo size `n` with seed `seed`
func MustGenerateKeyInfo(n int, seed io.Reader) []KeyInfo {
	var keyinfos []KeyInfo
	for i := 0; i < n; i++ {
		prv, err := internal.GenerateKeyFromSeed(seed)
		if err != nil {
			panic(err)
		}

		ki := &KeyInfo{
			PrivateKey: internal.ECDSAToBytes(prv),
			Curve:      SECP256K1,
		}
		keyinfos = append(keyinfos, *ki)
	}
	return keyinfos
}

// GenerateKeyInfoSeed returns a reader to be passed to MustGenerateKeyInfo
func GenerateKeyInfoSeed() io.Reader {
	token := make([]byte, 512)
	if _, err := rand.Read(token); err != nil {
		panic(err)
	}
	return bytes.NewReader(token)
}
