package structured

import (
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/attic-labs/noms/go/types"
)

// Ref is a reference to another Value. This is different than a Cid, because a
// Ref is itself also a Value. Wherever you can find a Value, you can find a Ref.
type Ref struct {
	Value
}

func (r Ref) noms() types.Ref {
	return r.v.(types.Ref)
}

func (r Ref) Target() *cid.Cid {
	return mustHashToCid(r.noms().TargetHash())
}
