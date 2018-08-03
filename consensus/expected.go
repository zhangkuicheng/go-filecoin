package consensus

import (
	"context"
	"fmt"

	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("consensus.expected")

// Expected implements expected consensus.
type Expected struct {
}

// Ensure Expected satisfies the Consensus interface at compile time.
var _ Consensus = (*Expected)(nil)

func NewExpected() Expected {
	return Expected{}
}

// NewValidTipSet creates a new tipset from the input blocks that is guaranteed
// to be valid. It operates by validating each block and further checking that
// this tipset contains only blocks with the same heights, parent weights,
// and parent sets.
func (c *Expected) NewValidTipSet(ctx context.Context, blks []*types.Block) (core.TipSet, error) {
	for _, blk := range blks {
		if err := c.ValidateBlockStructure(ctx, blk); err != nil {
			return nil, err
		}
	}
	return core.NewTipSet(blks...)
}

// ValidateBlockStructure verifies that this block, on its own, is structurally and
// cryptographically valid. This means checking that all of its fields are
// properly filled out and its signatures are correct. Checking the validity of
// state changes must be done separately and only once the state of the
// previous block has been validated. TODO: not yet signature checking
func (c *Expected) ValidateBlockStructure(ctx context.Context, b *types.Block) error {
	// TODO: validate signatures on messages
	log.LogKV(ctx, "ValidateBlockStructure", b.Cid().String())
	if b.StateRoot == nil {
		return fmt.Errorf("block has nil StateRoot")
	}

	// TODO: validate that this miner had a winning ticket last block.
	// In general this may depend on block farther back in the chain (lookback param).

	return nil
}
