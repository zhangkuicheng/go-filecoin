package consensus

import (
	"context"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
)

type Algorithm interface {
	NewValidTipSet(ctx context.Context, blks []*types.Block) (core.TipSet, error)
	ValidateBlockStructure(ctx context.Context, b *types.Block) error
}
