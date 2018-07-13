package structured

import (
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
)

// IsType returns true if the value is of a specific type.
// The format of typedef is implementation-dependent.
func (v Value) IsType(typedef string) bool {
	t := nomdl.MustParseType(typedef)
	return types.TypeOf(v.v).Equals(t)
}

// IsSubtype returns true if the value if of a specific type or a subtype of that type.
// The format of typedef is implementation-dependent.
func (v Value) IsSubtype(typedef string) bool {
	t := nomdl.MustParseType(typedef)
	return types.IsSubtype(t, types.TypeOf(v.v))
}
