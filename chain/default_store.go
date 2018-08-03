package chain

import (
	"context"
	"fmt"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("chain.default_store")

// DefaultStore is a generic implementation of the Store interface.
// It works(tm) for now.
type DefaultStore struct {
	// blockstore is the raw on disk storage for blocks.
	blockstore *hamt.CborIpldStore
	// head is the set of blocks at the head of the best known chain.
	head struct {
		sync.Mutex
		ts core.TipSet
	}
	// genesis is the CID of the genesis block.
	genesis *cid.Cid
	// headEvents is a pubsub channel that publishes an event every time the head changes.
	// We operate under the assumption that tipsets published to this channel
	// will always be queued and delivered to subscribers in the order discovered.
	// Successive published tipsets may be supersets of previously published tipsets.
	headEvents *pubsub.PubSub
}

// Ensure DefaultStore satisfies the Store interface at compile time.
var _ Store = (*DefaultStore)(nil)

// NewDefaultStore constructs a new default store.
func NewDefaultStore(blockstore *hamt.CborIpldStore) Store {
	return DefaultStore{
		blockstore: blockstore,
		headEvents: pubsub.New(128),
	}
}

func (store *DefaultStore) HeadEvents() *pubsub.PubSub {
	return store.headEvents
}

// Put persists a block to disk.
func (store *DefaultStore) Put(ctx context.Context, block *types.Block) error {
	if _, err := store.blockstore.Put(ctx, block); err != nil {
		return errors.Wrap(err, "failed to put block")
	}
	return nil
}

// Get retrieves a block by cid.
func (store *DefaultStore) Get(ctx context.Context, c *cid.Cid) (types.Block, error) {
	var blk types.Block
	if err := store.blockstore.Get(ctx, c, &blk); err != nil {
		return nil, errors.Wrap(err, "failed to get block")
	}

	return blk, nil
}

// Has indicates whether the block is in the store.
func (store *DefaultStore) Has(ctx context.Context, c *cid.Cid) bool {
	// TODO: add Has method to HamtIpldCborstore if this used much
	blk, err := store.Get(ctx, c)

	return blk != nil && err == nil
}

func (store *DefaultStore) SetHead(ts core.TipSet) error {
	store.head.Lock()
	defer store.head.Unlock()

	// The heaviest tipset should not pick up changes from adding new blocks to the index.
	// It only changes explicitly when set through this function.
	store.head = ts.Clone()

	// If there is no genesis block set yet, this means we have our genesis block here.
	if store.genesis == nil {
		if len(ts) != 1 {
			return errors.Errorf("genesis tip set must be a single block, got %d blocks", len(ts))
		}

		// we know this will only be one iteration
		for c := range ts {
			store.genesis = c
		}
	}

	return nil
}

func (store *DefaultStore) Head() core.TipSet {
	store.head.Lock()
	defer store.head.Unlock()

	store.head.ts
}

// BlockHistory returns a channel of block pointers (or errors), starting with the current best tipset's blocks
// followed by each subsequent parent and ending with the genesis block, after which the channel
// is closed. If an error is encountered while fetching a block, the error is sent, and the channel is closed.
func (store *DefaultStore) BlockHistory(ctx context.Context) <-chan interface{} {
	out := make(chan interface{})
	tips := store.Head().ToSlice()

	go func() {
		defer close(out)
		err := store.WalkChain(tips, func(tips []*types.Block) (cont bool, err error) {
			var raw interface{}
			raw, err = core.NewTipSet(tips...)
			if err != nil {
				raw = err
			}
			select {
			case <-ctx.Done():
				return false, nil
			case out <- raw:
			}
			return true, nil
		})
		if err != nil {
			select {
			case <-ctx.Done():
			case out <- err:
			}
		}
	}()
	return out
}

// WaitForMessage searches for a message with Cid, msgCid, then passes it, along with the containing Block and any
// MessageRecipt, to the supplied callback, cb. If an error is encountered, it is returned. Note that it is logically
// possible that an error is returned and the success callback is called. In that case, the error can be safely ignored.
// TODO: This implementation will become prohibitively expensive since it involves traversing the entire blockchain.
//       We should replace with an index later.
func (store *DefaultStore) WaitForMessage(ctx context.Context, msgCid *cid.Cid, cb func(*types.Block, *types.SignedMessage, *types.MessageReceipt) error) error {
	ctx = log.Start(ctx, "WaitForMessage")
	log.Info("Calling WaitForMessage")
	// Ch will contain a stream of blocks to check for message (or errors).
	// Blocks are either in new heaviest tipsets, or next oldest historical blocks.
	ch := make(chan (interface{}))

	// New blocks
	newHeadCh := store.HeadEvents().Sub(NewHeadTopic)
	defer store.HeadEvents.Unsub(newHeadCh, NewHeadTopic)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Historical blocks
	historyCh := store.BlockHistory(ctx)

	// Merge historical and new block Channels.
	go func() {
		// TODO: accommodate a new chain being added, as opposed to just a single block.
		for raw := range newHeadCh {
			ch <- raw
		}
	}()
	go func() {
		// TODO make history serve up tipsets
		for raw := range historyCh {
			ch <- raw
		}
	}()

	for raw := range ch {
		switch ts := raw.(type) {
		case error:
			log.Errorf("WaitForMessage: %s", ts)
			return ts
		case TipSet:
			for _, blk := range ts {
				for _, msg := range blk.Messages {
					c, err := msg.Cid()
					if err != nil {
						log.Errorf("WaitForMessage: %s", err)
						return err
					}
					if c.Equals(msgCid) {
						recpt, err := store.receiptFromTipSet(ctx, msgCid, ts)
						if err != nil {
							return errors.Wrap(err, "error retrieving receipt from tipset")
						}
						return cb(blk, msg, recpt)
					}
				}
			}
		}
	}

	return retErr
}

// receiptFromTipSet finds the receipt for the message with msgCid in the input
// input tipset.  This can differ from the message's receipt as stored in its
// parent block in the case that the message is in conflict with another
// message of the tipset.
// TODO: find a better home for this method
func (store *DefaultStore) receiptFromTipSet(ctx context.Context, msgCid *cid.Cid, ts core.TipSet) (*types.MessageReceipt, error) {
	// Receipts always match block if tipset has only 1 member.
	var rcpt *types.MessageReceipt
	blks := ts.ToSlice()
	if len(ts) == 1 {
		b := blks[0]
		// TODO: this should return an error if a receipt doesn't exist.
		// Right now doing so breaks tests because our test helpers
		// don't correctly apply messages when making test chains.
		j, err := msgIndexOfTipSet(msgCid, ts, types.SortedCidSet{})
		if err != nil {
			return nil, err
		}
		if j < len(b.MessageReceipts) {
			rcpt = b.MessageReceipts[j]
		}
		return rcpt, nil
	}

	// Apply all the tipset's messages to determine the correct receipts.
	ids, err := ts.Parents()
	if err != nil {
		return nil, err
	}
	st, err := cm.stateForBlockIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res, err := cm.tipSetProcessor(ctx, ts, st)
	if err != nil {
		return nil, err
	}

	// If this is a failing conflict message there is no application receipt.
	if res.Failures.Has(msgCid) {
		return nil, nil
	}

	j, err := msgIndexOfTipSet(msgCid, ts, res.Failures)
	if err != nil {
		return nil, err
	}
	// TODO: and of bounds receipt index should return an error.
	if j < len(res.Results) {
		rcpt = res.Results[j].Receipt
	}
	return rcpt, nil
}

// msgIndexOfTipSet returns the order in which msgCid apperas in the canonical
// message ordering of the given tipset, or an error if it is not in the
// tipset.
// TODO: find a better home for this method
func msgIndexOfTipSet(msgCid *cid.Cid, ts core.TipSet, fails types.SortedCidSet) (int, error) {
	blks := ts.ToSlice()
	types.SortBlocks(blks)
	var duplicates types.SortedCidSet
	var msgCnt int
	for _, b := range blks {
		for _, msg := range b.Messages {
			c, err := msg.Cid()
			if err != nil {
				return -1, err
			}
			if fails.Has(c) {
				continue
			}
			if duplicates.Has(c) {
				continue
			}
			(&duplicates).Add(c)
			if c.Equals(msgCid) {
				return msgCnt, nil
			}
			msgCnt++
		}
	}

	return -1, fmt.Errorf("message cid %s not in tipset", msgCid.String())
}

func (store *DefaultStore) GetTipSetByBlock(blk *types.Block) (core.TipSet, error) {}

func (store *DefaultStore) GetTipSetsByHeight(height uint64) []core.TipSet {}

func (store *DefaultStore) WalkChain(tips []*types.Block, cb func(tips []*types.Block) (cont bool, err error)) error {
}

func (store *DefaultStore) GenesisCid() *cid.Cid {
	// TODO: think about locking
	store.genesis
}
