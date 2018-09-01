package types_test

import (
	"testing"

	"github.com/filecoin-project/go-filecoin/chain"

	"github.com/stretchr/testify/assert"
)

func TestCidForTestGetter(t *testing.T) {
	newCid := chain.NewCidForTestGetter()
	c1 := newCid()
	c2 := newCid()
	assert.False(t, c1.Equals(c2))
	assert.False(t, c1.Equals(chain.SomeCid())) // Just in case.
}

func TestNewMessageForTestGetter(t *testing.T) {
	newMsg := chain.NewMessageForTestGetter()
	m1 := newMsg()
	c1, _ := m1.Cid()
	m2 := newMsg()
	c2, _ := m2.Cid()
	assert.False(t, c1.Equals(c2))
}
