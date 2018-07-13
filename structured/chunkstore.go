package structured

import (
	"errors"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

// Chunkstore is implemented by the host to provide raw chunk storage.
type Chunkstore interface {
	Has(cid *cid.Cid) (bool, error)
	Get(cid *cid.Cid) (Chunk, error)
	Put(c Chunk) error
	Root() (*cid.Cid, error)
	Commit(new, old *cid.Cid) error
}

var ErrStaleHead = errors.New("Attempt to commit against outdated head")
