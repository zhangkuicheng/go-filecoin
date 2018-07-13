package structured

func (v Value) IsBool() bool {
	return v.Kind() == BoolKind
}

func (v Value) IsUint8() bool {
	return v.Kind() == Uint8Kind
}

func (v Value) IsUint16() bool {
	return v.Kind() == Uint16Kind
}

func (v Value) IsUint32() bool {
	return v.Kind() == Uint32Kind
}

func (v Value) IsUint64() bool {
	return v.Kind() == Uint64Kind
}

func (v Value) IsInt8() bool {
	return v.Kind() == Int8Kind
}

func (v Value) IsInt16() bool {
	return v.Kind() == Int16Kind
}

func (v Value) IsInt32() bool {
	return v.Kind() == Int32Kind
}

func (v Value) IsInt64() bool {
	return v.Kind() == Int64Kind
}

func (v Value) IsBigInt() bool {
	return v.Kind() == BigIntKind
}

func (v Value) IsBigRational() bool {
	return v.Kind() == BigRationalKind
}

func (v Value) IsString() bool {
	return v.Kind() == StringKind
}

func (v Value) IsBytes() bool {
	return v.Kind() == BytesKind
}

func (v Value) IsSet() bool {
	return v.Kind() == SetKind
}

func (v Value) IsMap() bool {
	return v.Kind() == MapKind
}

func (v Value) IsRef() bool {
	return v.Kind() == RefKind
}

func (v Value) IsStruct() bool {
	return v.Kind() == StructKind
}
