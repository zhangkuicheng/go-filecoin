package structured

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestValueBasics(t *testing.T) {
	assert := assert.New(t)
	ds := NewDatastore(newTestChunkstore())
	tc := []struct {
		v Value
		k Kind
	}{
		{Bool(true), BoolKind},
		{Bool(false), BoolKind},
		{String(""), StringKind},
		{String("foo"), StringKind},
		{NewSet(ds).Value, SetKind},
		{NewMap(ds).Value, MapKind},
		{NewStruct().Value, StructKind},
		{String("foo").AddressOf().Value, RefKind},
	}

	// kind
	for _, t := range tc {
		assert.Equal(t.k, t.v.Kind())
		for k, ks := range KindNames {
			ty := reflect.TypeOf(t.v)
			m, _ := ty.MethodByName("Is" + ks)
			r := m.Func.Call([]reflect.Value{reflect.ValueOf(t.v)})
			assert.Equal(1, len(r))
			rb := r[0].Interface().(bool)
			assert.Equal(k == t.k, rb)
		}
	}

	// equals
	for i, t1 := range tc {
		for j, t2 := range tc {
			eq := t1.v.Equals(t2.v)
			assert.Equal(eq, i == j)
		}
	}

	// ref
	for _, t := range tc {
		ds.Write(t.v)
		v, err := ds.Read(t.v.AddressOf())
		assert.NoError(err)
		assert.True(t.v.Equals(v))
	}
}

func TestValueWalkRef(t *testing.T) {
	assert := assert.New(t)
	ms := (&chunks.MemoryStorage{}).NewView()
	ns := types.NewSet(types.NewValueStore(ms))
	ne := ns.Edit()
	for i := 0; i < 10000; i++ {
		ne.Insert(types.String(fmt.Sprintf("%d", i)))
	}
	ns = ne.Set()
	nr := []types.Ref{}
	ns.WalkRefs(func(r types.Ref) {
		nr = append(nr, r)
	})

	ss := Set{Value{ns}}
	sr := []Ref{}
	ss.WalkRefs(func(r Ref) {
		sr = append(sr, r)
	})

	assert.Equal(len(nr), len(sr))
	for i, nr := range nr {
		sr := sr[i]
		assert.Equal(nr.Hash(), sr.noms().Hash())
	}
}
