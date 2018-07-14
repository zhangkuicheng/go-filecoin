package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	assert := assert.New(t)

	var c Chunk
	assert.True(c.Empty())
	assert.Nil(c.Data())
	assert.Nil(c.Cid())

	c = NewChunk(nil)
	assert.True(c.Empty())
	assert.Nil(c.Data())
	assert.Nil(c.Cid())

	c = NewChunk([]byte("abc"))
	assert.False(c.Empty())
	assert.Equal("abc", string(c.Data()))
	// https://github.com/attic-labs/noms/blob/master/go/hash/hash_test.go#L84
	assert.Equal("rmnjb8cjc5tblj21ed4qs821649eduie", mustCidToHash(c.Cid()).String())
}
