package structured

import (
	"testing"

	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/chunks"
	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/constants"
	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/hash"

	"github.com/stretchr/testify/assert"
)

func TestChunkStoreNoms(t *testing.T) {
	assert := assert.New(t)

	csn := chunkStoreNoms{newTestChunkstore()}
	c1 := chunks.NewChunk([]byte("foo"))
	c2 := chunks.NewChunk([]byte("bar"))
	c3 := chunks.NewChunk([]byte("baz"))
	csn.Put(c1)
	csn.Put(c2)

	assert.Equal(c1, csn.Get(c1.Hash()))
	assert.Equal(c2, csn.Get(c2.Hash()))
	assert.True(csn.Get(c3.Hash()).IsEmpty())

	ch := make(chan *chunks.Chunk, 10)
	csn.GetMany(hash.NewHashSet(c1.Hash(), c2.Hash()), ch)
	close(ch)
	found := hash.HashSet{}
	for c := range ch {
		found.Insert(c.Hash())
	}
	assert.Equal(2, len(found))
	assert.True(found.Has(c1.Hash()))
	assert.True(found.Has(c2.Hash()))

	assert.True(csn.Has(c1.Hash()))
	assert.True(csn.Has(c2.Hash()))
	assert.False(csn.Has(c3.Hash()))

	absent := csn.HasMany(hash.NewHashSet(c1.Hash(), c2.Hash(), c3.Hash()))
	assert.Equal(1, len(absent))
	assert.True(absent.Has(c3.Hash()))

	assert.Equal(constants.NomsVersion, csn.Version())

	assert.PanicsWithValue("notimplemented", func() {
		csn.Rebase()
	})

	assert.True(csn.Root().IsEmpty())
	assert.False(csn.Commit(c2.Hash(), c1.Hash()))
	assert.True(csn.Commit(c2.Hash(), hash.Hash{}))
	assert.Equal(c2.Hash(), csn.Root())
	assert.False(csn.Commit(c1.Hash(), hash.Hash{}))
}
