package structured

import (
	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/chunks"
	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/constants"
	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/hash"

	"github.com/filecoin-project/go-filecoin/util/chk"
)

// chunkStoreNoms implements the Noms ChunkStore interface in terms of structured's Chunkstore interface.
// This is an implementation detail of package structured.
type chunkStoreNoms struct {
	cs Chunkstore
}

func (cs chunkStoreNoms) Get(h hash.Hash) chunks.Chunk {
	cid := mustHashToCid(h)
	c, err := cs.cs.Get(cid)
	chk.True(err == nil)
	return c.c
}

func (cs chunkStoreNoms) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	for h := range hashes {
		c := cs.Get(h)
		foundChunks <- &c
	}
}

func (cs chunkStoreNoms) Has(h hash.Hash) bool {
	cid := mustHashToCid(h)
	ok, err := cs.cs.Has(cid)
	chk.True(err == nil)
	return ok
}

func (cs chunkStoreNoms) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
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

func (cs chunkStoreNoms) Put(c chunks.Chunk) {
	err := cs.cs.Put(Chunk{c})
	chk.True(err == nil)
}

func (cs chunkStoreNoms) Version() string {
	return constants.NomsVersion
}

func (cs chunkStoreNoms) Rebase() {
	panic("notimplemented")
}

func (cs chunkStoreNoms) Root() hash.Hash {
	cid, err := cs.cs.Root()
	chk.True(err == nil)
	return mustCidToHash(cid)
}

func (cs chunkStoreNoms) Commit(current, last hash.Hash) bool {
	err := cs.cs.Commit(mustHashToCid(current), mustHashToCid(last))
	if err == ErrStaleHead {
		return false
	}
	chk.True(err == nil)
	return true
}

func (cs chunkStoreNoms) Stats() interface{} {
	return nil
}

func (cs chunkStoreNoms) StatsSummary() string {
	return ""
}

func (cs chunkStoreNoms) Close() error {
	return nil
}
