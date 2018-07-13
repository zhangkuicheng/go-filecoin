package structured

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/hash"
	"github.com/filecoin-project/go-filecoin/util/chk"
)

// nomsChunkStore implements Noms' ChunkStore interface in terms of structured's Chunkstore interface.
// This is an implementation detail of package structured.
type nomsChunkStore struct {
	cs Chunkstore
}

func (cs nomsChunkStore) Get(h hash.Hash) chunks.Chunk {
	cid := mustHashToCid(h)
	c, err := cs.cs.Get(cid)
	chk.True(err == nil)
	return c.c
}

func (cs nomsChunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	for h := range hashes {
		c := cs.Get(h)
		foundChunks <- &c
	}
}

func (cs nomsChunkStore) Has(h hash.Hash) bool {
	cid := mustHashToCid(h)
	ok, err := cs.cs.Has(cid)
	chk.True(err == nil)
	return ok
}

func (cs nomsChunkStore) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
	for h := range hashes {
		if !cs.Has(h) {
			if absent == nil {
				absent = hash.HashSet{}
			}
			absent.Insert(h)
		}
	}
	return
}

func (cs nomsChunkStore) Put(c chunks.Chunk) {
	err := cs.cs.Put(Chunk{c})
	chk.True(err == nil)
}

func (cs nomsChunkStore) Version() string {
	return constants.NomsVersion
}

func (cs nomsChunkStore) Rebase() {
	panic("notimplemented")
}

func (cs nomsChunkStore) Root() hash.Hash {
	cid, err := cs.cs.Root()
	chk.True(err == nil)
	return mustCidToHash(cid)
}

func (cs nomsChunkStore) Commit(current, last hash.Hash) bool {
	err := cs.cs.Commit(mustHashToCid(current), mustHashToCid(last))
	if err == ErrStaleHead {
		return false
	}
	chk.True(err == nil)
	return true
}

func (cs nomsChunkStore) Stats() interface{} {
	return nil
}

func (cs nomsChunkStore) StatsSummary() string {
	return ""
}

func (cs nomsChunkStore) Close() error {
	return nil
}
