package crypto

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"

	sha256 "github.com/minio/sha256-simd"

	"github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/assert"
)

func TestSign(t *testing.T) {
	assert := assert.New(t)

	genPrivateKey, err := GenerateSecp256k1Key()
	assert.NoError(err)

	data := []byte("foo")
	poopoo := []byte("bar")

	msg := sha256.Sum256(data)
	badMsg := sha256.Sum256(poopoo)

	sig, err := Sign(msg[:], (*ecdsa.PrivateKey)(genPrivateKey))
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}

	sigv := sig[:len(sig)-1] // remove recovery id
	valid, err := VerifySignature(genPrivateKey.PubKey().SerializeUncompressed(), msg[:], sigv)
	assert.NoError(err)
	assert.True(valid)

	valid, err = VerifySignature(genPrivateKey.PubKey().SerializeUncompressed(), badMsg[:], sigv)
	assert.NoError(err)
	assert.False(valid)

	recoverPubKey, err := Ecrecover(msg[:], sig)
	assert.NoError(err)

	valid, err = VerifySignature(recoverPubKey, msg[:], sigv)
	assert.NoError(err)
	assert.True(valid)

	valid, err = VerifySignature(recoverPubKey, badMsg[:], sigv)
	assert.NoError(err)
	assert.False(valid)

	cig, err := sign(genPrivateKey, msg[:])
	assert.NoError(err)
	valid, err = verify(msg[:], cig)
	assert.True(valid)

	valid, err = verify(badMsg[:], cig)
	assert.NoError(err)
	assert.False(valid)
}

// Sign cryptographically signs `data` using the private key of address `addr`.
// TODO Zero out the sensitive data when complete
func sign(priv *btcec.PrivateKey, hash []byte) ([]byte, error) {

	// sign the content
	sig, err := Sign(hash[:], (*ecdsa.PrivateKey)(priv))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to sign data")
	}

	fmt.Printf("\nSIGN - \nsk:\t%x\npk:\t%x\nsig:\t%x\nhash:\t%x\n\n", priv.Serialize(), priv.PubKey().SerializeUncompressed(), sig, hash[:])
	return sig, nil
}

// Verify cryptographically verifies that 'sig' is the signed hash of 'data'.
func verify(hash, signature []byte) (bool, error) {
	// recover the public key from the content and the sig
	pk, err := Ecrecover(hash[:], signature)
	if err != nil {
		return false, errors.Wrap(err, "Failed to verify data")
	}

	// remove recovery id
	sig := signature[:len(signature)-1]
	valid, err := VerifySignature(pk, hash[:], sig)
	if err != nil {
		return false, err
	}

	fmt.Printf("\nVERIFY - \npk:\t%x\n sig:\t%x\n hash:\t%x\n valid:\t%t\n\n", pk, signature, hash[:], valid)
	return valid, nil
}
