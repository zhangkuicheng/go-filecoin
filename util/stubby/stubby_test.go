package stubby

import (
	"testing"

	"github.com/filecoin-project/go-filecoin/util/swap"
	"github.com/stretchr/testify/assert"
)

func TestStubby(t *testing.T) {
	assert := assert.New(t)

	r := Registry{}
	r.Add("foo", func(a int, b string) (ao int, bo string, err error) {
		return a, b, nil
	})
	var a int
	var b string
	var err error
	assert.True(r.Call("foo", 42, "foo", &a, &b, &err))
	assert.Equal(42, a)
	assert.Equal("foo", b)
	assert.Nil(err)

	assert.False(r.Call("bar"))
}

func TestEmpty(t *testing.T) {
	assert := assert.New(t)
	r := Registry{}
	assert.False(r.Call("foo"))
}

func TestAlloc(t *testing.T) {
	assert := assert.New(t)

	r := Registry{}
	r.Add("foo", func(i int, s string, r Registry, rp *Registry) {})
	assert.Equal(0.0, testing.AllocsPerRun(1, func() {
		r.Call("bar", 42, "bar", r, &r)
	}))
}

func TestBadOutputType(t *testing.T) {
	assert := assert.New(t)

	r := Registry{}
	r.Add("foo", func() (a int) {
		return 42
	})
	var a int
	assert.PanicsWithValue("Invalid dst type for output arg: int. Must be pointer.", func() {
		r.Call("foo", a)
	})
}

func TestPanicOutsideTest(t *testing.T) {
	defer swap.Swap(&underTest, false)()
	assert := assert.New(t)
	r := Registry{}
	assert.PanicsWithValue("Add should not be called outside of tests", func() { r.Add("foo", func() {}) })
	assert.PanicsWithValue("Remove should not be called outside of tests", func() { r.Remove("foo") })
}
