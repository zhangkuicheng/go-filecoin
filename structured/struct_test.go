package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStruct(t *testing.T) {
	assert := assert.New(t)

	s1 := NewStruct().Set("foo", Bool(true))
	assert.True(s1.Has("foo"))
	assert.False(s1.Has("bar"))
	assert.True(Bool(true).Equals(s1.Get("foo")))
	assert.True(Value{}.Equals(s1.Get("bar")))

	s2 := s1.Set("bar", String("baz"))
	assert.True(s2.Has("bar"))
	assert.True(s2.Has("foo"))

	assert.False(s1.Has("bar"))
}
