package actor

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/hash"
	noms "github.com/attic-labs/noms/go/types"
)

// chunkStore is a temporary hacky implementation of the Noms ChunkStore interface
// that stores an entire Noms database inside the old actor memory field.
type chunkStore struct {
	data map[hash.Hash]chunks.Chunk
	root hash.Hash
}

func (cs chunkStore) Get(h hash.Hash) chunks.Chunk {
	return cs.data[h]
}

func (cs chunkStore) GetMany(hs hash.HashSet, foundChunks chan *chunks.Chunk) {
	for h := range hs {
		c, ok := cs.data[h]
		if ok {
			foundChunks <- &c
		}
	}
}

func (cs chunkStore) Has(h hash.Hash) bool {
	_, ok := cs.data[h]
	return ok
}

func (cs chunkStore) HasMany(hs hash.HashSet) (absent hash.HashSet) {
	for h := range hs {
		_, ok := cs.data[h]
		if !ok {
			absent.Insert(h)
		}
	}
	return
}

func (cs *chunkStore) Put(c chunks.Chunk) {
	cs.data[c.Hash()] = c
}

func (cs chunkStore) Version() string {
	return constants.NomsVersion
}

func (cs *chunkStore) Rebase() {
	// TODO: this temporary chunkStore doesn't support concurrent users
}

func (cs chunkStore) Root() hash.Hash {
	return cs.root
}

func (cs *chunkStore) Commit(current, last hash.Hash) bool {
	if last != cs.root {
		return false
	}
	cs.root = current
	return true
}

func (cs chunkStore) Stats() interface{} {
	return nil
}

func (cs chunkStore) StatsSummary() string {
	return ""
}

func (cs chunkStore) Close() error {
	return nil
}

func NewValueStore() *noms.ValueStore {
	cs := &chunkStore{
		data: map[hash.Hash]chunks.Chunk{},
	}
	return noms.NewValueStore(cs)
}
