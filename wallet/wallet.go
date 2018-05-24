package wallet

import (
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"

	btcec "github.com/btcsuite/btcd/btcec"
	sha256 "github.com/minio/sha256-simd"

	"github.com/filecoin-project/go-filecoin/types"
)

var (
	// ErrUnknownAddress is returned when the given address is not stored in this wallet.
	ErrUnknownAddress = errors.New("unknown address")
)

// Wallet manages the locally stored addresses.
type Wallet struct {
	lk sync.Mutex

	backends map[reflect.Type][]Backend
}

// New constructs a new wallet, that manages addresses in all the
// passed in backends.
func New(backends ...Backend) *Wallet {
	backendsMap := make(map[reflect.Type][]Backend)

	for _, backend := range backends {
		kind := reflect.TypeOf(backend)
		backendsMap[kind] = append(backendsMap[kind], backend)
	}

	return &Wallet{
		backends: backendsMap,
	}
}

// HasAddress checks if the given address is stored.
// Safe for concurrent access.
func (w *Wallet) HasAddress(a types.Address) bool {
	_, err := w.Find(a)
	return err == nil
}

// Find searches through all backends and returns the one storing the passed
// in address.
// Safe for concurrent access.
func (w *Wallet) Find(addr types.Address) (Backend, error) {
	w.lk.Lock()
	defer w.lk.Unlock()

	for _, backends := range w.backends {
		for _, backend := range backends {
			if backend.HasAddress(addr) {
				return backend, nil
			}
		}
	}

	return nil, ErrUnknownAddress
}

// Addresses retrieves all stored addresses.
// Safe for concurrent access.
func (w *Wallet) Addresses() []types.Address {
	w.lk.Lock()
	defer w.lk.Unlock()

	var out []types.Address
	for _, backends := range w.backends {
		for _, backend := range backends {
			out = append(out, backend.Addresses()...)
		}
	}

	return out
}

// Backends returns backends by their kind.
func (w *Wallet) Backends(kind reflect.Type) []Backend {
	w.lk.Lock()
	defer w.lk.Unlock()

	cpy := make([]Backend, len(w.backends[kind]))
	copy(cpy, w.backends[kind])
	return cpy
}

// Sign cryptographically signs `data` using the private key of address `addr`.
// TODO Zero out the sensitive data when complete
func (w *Wallet) Sign(addr types.Address, data []byte) ([]byte, error) {
	// Check that we are storing the address to sign for.
	backend, err := w.Find(addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign data")
	}
	return backend.Sign(addr, data)
}

// Verify cryptographically verifies that 'sig' is the signed hash of 'data'.
func (w *Wallet) Verify(data, sig []byte) (bool, error) {
	hash := sha256.Sum256(data)

	mpk, _, err := btcec.RecoverCompact(btcec.S256(), sig, hash[:])
	if err != nil {
		return false, errors.Wrap(err, "wallet :: Failed to recover pk")
	}

	// Because of course this is how things work
	r := big.NewInt(0).SetBytes(sig[1:33])
	s := big.NewInt(0).SetBytes(sig[33:])
	osig := &btcec.Signature{R: r, S: s}

	valid, err := osig.Verify(hash[:], mpk), nil
	if err != nil {
		return false, errors.Wrap(err, "failed to verify data")
	}
	fmt.Printf("\nWALLET-VERIFY: valid: %t\nsig: %x\ndata: %s\nhash: %x\n\n", valid, sig, string(data), hash)
	return valid, nil
}
