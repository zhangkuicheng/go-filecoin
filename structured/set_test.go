package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	assert := assert.New(t)
	ds := NewDatastore(newTestChunkstore())

	s := NewSet(ds)
	assert.True(s.Empty())
	assert.Equal(uint64(0), s.Len())

	s1, err := s.Insert(String("foo"))
	assert.NoError(err)
	assert.False(s1.Empty())
	assert.Equal(uint64(1), s1.Len())
	assert.True(s1.Has(String("foo")))

	assert.True(s.Empty())
	assert.Equal(uint64(0), s.Len())

	assert.True(s.Equals(NewSet(ds).Value))
	assert.False(s.Equals(s1.Value))
	s11, err := s1.Insert(String("foo"))
	assert.NoError(err)
	assert.True(s1.Equals(s11.Value))

	s2, err := s1.Insert(String("bar"))
	assert.False(s2.Empty())
	assert.Equal(uint64(2), s2.Len())
	assert.False(s2.Equals(s1.Value))

	s12, err := s2.Delete(String("bar"))
	assert.NoError(err)
	assert.Equal(uint64(1), s12.Len())
	assert.True(s12.Equals(s1.Value))

	testIter := func(s Set, expected ...string) {
		it := s.Iterate()
		for i := 0; i < len(expected); i++ {
			ok, err := it.Next()
			assert.NoError(err)
			if !ok {
				assert.Equal(i, len(expected))
				break
			}
			assert.True(String(expected[i]).Equals(it.Value()))
		}
		ok, err := it.Next()
		assert.False(ok)
		assert.NoError(err)
		assert.True(it.Value().Zero())
		ok, err = it.Next()
		assert.False(ok)
		assert.NoError(err)
		assert.True(it.Value().Zero())
	}

	testIter(s)
	testIter(s1, "foo")
	testIter(s11, "foo")
	testIter(s12, "foo")
	testIter(s2, "bar", "foo")
}
