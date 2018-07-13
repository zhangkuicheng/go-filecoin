package structured

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

// Datastore provides high-level access to read and write structured values.
type Datastore struct {
	vs *types.ValueStore
}

func NewDatastore(cs Chunkstore) Datastore {
	return Datastore{
		vs: types.NewValueStore(nomsChunkStore{cs}),
	}
}

func (ds Datastore) Write(v Value) (r Ref, err error) {
	err = d.Try(func() {
		r = Ref{Value{ds.vs.WriteValue(v.v)}}
	})
	return r, err
}

func (ds Datastore) Read(r Ref) (v Value, err error) {
	err = d.Try(func() {
		v = Value{ds.vs.ReadValue(r.noms().TargetHash())}
	})
	return v, nil
}

func (ds Datastore) Head() (v Value, err error) {
	h := ds.vs.Root()
	if h.IsEmpty() {
		return v, nil
	}
	err = d.Try(func() {
		v = Value{ds.vs.ReadValue(h)}
	})
	return v, err
}

func (ds Datastore) Commit(new, old Value) (err error) {
	var nh, oh hash.Hash
	if !new.Zero() {
		nr, err := ds.Write(new)
		if err != nil {
			return err
		}
		nh = mustCidToHash(nr.Target())
	}
	if !old.Zero() {
		oh = mustCidToHash(old.Cid())
	}

	var ok bool
	err = d.Try(func() {
		ok = ds.vs.Commit(nh, oh)
	})
	if err != nil {
		return err
	}
	if !ok {
		return ErrStaleHead
	}
	return nil
}
