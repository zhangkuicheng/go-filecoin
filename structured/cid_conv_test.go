package structured

import (
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/stretchr/testify/assert"
)

func TestRoundtrip(t *testing.T) {
	assert := assert.New(t)

	expected := hash.Of([]byte("abc"))
	assert.Equal("rmnjb8cjc5tblj21ed4qs821649eduie", expected.String())
	cid, err := hashToCid(expected)
	assert.NoError(err)

	actual, err := cidToHash(cid)
	assert.Equal("rmnjb8cjc5tblj21ed4qs821649eduie", actual.String())
	assert.NoError(err)
	assert.Equal(expected, actual)
}

func TestEmpty(t *testing.T) {
	assert := assert.New(t)

	cid, err := hashToCid(hash.Hash{})
	assert.Nil(cid)
	assert.NoError(err)

	h, err := cidToHash(nil)
	assert.True(h.IsEmpty())
	assert.NoError(err)
}

func TestErrors(t *testing.T) {
	// TODO
}
