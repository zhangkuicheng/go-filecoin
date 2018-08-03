package consensus

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/protocol/coinlist-server/chain"
)

var log = logging.Logger("consensus.expected")

// ECV is the constant V defined in the EC spec.
// TODO: the value of V needs motivation at the protocol design level.
const ECV uint64 = 10

// ECPrM is the power ratio magnitude defined in the EC spec.
// TODO: the value of this constant needs motivation at the protocol level.
const ECPrM uint64 = 100

// Expected implements expected consensus.
type Expected struct {
	// PwrTableView provides miner and total power for the EC chain weight
	// computation.
	PwrTableView powerTableView

	chain chain.Chain
}

// Ensure Expected satisfies the Consensus interface at compile time.
var _ Consensus = (*Expected)(nil)

func NewExpected(chain chain.Chain) Expected {
	return Expected{
		chain:        chain,
		PwrTableView: &marketView{},
	}
}

// NewValidTipSet creates a new tipset from the input blocks that is guaranteed
// to be valid. It operates by validating each block and further checking that
// this tipset contains only blocks with the same heights, parent weights,
// and parent sets.
func (c *Expected) NewValidTipSet(ctx context.Context, blks []*types.Block) (types.TipSet, error) {
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

// Weight returns the numerator and denominator of the weight of the input tipset.
func (c *Expected) Weight(ctx context.Context, ts TipSet) (uint64, uint64, error) {
	w, err := c.weight(ctx, ts)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	wNum := w.Num()
	if !wNum.IsUint64() {
		return uint64(0), uint64(0), errors.New("weight numerator cannot be repr by uint64")
	}
	wDenom := w.Denom()
	if !wDenom.IsUint64() {
		return uint64(0), uint64(0), errors.New("weight denominator cannot be repr by uint64")
	}
	return wNum.Uint64(), wDenom.Uint64(), nil
}

// weight returns the EC weight of this TipSet
// TODO: this implementation needs to handle precision correctly, see issue #655.
func (c *Expected) weight(ctx context.Context, ts TipSet) (*big.Rat, error) {
	log.LogKV(ctx, "Weight", ts.String())
	if len(ts) == 1 && ts.ToSlice()[0].Cid().Equals(c.chain.genesisCid) {
		return big.NewRat(int64(0), int64(1)), nil
	}
	// Gather parent and state.
	parentIDs, err := ts.Parents()
	if err != nil {
		return nil, err
	}
	st, err := cm.stateForBlockIDs(ctx, parentIDs)
	if err != nil {
		return nil, err
	}

	wNum, wDenom, err := ts.ParentWeight()
	if err != nil {
		return nil, err
	}
	if wDenom == uint64(0) {
		return nil, errors.New("storage market with 0 bytes stored not handled")
	}
	w := big.NewRat(int64(wNum), int64(wDenom))

	// Each block in the tipset adds ECV + ECPrm * miner_power
	totalBytes, err := c.PwrTableView.Total(ctx, st)
	if err != nil {
		return nil, err
	}
	ratECV := big.NewRat(int64(ECV), int64(1))
	for _, blk := range ts {
		minerBytes, err := c.PwrTableView.Miner(ctx, st, blk.Miner)
		if err != nil {
			return nil, err
		}
		wNumBlk := int64(ECPrM * minerBytes)
		wBlk := big.NewRat(wNumBlk, int64(totalBytes))
		wBlk.Add(wBlk, ratECV)
		w.Add(w, wBlk)
	}
	return w, nil
}
