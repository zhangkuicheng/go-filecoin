package stubby

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubby(t *testing.T) {
	assert := assert.New(t)

	r := Registry{}
	r.Add("foo", func(a int, b string) (ao int, bo string, err error) {
		return a, b, nil
	})
	f := r.Get("foo")
	assert.NotNil(f)
	var a int
	var b string
	var err error
	f(42, "foo", &a, &b, &err)
	assert.Equal(42, a)
	assert.Equal("foo", b)
	assert.Nil(err)

	assert.Nil(r.Get("bar"))
}

func TestAlloc(t *testing.T) {
	assert := assert.New(t)

	r := Registry{}
	r.Add("foo", func() {})
	assert.Equal(0.0, testing.AllocsPerRun(1, func() {
		if f := r.Get("bar"); f != nil {
			panic("unexpected") // can't use assert.Nil(f) because the assert family all allocate
		}
	}))
}
