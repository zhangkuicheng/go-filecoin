package consensus

import (
	"context"

	"github.com/filecoin-project/go-filecoin/types"
)

type Algorithm interface {
	NewValidTipSet(ctx context.Context, blks []*types.Block) (types.TipSet, error)
	ValidateBlockStructure(ctx context.Context, b *types.Block) error
	Weight(ctx context.Context, ts types.TipSet) (uint64, uint64, error)
}
