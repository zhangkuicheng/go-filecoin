package types

import (
	"github.com/filecoin-project/go-filecoin/address"
	wutil "github.com/filecoin-project/go-filecoin/wallet/util"
)

// Signature is the result of a cryptographic sign operation.
type Signature = Bytes

// VerifySignature cryptographically verifies that 'sig' is the signed hash of 'data' with
// the public key belonging to `addr`.
func VerifySignature(data []byte, addr address.Address, sig Signature) bool {
	maybePk, err := wutil.Ecrecover(data, sig)
	if err != nil {
		// Any error returned from Ecrecover means this signature is not valid.
		log.WithError(err).Error("signature invalid")
		return false
	}
	maybeAddrHash := address.Hash(maybePk)

	return address.NewMainnet(maybeAddrHash) == addr
}
