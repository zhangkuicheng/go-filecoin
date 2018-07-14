package structured

import (
	"errors"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

// Chunkstore is implemented by the client of this package (e.g., the vm) to provide raw chunk storage.
type Chunkstore interface {
	// Has returns whether the Chunkstore contains a particluar chunk.
	Has(cid *cid.Cid) (bool, error)
	// Get retrieves a chunk. If no such chunk exists in the Chunkstore, then an empty Chunk is returned.
	Get(cid *cid.Cid) (Chunk, error)
	// Put puts a chunk into the Chunkstore. If it already exists in the Chunkstore, Put is a no-op.
	Put(c Chunk) error
	// Root returns the Cid of the current root of the Chunkstore. If no root exists, Root returns a nil pointer.
	Root() (*cid.Cid, error)
	// Commit updates the root of the Chunkstore and flushes all chunks put since last commit to persistent storage.
	// If the specified old cid is not the same as the current Root() then ErrStaleHead is returned and nothing is changed.
	Commit(new, old *cid.Cid) error
}

// ErrStaleHead is returned when Commit is called based on an outdated root value.
var ErrStaleHead = errors.New("Attempt to commit against outdated head")
