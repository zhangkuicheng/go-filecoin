package structured

import (
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

// testChunkstore is a mock implementation of structured.Chunkstore.
type testChunkstore struct {
	m map[string]Chunk
	h *cid.Cid
}

func newTestChunkstore() *testChunkstore {
	return &testChunkstore{
		m: map[string]Chunk{},
	}
}

func (cs testChunkstore) Has(cid *cid.Cid) (bool, error) {
	_, ok := cs.m[cid.String()]
	return ok, nil
}

func (cs testChunkstore) Get(cid *cid.Cid) (c Chunk, err error) {
	return cs.m[cid.String()], nil
}

func (cs *testChunkstore) Put(c Chunk) error {
	cs.m[c.Cid().String()] = c
	return nil
}

func (cs testChunkstore) Root() (*cid.Cid, error) {
	return cs.h, nil
}

func (cs *testChunkstore) Commit(new, old *cid.Cid) error {
	if cs.h == old || (cs.h != nil && old != nil && cs.h.Equals(old)) {
		cs.h = new
		return nil
	}
	return ErrStaleHead
}
