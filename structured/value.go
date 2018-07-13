package structured

import (
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// Value represents a single high-level structured value.
// Values can be small (the boolean value `true`) or huge (a PB Map).
// Every Value has one (and only one) Cid(). Computing the Cid() for
// a value requires at most one pass through the entire value.
// Computing a Cid() after an incremental change has a cost
// proportional to complexity of the change.
type Value struct {
	v types.Value
}

func (v Value) Kind() Kind {
	switch v.v.Kind() {
	case types.BoolKind:
		return BoolKind
	case types.StringKind:
		return StringKind
	case types.SetKind:
		return SetKind
	case types.MapKind:
		return MapKind
	case types.StructKind:
		// TODO: check for (u)int*, bytes, Cid
		// Noms does not suppor these types natively, but we can implement them
		// in terms of Noms structs. In the future, as Noms supports more of them,
		// we can just change this bit.
		return StructKind
	case types.RefKind:
		return RefKind
	default:
		// The rest of the Noms types are not supported.
		panic("notreached")
	}
}

func (v Value) Equals(other Value) bool {
	if v.Zero() {
		return other.Zero()
	}
	return v.v.Equals(other.v)
}

func (v Value) AddressOf() Ref {
	return Ref{Value{types.NewRef(v.v)}}
}

func (v Value) Cid() *cid.Cid {
	return mustHashToCid(v.v.Hash())
}

func (v Value) Zero() bool {
	return v == Value{}
}

func (v Value) WalkRefs(f func(r Ref)) error {
	return d.Try(func() {
		v.v.WalkRefs(func(nr types.Ref) {
			f(Ref{Value{nr}})
		})
	})
}
