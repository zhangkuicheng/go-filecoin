package structured

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// Set is unordered and arbitrarily scalable.
// It is possible to implement ordered sets in a variety of ways (prolly trees, b-treaps, etc)
// so we could consider supporting ordering if use cases call for it.
type Set struct {
	Value
}

func NewSet(ds Datastore) Set {
	return Set{Value{types.NewSet(ds.vs)}}
}

func (s Set) noms() types.Set {
	return s.v.(types.Set)
}

func (s Set) Empty() bool {
	return s.noms().Empty()
}

func (s Set) Len() uint64 {
	return s.noms().Len()
}

func (s Set) Has(v Value) (r bool, err error) {
	err = d.Try(func() {
		r = s.noms().Has(v.v)
	})
	return r, err
}

func (s Set) Insert(v Value) (r Set, err error) {
	err = d.Try(func() {
		r = Set{Value{s.noms().Edit().Insert(v.v).Set()}}
	})
	return r, err
}

func (s Set) Delete(v Value) (r Set, err error) {
	err = d.Try(func() {
		r = Set{Value{s.noms().Edit().Remove(v.v).Set()}}
	})
	return r, err
}

func (s Set) Iterate() SetIterator {
	return SetIterator{
		i: s.noms().Iterator(),
	}
}

type SetIterator struct {
	i    types.SetIterator
	v    Value
	done bool
}

func (i SetIterator) Value() Value {
	return i.v
}

func (i *SetIterator) Next() (r bool, err error) {
	// TODO: We should do something to make iteration order non-sorted, so users don't rely on it
	if i.done {
		return false, nil
	}
	var nv types.Value
	err = d.Try(func() {
		nv = i.i.Next()
	})
	if err != nil {
		return false, err
	}
	if nv == nil {
		i.v = Value{}
		i.done = true
		return false, nil
	}
	i.v = Value{nv}
	return true, nil
}
