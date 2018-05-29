package wallet

import (
	"testing"

	"gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"

	"github.com/filecoin-project/go-filecoin/types"

	"github.com/stretchr/testify/assert"
)

func TestWalletSimple(t *testing.T) {
	assert := assert.New(t)

	t.Log("create a backend")
	ds := datastore.NewMapDatastore()
	fs, err := NewDSBackend(ds)
	assert.NoError(err)

	t.Log("create a wallet with a single backend")
	w := New(fs)

	t.Log("check backends")
	assert.Len(w.Backends(DSBackendType), 1)

	t.Log("create a new address in the backend")
	addr, err := fs.NewAddress()
	assert.NoError(err)

	t.Log("test HasAddress")
	assert.True(w.HasAddress(addr))

	t.Log("find backend")
	backend, err := w.Find(addr)
	assert.NoError(err)
	assert.Equal(fs, backend)

	t.Log("find unknown address")
	randomAddr := types.NewAddressForTestGetter()()

	assert.False(w.HasAddress(randomAddr))

	t.Log("list all addresses")
	list := w.Addresses()
	assert.Len(list, 1)
}

func TestSimpleSignAndVerify(t *testing.T) {
	assert := assert.New(t)

	t.Log("create a backend")
	ds := datastore.NewMapDatastore()
	fs, err := NewDSBackend(ds)
	assert.NoError(err)

	t.Log("create a wallet with a single backend")
	w := New(fs)

	t.Log("check backends")
	assert.Len(w.Backends(DSBackendType), 1)

	t.Log("create a new address in the backend")
	addr, err := fs.NewAddress()
	assert.NoError(err)

	t.Log("test HasAddress")
	assert.True(w.HasAddress(addr))

	t.Log("find backend")
	backend, err := w.Find(addr)
	assert.NoError(err)
	assert.Equal(fs, backend)

	// data to sign
	dataA := []byte("THIS IS A SIGNED SLICE OF DATA")
	t.Log("sign content")
	sig, err := w.Sign(addr, dataA)
	assert.NoError(err)

	t.Log("verify signed content")
	valid, err := w.Verify(dataA, sig)
	assert.NoError(err)
	assert.True(valid)

	// data that is unsigned
	dataB := []byte("I AM UNSIGNED DATA!")
	t.Log("verify fails for unsigned content")
	valid, err = w.Verify(dataB, sig)
	assert.NoError(err)
	assert.False(valid)
}

func TestMultiWalletSignAndVerify(t *testing.T) {
	assert := assert.New(t)

	t.Log("create two backends")
	ds1 := datastore.NewMapDatastore()
	fs1, err := NewDSBackend(ds1)
	assert.NoError(err)

	ds2 := datastore.NewMapDatastore()
	fs2, err := NewDSBackend(ds2)
	assert.NoError(err)

	t.Log("create two wallets each with single backend")
	w1 := New(fs1)
	w2 := New(fs2)

	t.Log("check backends")
	assert.Len(w1.Backends(DSBackendType), 1)
	assert.Len(w2.Backends(DSBackendType), 1)

	t.Log("create a new address in each backend")
	addr1, err := fs1.NewAddress()
	assert.NoError(err)

	addr2, err := fs2.NewAddress()
	assert.NoError(err)

	t.Log("test HasAddress")
	assert.True(w1.HasAddress(addr1))
	assert.True(w2.HasAddress(addr2))

	t.Log("find backends")
	be1, err := w1.Find(addr1)
	assert.NoError(err)
	assert.Equal(fs1, be1)

	be2, err := w2.Find(addr2)
	assert.NoError(err)
	assert.Equal(fs2, be2)

	// produce some valid sig's for testing
	data1 := []byte("foo")
	data2 := []byte("bar")
	sig1, _ := signAndVerify(t, data1, addr1, w1, fs1)
	sig2, _ := signAndVerify(t, data2, addr2, w2, fs2)

	// Verify the work done by a different peer
	t.Log("verify when missing private key")
	valid, err := w2.Verify(data1, sig1)
	assert.NoError(err)
	assert.True(valid)
	valid, err = w1.Verify(data2, sig2)
	assert.NoError(err)
	assert.True(valid)

	// Error test cases below
	t.Log("invalid private key / address for signing")
	_, err = w1.Sign(addr2, []byte("i am data!"))
	assert.Contains(err.Error(), ErrUnknownAddress.Error())

	t.Log("invalid public key for verify")
	valid, err = w1.Verify(data1, sig1)
	assert.NoError(err)
	assert.False(valid)
	valid, err = w2.Verify(data2, sig2)
	assert.NoError(err)
	assert.False(valid)

	t.Log("invalid signature for verify")
	valid, err = w1.Verify(data1, sig2)
	assert.NoError(err)
	assert.False(valid)
	valid, err = w2.Verify(data2, sig1)
	assert.NoError(err)
	assert.False(valid)

	t.Log("invalid data for verify")
	valid, err = w1.Verify(data2, sig1)
	assert.NoError(err)
	assert.False(valid)
	valid, err = w2.Verify(data1, sig2)
	assert.NoError(err)
	assert.False(valid)
}

// a helper to sign and verify data, returning the sig from sign and the public key for verify.
func signAndVerify(t *testing.T, data []byte, addr types.Address, w *Wallet, fs *DSBackend) (sig, bpub []byte) {
	t.Log("sign content")
	sig, err := w.Sign(addr, data)
	assert.NoError(t, err)

	// Get the public used to generate the address
	pub, err := fs.getPublicKey(addr)
	assert.NoError(t, err)
	bpub = pub.SerializeUncompressed()

	t.Log("verify signed content")
	valid, err := w.Verify(data, sig)
	assert.NoError(t, err)
	assert.True(t, valid)

	return sig, bpub
}
