package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"
	"gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("chain.default_store")

var headKey = datastore.NewKey("/chain/heaviestTipSet")

// DefaultStore is a generic implementation of the Store interface.
// It works(tm) for now.
type DefaultStore struct {
	// blockstore is the raw on disk storage for blocks.
	blockstore *hamt.CborIpldStore
	// head is the set of blocks at the head of the best known chain.
	head struct {
		sync.Mutex
		ts types.TipSet
	}
	// genesis is the CID of the genesis block.
	genesis *cid.Cid
	// headEvents is a pubsub channel that publishes an event every time the head changes.
	// We operate under the assumption that tipsets published to this channel
	// will always be queued and delivered to subscribers in the order discovered.
	// Successive published tipsets may be supersets of previously published tipsets.
	headEvents *pubsub.PubSub
	// ds is the datastore the hold meta information about the chain, like the current head.
	ds datastore.Datastore
	// Protects knownGoodBlocks and tipsIndex.
	mu sync.Mutex

	// knownGoodBlocks is a cache of 'good blocks'. It is a cache to prevent us
	// from having to rescan parts of the blockchain when determining the
	// validity of a given chain.
	// In the future we will need a more sophisticated mechanism here.
	// TODO: this should probably be an LRU, needs more consideration.
	// For example, the genesis block should always be considered a "good" block.
	knownGoodBlocks *cid.Set

	// Tracks tipsets by height/parentset for use by expected consensus.
	tips tipIndex
}

// Ensure DefaultStore satisfies the Store interface at compile time.
var _ Store = (*DefaultStore)(nil)

// NewDefaultStore constructs a new default store.
func NewDefaultStore(blockstore *hamt.CborIpldStore, ds datastore.Datastore) Store {
	return DefaultStore{
		blockstore:      blockstore,
		headEvents:      pubsub.New(128),
		ds:              ds,
		knownGoodBlocks: cid.NewSet(),
		tips:            tipIndex{},
	}
}

func (store *DefaultStore) Load(ctx context.Context) error {
	tipCids, err := store.loadHead()
	if err != nil {
		return err
	}
	ts := types.TipSet{}
	// traverse starting from one TipSet to begin loading the chain
	for it := tipCids.Iter(); !it.Complete(); it.Next() {
		blk, err := cm.Get(ctx, it.Value())
		if err != nil {
			return errors.Wrap(err, "failed to load block in head TipSet")
		}
		err = ts.AddBlock(blk)
		if err != nil {
			return errors.Wrap(err, "failed to add validated block to TipSet")
		}
	}

	var genesii []*types.Block
	err = store.WalkChain(ts.ToSlice(), func(tips []*types.Block) (cont bool, err error) {
		for _, t := range tips {
			id := t.Cid()
			cm.addBlock(t, id)
		}
		genesii = tips
		return true, nil
	})
	if err != nil {
		return err
	}
	switch len(genesii) {
	case 1:
		// TODO: probably want to load the expected genesis block and assert it here?
		store.genesis = genesii[0].Cid()
		store.head.ts = ts
	case 0:
		panic("unreached")
	default:
		panic("invalid chain - more than one genesis block found")
	}

	return nil
}

// loadHead loads the latest known head from disk.
func (store *DefaultStore) loadHead() (types.SortedCidSet, error) {
	bbi, err := store.ds.Get(headKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read headKey")
	}
	bb, ok := bbi.([]byte)
	if !ok {
		return nil, fmt.Errorf("stored headCids not []byte")
	}

	var cids types.SortedCidSet
	err = json.Unmarshal(bb, &cids)
	if err != nil {
		return nil, errors.Wrap(err, "failed to cast headCids")
	}

	return cids, nil
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

func (store *DefaultStore) HeadEvents() *pubsub.PubSub {
	return store.headEvents
}

// SetHead sets the passed in tipset as the new head of this chain.
func (store *DefaultStore) SetHead(ts types.TipSet) error {
	log.LogKV(ctx, "SetHead", ts.String())

	store.head.Lock()
	defer store.head.Unlock()

	// Ensure consistency by storing this new head on disk.
	if err := store.writeHead(ctx, ts.ToSortedCidSet()); err != nil {
		return err
	}

	// Publish an event that we have a new head.
	store.HeadEvents.Pub(ts, NewHeadTopic)

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

// writeHead writes the given cid set as head to disk.
func (store *DefaultStore) writeHead(ctx context.Context, cids types.SortedCidSet) error {
	log.LogKV(ctx, "writeHEad", cids.String())
	val, err := json.Marshal(cids)
	if err != nil {
		return err
	}

	return store.ds.Put(headKey, val)
}

// Head returns the current head.
func (store *DefaultStore) Head() types.TipSet {
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

// GetTipSetByBlock returns the tipset associated with a given block by
// performing a lookup on its parent set. The tipset returned is a
// cloned shallow copy of the version stored in the index
func (store *DefaultStore) GetTipSetByBlock(blk *types.Block) (types.TipSet, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	ts, ok := store.tips[uint64(blk.Height)][keyForParentSet(blk.Parents)]
	if !ok {
		return TipSet{}, errors.New("block's tipset not indexed by chain_mgr")
	}
	return ts.Clone(), nil
}

// GetTipSetsByHeight returns all tipsets at the given height. Neither the returned
// slice nor its members will be mutated by the ChainManager once returned.
func (store *DefaultStore) GetTipSetsByHeight(height uint64) []types.TipSet {
	store.mu.Lock()
	defer store.mu.Unlock()

	tsbp, ok := store.tips[height]
	if ok {
		for _, ts := range tsbp {
			// Assumption here that the blocks contained in `ts` are never mutated.
			tips = append(tips, ts.Clone())
		}
	}
	return tips
}

// WalkChain walks backward through the chain, starting at tips, invoking cb() at each height.
func (store *DefaultStore) WalkChain(ctx context.Context, tips []*types.Block, cb func(tips []*types.Block) (cont bool, err error)) error {
	for {
		cont, err := cb(tips)
		if err != nil {
			return errors.Wrap(err, "error processing block")
		}
		if !cont {
			return nil
		}
		ids := tips[0].Parents
		if ids.Empty() {
			break
		}

		tips = tips[:0]
		for it := ids.Iter(); !it.Complete(); it.Next() {
			pid := it.Value()
			p, err := store.Get(ctx, pid)
			if err != nil {
				return errors.Wrap(err, "error fetching block")
			}
			tips = append(tips, p)
		}
	}

	return nil
}

func (store *DefaultStore) GenesisCid() *cid.Cid {
	// TODO: think about locking
	store.genesis
}

// GetForIDs returns the blocks in the input cid set.
func (store *DefaultStore) GetForIDs(ctx context.Context, ids types.SortedCidSet) ([]*types.Block, error) {
	var pBlks []*types.Block
	for it := ids.Iter(); !it.Complete(); it.Next() {
		pid := it.Value()
		p, err := store.Get(ctx, pid)
		if err != nil {
			return nil, errors.Wrap(err, "error fetching block")
		}
		pBlks = append(pBlks, p)
	}
	return pBlks, nil
}

// Stop stops all activities and cleans up.
func (store *DefaultStore) Stop() {
	store.headEvents.Shutdown()
}
