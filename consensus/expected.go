package consensus

import (
	"context"
	"fmt"
	"math/big"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/chain"
	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	pp "github.com/filecoin-project/go-filecoin/util/prettyprint"
)

var log = logging.Logger("consensus.expected")

var (
	// ErrStateRootMismatch is returned when the computed state root doesn't match the expected result.
	ErrStateRootMismatch = errors.New("blocks state root does not match computed result")
	// ErrInvalidBase is returned when the chain doesn't connect back to a known good block.
	ErrInvalidBase = errors.New("block does not connect to a known good chain")
)

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

	chain chain.Store
	store *hamt.CborIpldStore

	// Tracks state by tipset identifier
	stateCache map[string]*cid.Cid
}

// Ensure Expected satisfies the Consensus interface at compile time.
var _ Algorithm = (*Expected)(nil)

func NewExpected(chain chain.Store, store *hamt.CborIpldStore) Expected {
	return Expected{
		chain:        chain,
		store:        store,
		stateCache:   make(map[string]*cid.Cid),
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
	return types.NewTipSet(blks...)
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
func (c *Expected) Weight(ctx context.Context, ts types.TipSet) (uint64, uint64, error) {
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
func (c *Expected) weight(ctx context.Context, ts types.TipSet) (*big.Rat, error) {
	log.LogKV(ctx, "Weight", ts.String())
	if len(ts) == 1 && ts.ToSlice()[0].Cid().Equals(c.chain.GenesisCid()) {
		return big.NewRat(int64(0), int64(1)), nil
	}
	// Gather parent and state.
	parentIDs, err := ts.Parents()
	if err != nil {
		return nil, err
	}
	// TODO: how to access state here
	st, err := c.StateForBlockIDs(ctx, parentIDs)
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

// State returns the aggregate state tree for the blocks or an error if the
// blocks are not a valid tipset or are not part of a valid chain.
func (c *Expected) State(ctx context.Context, blks []*types.Block) (state.Tree, error) {
	ts, err := c.NewValidTipSet(ctx, blks)
	if err != nil {
		return nil, errors.Wrapf(err, "blks do not form a valid tipset: %s", pp.StringFromBlocks(blks))
	}

	// Return cache hit
	if root, ok := c.stateCache[ts.String()]; ok { // tipset in cache
		return state.LoadStateTree(ctx, c.store, root, builtin.Actors)
	}

	// Base case is the genesis block
	if len(ts) == 1 && blks[0].Cid().Equals(c.chain.GenesisCid()) { // genesis tipset
		return state.LoadStateTree(ctx, c.store, blks[0].StateRoot, builtin.Actors)
	}

	// Recursive case: construct valid tipset from valid parent
	pBlks, err := c.chain.GetParents(ctx, ts)
	if err != nil {
		return nil, err
	}
	if len(pBlks) == 0 { // invalid genesis tipset
		return nil, ErrInvalidBase
	}
	st, err := c.State(ctx, pBlks)
	if err != nil {
		return nil, err
	}
	st, err = c.runMessages(ctx, st, ts)
	if err != nil {
		return nil, err
	}
	if err = c.flushAndCache(ctx, st, ts); err != nil {
		return nil, err
	}
	return st, nil
}

// runMessages applies the messages of all blocks within the input
// tipset to the input base state.  Messages are applied block by
// block with blocks sorted by their ticket bytes.  The output state must be
// flushed after calling to guarantee that the state transitions propagate.
//
// An error is returned if individual blocks contain messages that do not
// lead to successful state transitions.  An error is also returned if the node
// faults while running aggregate state computation.
func (c *Expected) runMessages(ctx context.Context, st state.Tree, ts types.TipSet) (state.Tree, error) {
	var cpySt state.Tree
	for _, blk := range ts {
		cpyCid, err := st.Flush(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		// state copied so changes don't propagate between block validations
		cpySt, err = state.LoadStateTree(ctx, c.store, cpyCid, builtin.Actors)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}

		receipts, err := core.ProcessBlock(ctx, blk, cpySt)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		// TODO: check that receipts actually match
		if len(receipts) != len(blk.MessageReceipts) {
			return nil, fmt.Errorf("found invalid message receipts: %v %v", receipts, blk.MessageReceipts)
		}

		outCid, err := cpySt.Flush(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		if !outCid.Equals(blk.StateRoot) {
			return nil, ErrStateRootMismatch
		}
	}
	if len(ts) == 1 { // block validation state == aggregate parent state
		return cpySt, nil
	}
	// multiblock tipsets require reapplying messages to get aggregate state
	// NOTE: It is possible to optimize further by applying block validation
	// in sorted order to reuse first block transitions as the starting state
	// for the tipSetProcessor.
	_, err := core.ProcessTipSet(ctx, ts, st)
	if err != nil {
		return nil, errors.Wrap(err, "error validating tipset")
	}
	return st, nil
}

// flushAndCache flushes and caches the input tipset's state. It also persists
// the tipset's blocks in the ChainManager's data store.
func (c *Expected) flushAndCache(ctx context.Context, st state.Tree, ts types.TipSet) error {
	for _, blk := range ts {
		if err := c.chain.Put(ctx, blk); err != nil {
			return errors.Wrap(err, "failed to store block")
		}
	}

	root, err := st.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush state")
	}

	c.stateCache[ts.String()] = root
	return nil
}

// StateForBlockIDs returns the state of the tipset consisting of the input blockIDs.
func (c *Expected) StateForBlockIDs(ctx context.Context, ids types.SortedCidSet) (state.Tree, error) {
	blks, err := c.chain.GetForIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(blks) == 0 { // no ids
		return nil, errors.New("cannot get state of tipset with no members")
	}
	return c.State(ctx, blks)
}

func (c *Expected) LatestState(ctx context.Context) (state.Tree, error) {
	return c.State(ctx, c.chain.Head().ToSlice())
}
