package crypto

import (
	"crypto/ecdsa"
	"testing"

	sha256 "github.com/minio/sha256-simd"
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
}
