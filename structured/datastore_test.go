package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatastoreWriteRead(t *testing.T) {
	assert := assert.New(t)
	ds := NewDatastore(newTestChunkstore())
	expected := String("foo")
	r, err := ds.Write(expected)
	assert.NoError(err)
	assert.False(r.Zero())
	actual, err := ds.Read(r)
	assert.NoError(err)
	assert.True(expected.Equals(actual))

	expected = String("bar")
	actual, err = ds.Read(expected.AddressOf())
	assert.True(actual.Zero())
	assert.NoError(err)
}

func TestDatastoreCommitHead(t *testing.T) {
	assert := assert.New(t)
	ds := NewDatastore(newTestChunkstore())

	foo := String("foo")
	bar := String("bar")

	h, err := ds.Head()
	assert.NoError(err)
	assert.True(h.Zero())
	err = ds.Commit(foo, bar)
	assert.Equal(ErrStaleHead, err)

	err = ds.Commit(foo, Value{})
	assert.NoError(err)
	h, err = ds.Head()
	assert.NoError(err)
	assert.True(foo.Equals(h))

	err = ds.Commit(foo, bar)
	assert.Equal(ErrStaleHead, err)
	h, err = ds.Head()
	assert.NoError(err)
	assert.True(foo.Equals(h))

	err = ds.Commit(bar, foo)
	assert.NoError(err)
	h, err = ds.Head()
	assert.NoError(err)
	assert.True(bar.Equals(h))
}
