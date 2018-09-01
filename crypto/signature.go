package crypto

import (
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

// Signature is the result of a cryptographic sign operation.
type Signature = types.Bytes

// VerifySignature cryptographically verifies that 'sig' is the signed hash of 'data' with
// the public key belonging to `addr`.
func VerifySignature(data []byte, addr address.Address, sig Signature) bool {
	maybePk, err := Ecrecover(data, sig)
	if err != nil {
		// Any error returned from Ecrecover means this signature is not valid.
		log.Infof("error in signature validation: %s", err)
		return false
	}
	maybeAddrHash := address.Hash(maybePk)

	return address.NewMainnet(maybeAddrHash) == addr
}
