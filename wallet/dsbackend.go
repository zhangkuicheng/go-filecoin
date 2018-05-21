package wallet

import (
	"crypto/rand"
	"reflect"
	"strings"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	dsq "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/query"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"

	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/types"
)

// DSBackendType is the reflect type of the DSBackend.
var DSBackendType = reflect.TypeOf(&DSBackend{})

// DSBackend is a wallet backend implementation for storing addresses in a datastore.
type DSBackend struct {
	lk sync.RWMutex

	// TODO: use a better interface that supports time locks, encryption, etc.
	ds repo.Datastore

	// TODO: proper cache
	cache map[types.Address]struct{}
}

var _ Backend = (*DSBackend)(nil)

// NewDSBackend constructs a new backend using the passed in datastore.
func NewDSBackend(ds repo.Datastore) (*DSBackend, error) {
	result, err := ds.Query(dsq.Query{
		KeysOnly: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query datastore")
	}

	list, err := result.Rest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read query results")
	}

	cache := make(map[types.Address]struct{})
	for _, el := range list {
		parsedAddr, err := types.NewAddressFromString(strings.Trim(el.Key, "/"))
		if err != nil {
			return nil, errors.Wrapf(err, "trying to restore invalid address: %s", el.Key)
		}
		cache[parsedAddr] = struct{}{}
	}

	return &DSBackend{
		ds:    ds,
		cache: cache,
	}, nil
}

// Addresses returns a list of all addresses that are stored in this backend.
func (backend *DSBackend) Addresses() []types.Address {
	backend.lk.RLock()
	defer backend.lk.RUnlock()

	var cpy []types.Address
	for addr := range backend.cache {
		cpy = append(cpy, addr)
	}
	return cpy
}

// HasAddress checks if the passed in address is stored in this backend.
// Safe for concurrent access.
func (backend *DSBackend) HasAddress(addr types.Address) bool {
	backend.lk.RLock()
	defer backend.lk.RUnlock()

	_, ok := backend.cache[addr]
	return ok
}

// NewAddress creates a new address and stores it.
// Safe for concurrent access.
func (backend *DSBackend) NewAddress() (types.Address, error) {
	// Generate a key pair that may be used to authenticate messages
	// from an address.
	priv, pub, err := ci.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return types.Address{}, err
	}

	// Zero out the public and private keys for security reasons.
	// TODO: We need a common way to zero out sensitive data
	var bpriv []byte
	var bpub []byte
	defer func() {
		priv = nil
		pub = nil
		bpriv = bpriv[:cap(bpriv)]
		bpub = bpub[:cap(bpriv)]
	}()

	bpub, err = pub.Bytes()
	if err != nil {
		return types.Address{}, err
	}

	// An address is derived from a public key. This is what allows you to get
	// money out of the actor, if you have the matching private key for the address.
	adderHash, err := types.AddressHash(bpub)
	if err != nil {
		return types.Address{}, err
	}
	// TODO: Use the address type we are running on from the config.
	newAdder := types.NewMainnetAddress(adderHash)

	bpriv, err = priv.Bytes()
	if err != nil {
		return types.Address{}, err
	}

	backend.lk.Lock()
	defer backend.lk.Unlock()

	// Persist the address (public key) and its corresponding private key.
	if err := backend.ds.Put(ds.NewKey(newAdder.String()), bpriv); err != nil {
		return types.Address{}, errors.Wrap(err, "failed to store new address")
	}

	backend.cache[newAdder] = struct{}{}

	return newAdder, nil
}

// Sign cryptographically signs `data` using the private key of address `addr`.
// TODO Zero out the sensitive data when complete
func (backend *DSBackend) Sign(addr types.Address, data []byte) ([]byte, error) {
	// Check that we are storing the address to sign for.
	priv, err := backend.getPrivateKey(addr)
	if err != nil {
		return nil, err
	}

	return priv.Sign(data)
}

// Verify cryptographically verifies that 'sig' is the signed hash of 'data' for
// the key `bpub`.
func (backend *DSBackend) Verify(bpub, data, sig []byte) (bool, error) {
	pub, err := ci.UnmarshalPublicKey(bpub)
	if err != nil {
		return false, err
	}
	return pub.Verify(data, sig)
}

// getPrivateKey fetches and unmarshals the private key pointed to by address `addr`.
// TODO Zero out the sensitive data when complete
func (backend *DSBackend) getPrivateKey(addr types.Address) (ci.PrivKey, error) {
	bpriv, err := backend.ds.Get(ds.NewKey(addr.String()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch private key from backend")
	}
	return ci.UnmarshalPrivateKey(bpriv.([]byte))
}

// getPublicKey fetches and unmarshals the public key pointed used to generate `addr`.
// TODO Zero out the sensitive data when complete
func (backend *DSBackend) getPublicKey(addr types.Address) (ci.PubKey, error) {
	priv, err := backend.getPrivateKey(addr)
	if err != nil {
		return nil, err
	}
	return priv.GetPublic(), nil
}
