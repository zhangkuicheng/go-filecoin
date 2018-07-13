package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubtype(t *testing.T) {
	assert := assert.New(t)
	s0 := NewStruct()
	s1 := NewStruct().Set("foo", String("bar")).Set("hot", Bool(true))
	s2 := NewStruct().Set("foo", String("bar"))
	t1 := `Struct { foo: String, hot: Bool, }`
	t2 := `Struct { foo: String, }`
	t3 := `Struct { foo: String, hot?: Bool, }`
	tc := []struct {
		s     Struct
		isT1  bool
		isT2  bool
		isT3  bool
		isST1 bool
		isST2 bool
		isST3 bool
	}{
		{s0, false, false, false, false, false, false},
		{s1, true, false, false, true, true, true},
		{s2, false, true, false, false, true, true},
	}
	for i, t := range tc {
		assert.Equal(t.isT1, t.s.IsType(t1), "test case %d", i)
		assert.Equal(t.isT2, t.s.IsType(t2), "test case %d", i)
		assert.Equal(t.isT3, t.s.IsType(t3), "test case %d", i)
		assert.Equal(t.isST1, t.s.IsSubtype(t1), "test case %d", i)
		assert.Equal(t.isST2, t.s.IsSubtype(t2), "test case %d", i)
		assert.Equal(t.isST3, t.s.IsSubtype(t3), "test case %d", i)
	}
}
