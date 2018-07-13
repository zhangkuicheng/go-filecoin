package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	assert := assert.New(t)
	ds := NewDatastore(newTestChunkstore())

	m := NewMap(ds)
	assert.True(m.Empty())
	assert.Equal(uint64(0), m.Len())
	v, err := m.Get(String("foo"))
	assert.NoError(err)
	assert.True(v.Zero())

	m1, err := m.Set(String("foo"), String("foot"))
	assert.NoError(err)
	assert.False(m1.Empty())
	assert.Equal(uint64(1), m1.Len())
	assert.True(m1.Has(String("foo")))
	v, err = m1.Get(String("foo"))
	assert.True(String("foot").Equals(v))

	assert.True(m.Empty())
	assert.Equal(uint64(0), m.Len())
	assert.True(m.Equals(NewMap(ds).Value))
	assert.False(m.Equals(m1.Value))

	m11, err := m1.Set(String("foo"), String("foot"))
	assert.NoError(err)
	assert.True(m1.Equals(m11.Value))

	m2, err := m1.Set(String("bar"), String("bark"))
	assert.False(m2.Empty())
	assert.Equal(uint64(2), m2.Len())
	assert.False(m2.Equals(m1.Value))

	m12, err := m2.Delete(String("bar"))
	assert.NoError(err)
	assert.Equal(uint64(1), m12.Len())
	assert.True(m12.Equals(m1.Value))

	testIter := func(m Map, expected ...string) {
		it := m.Iterate()
		for i := 0; i < len(expected); i += 2 {
			ok, err := it.Next()
			assert.NoError(err)
			if !ok {
				break
			}
			assert.True(String(expected[i]).Equals(it.Key()))
			assert.True(String(expected[i+1]).Equals(it.Value()))
		}
		ok, err := it.Next()
		assert.False(ok)
		assert.NoError(err)
		assert.True(it.Key().Zero())
		assert.True(it.Value().Zero())
		ok, err = it.Next()
		assert.False(ok)
		assert.NoError(err)
		assert.True(it.Key().Zero())
		assert.True(it.Value().Zero())
	}

	testIter(m)
	testIter(m1, "foo", "foot")
	testIter(m11, "foo", "foot")
	testIter(m12, "foo", "foot")
	testIter(m2, "bar", "bark", "foo", "foot")
}
