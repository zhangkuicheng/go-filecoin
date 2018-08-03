package state

import (
	"context"

	"github.com/filecoin-project/go-filecoin/types"
)

// Processor manages all state transitions.
type Proessor interface {
	State(ctx context.Context, blks []*types.Block) (statetree.Tree, error)
}
