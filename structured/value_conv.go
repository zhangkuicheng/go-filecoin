package structured

import (
	"math/big"

	"github.com/attic-labs/noms/go/types"
)

// structured -> go

func (v Value) Bool() (r bool) {
	if v.Kind() == BoolKind {
		r = bool(v.v.(types.Bool))
	}
	return r
}

func (v Value) Uint8() uint8 {
	panic("TODO")
}
func (v Value) Uint16() uint16 {
	panic("TODO")
}
func (v Value) Uint32() uint32 {
	panic("TODO")
}
func (v Value) Uint64() uint64 {
	panic("TODO")
}
func (v Value) Int8() int8 {
	panic("TODO")
}
func (v Value) Int16() int16 {
	panic("TODO")
}
func (v Value) Int32() int32 {
	panic("TODO")
}
func (v Value) Int64() int64 {
	panic("TODO")
}
func (v Value) BigInt() big.Int {
	panic("TODO")
}
func (v Value) BigRational() big.Rat {
	panic("TODO")
}

func (v Value) String() (r string) {
	if v.Kind() == StringKind {
		r = string(v.v.(types.String))
	}
	return r
}

func (v Value) Bytes() []byte {
	panic("TODO")
}

func (v Value) Set() (r Set) {
	if v.Kind() == SetKind {
		r = Set{v}
	}
	return r
}

func (v Value) Map() (r Map) {
	if v.Kind() == MapKind {
		r = Map{v}
	}
	return r
}

func (v Value) Ref() (r Ref) {
	if v.Kind() == RefKind {
		r = Ref{v}
	}
	return r
}

func (v Value) Struct() (r Struct) {
	if v.Kind() == StructKind {
		r = Struct{v}
	}
	return r
}

// go -> structured

func Bool(v bool) Value {
	return Value{types.Bool(v)}
}

func Uint8(v uint8) (r Value) {
	panic("TODO")
	return
}
func Uint16(v uint16) (r Value) {
	panic("TODO")
	return
}
func Uint32(v uint32) (r Value) {
	panic("TODO")
	return
}
func Uint64(v uint64) (r Value) {
	panic("TODO")
	return
}
func Int8(v int8) (r Value) {
	panic("TODO")
	return
}
func Int16(v int16) (r Value) {
	panic("TODO")
	return
}
func Int32(v int32) (r Value) {
	panic("TODO")
	return
}
func Int64(v int64) (r Value) {
	panic("TODO")
	return
}
func BigInt(v big.Int) (r Value) {
	panic("TODO")
	return
}
func BigRational(v big.Rat) (r Value) {
	panic("TODO")
	return
}

func String(v string) Value {
	return Value{types.String(v)}
}

func Bytes(v []byte) (r Value) {
	panic("TODO")
	return
}

// more complex/heavy types use "New..." constructors
