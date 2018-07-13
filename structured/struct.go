package structured

import (
	"github.com/attic-labs/noms/go/types"
)

// Struct is an unnamed record of name/value pairs. A common question
// is how Struct differs from Map. Two main ways:
//
// 1. Struct is an atomic value, like bool or string. It is intended
//    for smallish amounts of fields, and is therefore not
//    automatically chunked. Maps are intended for potentially huge
//    numbers of keys and therefore are broken down into trees of
//    chunks automatically.
//
// 2. The fields of a struct are part of its type, and can be checked
//    with IsType and IsSubtype(). The keys of a map are not part of
//    its type.
//
// A good rule of thumb is that structs should be used in cases
// where the fields are statically known at design time, Maps should
// be used in cases where the keys are known only at runtime.
type Struct struct {
	Value
}

func NewStruct() Struct {
	return Struct{Value{types.NewStruct("", types.StructData{})}}
}

func (s Struct) noms() types.Struct {
	return s.v.(types.Struct)
}

func (s Struct) Get(n string) Value {
	nv, ok := s.noms().MaybeGet(n)
	if !ok {
		return Value{}
	}
	return Value{nv}
}

func (s Struct) Has(n string) bool {
	_, ok := s.noms().MaybeGet(n)
	return ok
}

func (s Struct) Set(n string, v Value) Struct {
	return Struct{
		Value{
			s.noms().Set(n, v.v),
		},
	}
}
