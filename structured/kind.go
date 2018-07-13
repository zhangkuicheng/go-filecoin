package structured

// Kind is a classification of values. By design this is not as rich as a "type". It's a coarse
// classification that serves our purposes. Full support for rich types is left as a potential
// future feature (also relates to IPLD work that is happening).
type Kind uint8

const (
	// Non-chunked kinds

	// BoolKind is the kind for boolean values.
	BoolKind Kind = 0
	// Uint8Kind is the kind for unsigned 8 bit integers.
	Uint8Kind Kind = 1
	// Uint16Kind is the kind for unsigned 16 bit integers.
	Uint16Kind Kind = 2
	// Uint32Kind is the kind for unsigned 32 bit integers.
	Uint32Kind Kind = 3
	// Uint64Kind is the kind for unsigned 64 bit integers.
	Uint64Kind Kind = 4
	// Int8Kind is the kind for signed 8 bit integers.
	Int8Kind Kind = 5
	// Int16Kind is the kind for signed 16 bit integers.
	Int16Kind Kind = 6
	// Int32Kind is the kind for signed 32 bit integers.
	Int32Kind Kind = 7
	// Int64Kind is the kind for signed 64 bit integers.
	Int64Kind Kind = 8
	// BigIntKind is a kind for signed integers larger than 64 bits.
	BigIntKind Kind = 9
	// BigRationalKind is a kind for arbitrary sized rational numbers.
	BigRationalKind Kind = 10
	// StringKind is a kind for variable length single-chunk strings.
	StringKind Kind = 11
	// BytesKind is a kind for variable length single-chunk byte arrays.
	BytesKind Kind = 12

	// Chunked kinds - these will automatically be represented by a merkle dag of chunks internally.

	// SetKind is the kind for structured.Set.
	SetKind Kind = 13

	// MapKind is the kind for structured.Map.
	MapKind Kind = 14

	// Other kinds

	// StructKind is the kind for structured.Set.
	StructKind Kind = 15
	// RefKind is the kind for structured.Ref.
	RefKind Kind = 16 // on-chain reference

	// TODO: OffChainRef ("ExternalRef"? "ForeignRef"? "Foreign"?)

	// List purposely omitted
	// Unknown if there are efficient canonicalizable implementations that support splice other than
	// prolly trees. Also it wasn't that useful in Noms anyway. If we need something like this, we
	// could have Queue or Sequence or something that is non-indexed. That would be more broadly
	// implementable.

	// Blob purposely omitted
	// Same problems as list, plus we don't want to store large binary objects on-chain now anyway.

	// Potential future types:
	// - Bag/Multimap -- useful as indices. Require sorted maps/sets as pre-requisite.
	// - Type -- useful to represent type of values as first-class in some applications.
)

var KindNames = map[Kind]string{
	BoolKind:        "Bool",
	Uint8Kind:       "Uint8",
	Uint16Kind:      "Uint16",
	Uint32Kind:      "Uint32",
	Uint64Kind:      "Uint64",
	Int8Kind:        "Int8",
	Int16Kind:       "Int16",
	Int32Kind:       "Int32",
	Int64Kind:       "Int64",
	BigIntKind:      "BigInt",
	BigRationalKind: "BigRational",
	StringKind:      "String",
	BytesKind:       "Bytes",
	SetKind:         "Set",
	MapKind:         "Map",
	StructKind:      "Struct",
	RefKind:         "Ref",
}
