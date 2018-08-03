package state

import (
	"context"
	"fmt"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	cid "gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
	hamt "github.com/ipfs/go-hamt-ipld"
	"github.com/protocol/coinlist-server/chain"
)

type DefaultProcessor struct {
	consensus consensus.Algorithm
	store     *hamt.CborIpldStore
	chain     chain.Chain

	// Tracks state by tipset identifier
	stateCache map[string]*cid.Cid
}

// Ensure Default satisfies the Processor interface at compile time.
var _ DefaultProcessor = (*Processor)(nil)

func NewDefaultProcessor(consensus consensus.Algorithm, store *hamt.CborIpldStore, chain chain.Chain) Processor {
	return DefaultProcessor{
		consensus:  consensus,
		store:      store,
		chain:      chain,
		stateCache: make(map[string]*cid.Cid),
	}
}

// State returns the aggregate state tree for the blocks or an error if the
// blocks are not a valid tipset or are not part of a valid chain.
func (p *DefaultProcessor) State(ctx context.Context, blks []*types.Block) (Tree, error) {
	ts, err := p.consensus.NewValidTipSet(ctx, blks)
	if err != nil {
		return nil, errors.Wrapf(err, "blks do not form a valid tipset: %s", pp.StringFromBlocks(blks))
	}

	// Return cache hit
	if root, ok := p.stateCache[ts.String()]; ok { // tipset in cache
		return LoadStateTree(ctx, p.store, root, builtin.Actors)
	}

	// Base case is the genesis block
	if len(ts) == 1 && blks[0].Cid().Equals(p.chain.GenesisCid()) { // genesis tipset
		return LoadStateTree(ctx, p.store, blks[0].StateRoot, builtin.Actors)
	}

	// Recursive case: construct valid tipset from valid parent
	pBlks, err := p.chain.FetchParents(ctx, ts)
	if err != nil {
		return nil, err
	}
	if len(pBlks) == 0 { // invalid genesis tipset
		return nil, ErrInvalidBase
	}
	st, err := p.State(ctx, pBlks)
	if err != nil {
		return nil, err
	}
	st, err = p.runMessages(ctx, st, ts)
	if err != nil {
		return nil, err
	}
	if err = p.flushAndCache(ctx, st, ts); err != nil {
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
func (p *DefaultProcessor) runMessages(ctx context.Context, st Tree, ts TipSet) (Tree, error) {
	var cpySt Tree
	for _, blk := range ts {
		cpyCid, err := st.Flush(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		// state copied so changes don't propagate between block validations
		cpySt, err = LoadStateTree(ctx, p.store, cpyCid, builtin.Actors)
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
func (p *DefaultProcessor) flushAndCache(ctx context.Context, st statetree.Tree, ts TipSet) error {
	for _, blk := range ts {
		if err := p.chain.Put(ctx, blk); err != nil {
			return errors.Wrap(err, "failed to store block")
		}
	}

	root, err := st.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush state")
	}

	p.stateCache[ts.String()] = root
	return nil
}

// StateForBlockIDs returns the state of the tipset consisting of the input blockIDs.
func (p *DefaultProcessor) StateForBlockIDs(ctx context.Context, ids types.SortedCidSet) (Tree, error) {
	blks, err := p.chain.GetForIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(blks) == 0 { // no ids
		return nil, errors.New("cannot get state of tipset with no members")
	}
	return p.State(ctx, blks)
}
