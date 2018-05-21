package wallet

import "github.com/filecoin-project/go-filecoin/types"

// Backend is the interface to represent different storage backends
// that can contain many addresses.
type Backend interface {
	// Addresses returns a list of all accounts currently stored in this backend.
	Addresses() []types.Address

	// Contains returns true if this backend stores the passed in address.
	HasAddress(addr types.Address) bool

	// Sign cryptographically signs `data` using the private key of address `addr`.
	// TODO Zero out the sensitive data when complete
	Sign(addr types.Address, data []byte) ([]byte, error)

	// Verify cryptographically verifies that 'sig' is the signed hash of 'data' for
	// the key `bpub`.
	Verify(bpub, data, sig []byte) (bool, error)
}
