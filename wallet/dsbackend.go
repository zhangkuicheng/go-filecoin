package wallet

import (
	"reflect"
	"strings"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	dsq "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/query"

	"github.com/btcsuite/btcd/btcec"

	"github.com/filecoin-project/go-filecoin/crypto"
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
	// generate a private key
	priv, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return types.Address{}, err
	}

	bpub := priv.PubKey().SerializeUncompressed()
	addrHash, err := types.AddressHash(bpub)
	if err != nil {
		return types.Address{}, err
	}
	// TODO: Use the address type we are running on from the config.
	newAddr := types.NewMainnetAddress(addrHash)

	backend.lk.Lock()
	defer backend.lk.Unlock()

	bpriv := priv.Serialize()
	if err := backend.ds.Put(ds.NewKey(newAddr.String()), bpriv); err != nil {
		return types.Address{}, errors.Wrap(err, "failed to store new address")
	}

	backend.cache[newAddr] = struct{}{}

	return newAddr, nil
}

// Sign Needs comments TODO
func (backend *DSBackend) Sign(addr types.Address, data []byte) ([]byte, error) {
	privateKey, _, err := backend.getKeyPair(addr)
	if err != nil {
		return nil, err
	}
	return sign(privateKey, data)
}

// Verify Needs comments TODO
func (backend *DSBackend) Verify(data, sig []byte) (bool, error) {
	return verify(data, sig)
}

func (backend *DSBackend) getKeyPair(addr types.Address) (*btcec.PrivateKey, *btcec.PublicKey, error) {
	bpriv, err := backend.ds.Get(ds.NewKey(addr.String()))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch private key from backend")
	}
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), bpriv.([]byte))
	return priv, pub, nil
}

func (backend *DSBackend) getPublicKey(addr types.Address) (*btcec.PublicKey, error) {
	priv, _, err := backend.getKeyPair(addr)
	if err != nil {
		return nil, err
	}
	return priv.PubKey(), nil
}
