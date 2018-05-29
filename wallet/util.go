package wallet

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	sha256 "github.com/minio/sha256-simd"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"

	"github.com/filecoin-project/go-filecoin/crypto"
)

// Sign cryptographically signs `data` using the private key of address `addr`.
// TODO Zero out the sensitive data when complete
func sign(priv *btcec.PrivateKey, data []byte) ([]byte, error) {
	// hash the content before signing
	hash := sha256.Sum256(data)

	// sign the content
	sig, err := crypto.Sign(hash[:], (*ecdsa.PrivateKey)(priv))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to sign data")
	}

	fmt.Printf("\nSIGN - \nsk:\t%x\npk:\t%x\nsig:\t%x\ndata:\t%s\nhash:\t%x\n\n", priv.Serialize(), priv.PubKey().SerializeUncompressed(), sig, string(data), hash[:])
	return sig, nil
}

// Verify cryptographically verifies that 'sig' is the signed hash of 'data'.
func verify(data, signature []byte) (bool, error) {
	// hash the content before verify
	hash := sha256.Sum256(data)

	// recover the public key from the content and the sig
	pk, err := crypto.Ecrecover(hash[:], signature)
	if err != nil {
		return false, errors.Wrap(err, "Failed to verify data")
	}

	// remove recovery id
	sig := signature[:len(signature)-1]
	valid, err := crypto.VerifySignature(pk, hash[:], sig)
	if err != nil {
		return false, err
	}

	fmt.Printf("\nVERIFY - \npk:\t%x\nsig:\t%x\ndata:\t%s\nhash:\t%x\nvalid:\t%t\n\n", pk, signature, string(data), hash[:], valid)
	return valid, nil
}
