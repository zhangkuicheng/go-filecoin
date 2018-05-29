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
	msg := sha256.Sum256(data)

	sig, err := Sign(msg[:], (*ecdsa.PrivateKey)(genPrivateKey))
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}

	recoveredPub, err := Ecrecover(msg[:], sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}

	assert.Equal(genPrivateKey.PubKey().SerializeUncompressed(), recoveredPub)

}
