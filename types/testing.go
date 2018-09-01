package types

import (
	"gx/ipfs/QmZFbDTY9jfSBms2MchvYM9oYRbAF19K7Pby47yDBfpPrb/go-cid"

	"github.com/stretchr/testify/assert"
)

// Type-related test helpers.

// HasCid allows two values with CIDs to be compared.
type HasCid interface {
	Cid() *cid.Cid
}

// AssertHaveSameCid asserts that two values have identical CIDs.
func AssertHaveSameCid(a *assert.Assertions, m HasCid, n HasCid) {
	if !m.Cid().Equals(n.Cid()) {
		a.Fail("CIDs don't match", "not equal %v %v", m.Cid(), n.Cid())
	}
}

// AssertCidsEqual asserts that two CIDS are identical.
func AssertCidsEqual(a *assert.Assertions, m *cid.Cid, n *cid.Cid) {
	if !m.Equals(n) {
		a.Fail("CIDs don't match", "not equal %v %v", m, n)
	}
}
