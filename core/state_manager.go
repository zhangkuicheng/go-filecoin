package core

import (
	"context"
	"fmt"
	"time"

	"gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"

	hamt "github.com/ipfs/go-hamt-ipld"
	dag "github.com/ipfs/go-ipfs/merkledag"
)

// StateManager manages the current state of the chain and handles validating
// and applying updates.
type StateManager struct {
	bestBlock *Block
	headCid   *cid.Cid

	stateRoot *State

	txPool *TransactionPool

	knownGoodBlocks *cid.Set

	cs  *hamt.CborIpldStore
	dag dag.DAGService

	miner *Miner
}

// Inform informs the state manager that we received a new block from the given
// peer
func (s *StateManager) Inform(p peer.ID, blk *Block) {
	if err := s.processNewBlock(context.Background(), blk); err != nil {
		log.Error(err)
		return
	}
	s.miner.newBlocks <- blk
}

func (s *StateManager) GetStateRoot() *State {
	// TODO: maybe return immutable copy or something? Don't necessarily want
	// the caller to be able to mutate this without them intending to
	return s.stateRoot
}

func (s *StateManager) processNewBlock(ctx context.Context, blk *Block) error {
	if err := s.validateBlock(ctx, blk); err != nil {
		return fmt.Errorf("validate block failed: %s", err)
	}

	if blk.Score() > s.bestBlock.Score() {
		return s.acceptNewBlock(blk)
	}

	return fmt.Errorf("new block not better than current block (%d <= %d)",
		blk.Score(), s.bestBlock.Score())
}

// acceptNewBlock sets the given block as our current 'best chain' block
func (s *StateManager) acceptNewBlock(blk *Block) error {
	_, err := s.dag.Add(blk.ToNode())
	if err != nil {
		return fmt.Errorf("failed to put block to disk: %s", err)
	}

	// update our accounting of the 'best block'
	s.knownGoodBlocks.Add(blk.Cid())
	s.bestBlock = blk
	s.headCid = blk.Cid()

	// Remove any transactions that were mined in the new block from the mempool
	// TODO: actually go through transactions for each block back to the last
	// common block and remove transactions/re-add transactions in blocks we
	// had but arent in the new chain
	for _, tx := range blk.Txs {
		c, err := tx.Cid()
		if err != nil {
			return err
		}

		s.txPool.ClearTx(c)
	}

	st, err := LoadState(context.Background(), s.cs, blk.StateRoot)
	if err != nil {
		return fmt.Errorf("failed to get newly approved state: %s", err)
	}
	s.stateRoot = st

	fmt.Printf("accepted new block, [s=%d, h=%s, st=%s]\n", blk.Score(), blk.Cid(), blk.StateRoot)
	return nil
}

func (s *StateManager) fetchBlock(ctx context.Context, c *cid.Cid) (*Block, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var blk Block
	if err := s.cs.Get(ctx, c, &blk); err != nil {
		return nil, err
	}

	return &blk, nil
}

// checkSingleBlock verifies that this block, on its own, is structurally and
// cryptographically valid. This means checking that all of its fields are
// properly filled out and its signature is correct. Checking the validity of
// state changes must be done separately and only once the state of the
// previous block has been validated.
func (s *StateManager) checkBlockValid(ctx context.Context, b *Block) error {
	return nil
}

func (s *StateManager) checkBlockStateChangeValid(ctx context.Context, st *State, b *Block) error {
	if err := st.ApplyTransactions(ctx, b.Txs); err != nil {
		return err
	}

	c, err := st.Flush(ctx)
	if err != nil {
		return err
	}

	if !c.Equals(b.StateRoot) {
		return fmt.Errorf("state root failed to validate! (%s != %s)", c, b.StateRoot)
	}

	return nil
}

// TODO: this method really needs to be thought through carefully. Probably one
// of the most complicated bits of the system
func (s *StateManager) validateBlock(ctx context.Context, b *Block) error {
	if err := s.checkBlockValid(ctx, b); err != nil {
		return fmt.Errorf("check block valid failed: %s", err)
	}

	if b.Score() <= s.bestBlock.Score() {
		// TODO: likely should still validate this chain and keep it around.
		// Someone else could mine on top of it
		return fmt.Errorf("new block is not better than our current block")
	}

	var validating []*Block
	baseBlk := b
	for !s.knownGoodBlocks.Has(baseBlk.Cid()) { // probably should be some sort of limit here
		validating = append(validating, baseBlk)

		next, err := s.fetchBlock(ctx, baseBlk.Parent)
		if err != nil {
			return fmt.Errorf("fetch block failed: %s", err)
		}

		if err := s.checkBlockValid(ctx, next); err != nil {
			return err
		}

		baseBlk = next
	}

	st, err := LoadState(ctx, s.cs, baseBlk.StateRoot)
	if err != nil {
		return fmt.Errorf("load state failed: %s", err)
	}

	for i := len(validating) - 1; i >= 0; i-- {
		if err := s.checkBlockStateChangeValid(ctx, st, validating[i]); err != nil {
			return err
		}
		s.knownGoodBlocks.Add(validating[i].Cid())
	}

	return nil
}
