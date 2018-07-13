package structured

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// Map is unordered and arbitrarily scalable.
// It is possible to implement ordered maps in a variety of ways (prolly trees, b-treaps, etc)
// so we could consider supporting ordering if use cases call for it.
type Map struct {
	Value
}

func NewMap(ds Datastore) Map {
	return Map{Value{types.NewMap(ds.vs)}}
}

func (m Map) noms() types.Map {
	return m.v.(types.Map)
}

func (m Map) Empty() bool {
	return m.noms().Empty()
}

func (m Map) Len() uint64 {
	return m.noms().Len()
}

func (m Map) Has(k Value) (r bool, err error) {
	err = d.Try(func() {
		r = m.noms().Has(k.v)
	})
	return r, err
}

func (m Map) Get(k Value) (r Value, err error) {
	err = d.Try(func() {
		nr, ok := m.noms().MaybeGet(k.v)
		if ok {
			r = Value{nr}
		}
	})
	return r, err
}

func (m Map) Set(k, v Value) (r Map, err error) {
	err = d.Try(func() {
		r = Map{Value{m.noms().Edit().Set(k.v, v.v).Map()}}
	})
	return r, err
}

func (m Map) Delete(k Value) (r Map, err error) {
	err = d.Try(func() {
		r = Map{Value{m.noms().Edit().Remove(k.v).Map()}}
	})
	return r, err
}

func (m Map) Iterate() MapIterator {
	return MapIterator{
		i: m.noms().Iterator(),
	}
}

type MapIterator struct {
	i    types.MapIterator
	k    Value
	v    Value
	done bool
}

func (i MapIterator) Key() Value {
	return i.k
}

func (i MapIterator) Value() Value {
	return i.v
}

func (i *MapIterator) Next() (r bool, err error) {
	// TODO: We should do something to make iteration order non-sorted, so users don't rely on it
	if i.done {
		return false, nil
	}

	var nk, nv types.Value
	err = d.Try(func() {
		nk, nv = i.i.Next()
	})
	if err != nil {
		return false, err
	}
	if nk == nil {
		i.done = true
		i.k = Value{}
		i.v = Value{}
		return false, nil
	}

	i.k = Value{nk}
	i.v = Value{nv}
	return true, nil
}
