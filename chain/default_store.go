package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gx/ipfs/QmSkuaNgyGmV8c1L3cZNWcUxRJV6J3nsD96JVQPcWcwtyW/go-hamt-ipld"
	bserv "gx/ipfs/QmUSuYd5Q1N291DH679AVvHwGLwtS1V9VPDWvnUN9nGJPT/go-blockservice"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmWdao8WJqYU65ZbYQyQWMFqku6QFxkPiv8HSUAkXdHZoe/go-ipfs-exchange-offline"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	bstore "gx/ipfs/QmcD7SqfyQyA91TZUQ7VPRYbGarxmY7EsQewVYMuN5LNSv/go-ipfs-blockstore"
	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"
	"gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("chain")

var headKey = datastore.NewKey("/chain/heaviestTipSet")

// DefaultStore is a generic implementation of the Store interface.
// It works(tm) for now.
type DefaultStore struct {
	// privateStore is the on disk storage for blocks.  This is private to
	// the store.  The filecoin node does not keep a reference to this.
	privateStore *hamt.CborIpldStore
	//stateStore is the on disk storage used for loading states.  This is
	// shared with the rest of the filecoin node.
	stateStore *hamt.CborIpldStore
	// head is the tipset at the head of the best known chain.
	head consensus.TipSet
	// Protects head and genesisCid.
	mu sync.Mutex
	// genesis is the CID of the genesis block.
	genesis *cid.Cid
	// headEvents is a pubsub channel that publishes an event every time the head changes.
	// We operate under the assumption that tipsets published to this channel
	// will always be queued and delivered to subscribers in the order discovered.
	// Successive published tipsets may be supersets of previously published tipsets.
	headEvents *pubsub.PubSub

	// ds is the datastore the hold meta information about the chain, like the current head.
	ds repo.Datastore

	// Tracks tipsets by height/parentset for use by expected consensus.
	tipIndex *TipIndex

	// Caches blocks in the store.  TODO: limit cache size: eviction policy etc.
	blockCache map[string]*types.Block
}

// Ensure DefaultStore satisfies the Store interface at compile time.
var _ Store = (*DefaultStore)(nil)

// NewDefaultStore constructs a new default store.
func NewDefaultStore(ds repo.Datastore, stateStore *hamt.CborIpldStore, genesisCid *cid.Cid) Store {
	bs := bstore.NewBlockstore(ds)
	priv := hamt.CborIpldStore{Blocks: bserv.New(bs, offline.Exchange(bs))}
	return &DefaultStore{
		privateStore: &priv,
		stateStore:   stateStore,
		headEvents:   pubsub.New(128),
		ds:           ds,
		tipIndex:     NewTipIndex(),
		blockCache:   make(map[string]*types.Block),
		genesis:      genesisCid,
	}
}

// Load rebuilds the DefaultStore's caches by traversing backwards from the
// most recent best head as stored in its datastore.  Load will error if the
// head does not link back to the expected genesis block, or the Store's
// privateStore does not store a link in the chain.
func (store *DefaultStore) Load(ctx context.Context) error {
	tipCids, err := store.loadHead()
	if err != nil {
		return err
	}
	headTs := consensus.TipSet{}
	// traverse starting from head to begin loading the chain
	for it := tipCids.Iter(); !it.Complete(); it.Next() {
		blk, err := store.GetBlock(ctx, it.Value())
		if err != nil {
			return errors.Wrap(err, "failed to load block in head TipSet")
		}
		err = headTs.AddBlock(blk)
		if err != nil {
			return errors.Wrap(err, "failed to add validated block to TipSet")
		}
	}

	var genesii consensus.TipSet
	err = store.walkChain(ctx, headTs.ToSlice(), func(tips []*types.Block) (cont bool, err error) {
		ts, err := consensus.NewTipSet(tips...)
		if err != nil {
			return false, err
		}
		stateRoot, err := store.loadStateRoot(ts)
		if err != nil {
			return false, err
		}
		err = store.PutTipSetAndState(ctx, &TipSetAndState{
			TipSet:          ts,
			TipSetStateRoot: stateRoot,
		})
		if err != nil {
			return false, err
		}
		// TODO: populate block cache here.
		genesii = ts
		return true, nil
	})
	if err != nil {
		return err
	}
	// Check genesis here.
	if len(genesii) != 1 {
		return errors.Errorf("genesis tip set must be a single block, got %d blocks", len(genesii))
	}

	loadCid := genesii.ToSlice()[0].Cid()
	if !loadCid.Equals(store.genesis) {
		return errors.Errorf("expected genesis cid: %s, loaded genesis cid: %s", store.genesis, loadCid)
	}

	// Set actual head.
	return store.SetHead(ctx, headTs)
}

// loadHead loads the latest known head from disk.
func (store *DefaultStore) loadHead() (types.SortedCidSet, error) {
	var emptyCidSet types.SortedCidSet
	bbi, err := store.ds.Get(headKey)
	if err != nil {
		return emptyCidSet, errors.Wrap(err, "failed to read headKey")
	}
	bb, ok := bbi.([]byte)
	if !ok {
		return emptyCidSet, fmt.Errorf("stored headCids not []byte")
	}

	var cids types.SortedCidSet
	err = json.Unmarshal(bb, &cids)
	if err != nil {
		return emptyCidSet, errors.Wrap(err, "failed to cast headCids")
	}

	return cids, nil
}

func (store *DefaultStore) loadStateRoot(ts consensus.TipSet) (*cid.Cid, error) {
	key := datastore.NewKey(ts.String())
	bbi, err := store.ds.Get(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read tipset key %s", ts.String())
	}
	bb, ok := bbi.([]byte)
	if !ok {
		return nil, fmt.Errorf("stored tipset state root not []byte")
	}

	var stateRoot cid.Cid
	err = json.Unmarshal(bb, &stateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to cast state root of tipset %s", ts.String())
	}
	return &stateRoot, nil
}

// putBlk persists a block to disk.
func (store *DefaultStore) putBlk(ctx context.Context, block *types.Block) error {
	if _, err := store.privateStore.Put(ctx, block); err != nil {
		return errors.Wrap(err, "failed to put block")
	}
	return nil
}

// PutTipSetAndState persists the blocks of a tipset and the tipset index.
func (store *DefaultStore) PutTipSetAndState(ctx context.Context, tsas *TipSetAndState) error {
	// Persist blocks.
	for _, blk := range tsas.TipSet {
		if err := store.putBlk(ctx, blk); err != nil {
			return err
		}
	}

	// Update tipindex.
	err := store.tipIndex.Put(tsas)
	if err != nil {
		return err
	}
	// Persist the state mapping.
	if err = store.writeTipSetAndState(tsas); err != nil {
		return err
	}

	return nil
}

// GetTipSetAndState returns the tipset and state of the tipset whose block
// cids correspond to the input string.
func (store *DefaultStore) GetTipSetAndState(ctx context.Context, tsKey string) (*TipSetAndState, error) {
	return store.tipIndex.Get(tsKey)
}

// HasTipSetAndState returns true iff the default store's tipindex is indexing
// the tipset referenced in the input key.
func (store *DefaultStore) HasTipSetAndState(ctx context.Context, tsKey string) bool {
	return store.tipIndex.Has(tsKey)
}

// GetTipSetAndStatesByParents returns the the tipsets and states tracked by
// the default store's tipIndex that have the parent set corresponding to the
// input key.
func (store *DefaultStore) GetTipSetAndStatesByParents(ctx context.Context, pTsKey string) ([]*TipSetAndState, error) {
	return store.tipIndex.GetByParents(pTsKey)
}

// HasTipSetAndStatesWithParents returns true if the default store's tipindex
// contains any tipset indexed by the provided parent ID.
func (store *DefaultStore) HasTipSetAndStatesWithParents(ctx context.Context, pTsKey string) bool {
	return store.tipIndex.HasByParents(pTsKey)
}

// GetBlocks retrieves the blocks referenced in the input cid set.
func (store *DefaultStore) GetBlocks(ctx context.Context, ids types.SortedCidSet) ([]*types.Block, error) {
	var blks []*types.Block
	for it := ids.Iter(); !it.Complete(); it.Next() {
		id := it.Value()
		blk, err := store.GetBlock(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "error fetching block")
		}
		blks = append(blks, blk)
	}
	return blks, nil

}

// GetBlock retrieves a block by cid.
func (store *DefaultStore) GetBlock(ctx context.Context, c *cid.Cid) (*types.Block, error) {
	var blk types.Block
	if err := store.privateStore.Get(ctx, c, &blk); err != nil {
		return nil, errors.Wrapf(err, "failed to get block %s", c.String())
	}
	return &blk, nil
}

// HasAllBlocks indicates whether the blocks are in the store.
func (store *DefaultStore) HasAllBlocks(ctx context.Context, cids []*cid.Cid) bool {
	for _, c := range cids {
		if !store.HasBlock(ctx, c) {
			return false
		}
	}
	return true
}

// HasBlock indicates whether the block is in the store.
func (store *DefaultStore) HasBlock(ctx context.Context, c *cid.Cid) bool {
	// TODO: consider adding Has method to HamtIpldCborstore if this used much,
	// or using a different store interface for quick Has.
	blk, err := store.GetBlock(ctx, c)

	return blk != nil && err == nil
}

// HeadEvents returns a pubsub interface the pushes events each time the
// default store's head is reset.
func (store *DefaultStore) HeadEvents() *pubsub.PubSub {
	return store.headEvents
}

// SetHead sets the passed in tipset as the new head of this chain.
func (store *DefaultStore) SetHead(ctx context.Context, ts consensus.TipSet) error {
	log.Infof("SetHead %s", ts.String())

	store.mu.Lock()
	defer store.mu.Unlock()

	// Ensure consistency by storing this new head on disk.
	if err := store.writeHead(ctx, ts.ToSortedCidSet()); err != nil {
		return errors.Wrap(err, "failed to write new Head to datastore")
	}

	store.head = ts

	// Publish an event that we have a new head.
	store.HeadEvents().Pub(ts, NewHeadTopic)

	return nil
}

// writeHead writes the given cid set as head to disk.
func (store *DefaultStore) writeHead(ctx context.Context, cids types.SortedCidSet) error {
	log.Infof("writeHead %s", cids.String())
	val, err := json.Marshal(cids)
	if err != nil {
		return err
	}

	return store.ds.Put(headKey, val)
}

// writeTipSetAndState writes the tipset key and the state root id to the
// datastore.
func (store *DefaultStore) writeTipSetAndState(tsas *TipSetAndState) error {
	val, err := json.Marshal(tsas.TipSetStateRoot)
	if err != nil {
		return err
	}

	// datastore keeps tsKey:stateRoot (k,v) pairs.
	key := datastore.NewKey(tsas.TipSet.String())
	return store.ds.Put(key, val)
}

// Head returns the current head.
func (store *DefaultStore) Head() consensus.TipSet {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.head
}

// LatestState returns the state associated with the latest chain head.
func (store *DefaultStore) LatestState(ctx context.Context) (state.Tree, error) {
	h := store.Head()
	if h == nil {
		return nil, errors.New("Unset head")
	}
	tsas, err := store.GetTipSetAndState(ctx, h.String())
	if err != nil {
		return nil, err
	}
	return state.LoadStateTree(ctx, store.stateStore, tsas.TipSetStateRoot, builtin.Actors)
}

// BlockHistory returns a channel of block pointers (or errors), starting with the current best tipset's blocks
// followed by each subsequent parent and ending with the genesis block, after which the channel
// is closed. If an error is encountered while fetching a block, the error is sent, and the channel is closed.
func (store *DefaultStore) BlockHistory(ctx context.Context) <-chan interface{} {
	ctx = log.Start(ctx, "Chain.Store.BlockHistory")
	out := make(chan interface{})
	tips := store.Head().ToSlice()

	go func() {
		defer close(out)
		defer log.Finish(ctx)
		err := store.walkChain(ctx, tips, func(tips []*types.Block) (cont bool, err error) {
			var raw interface{}
			raw, err = consensus.NewTipSet(tips...)
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

// walkChain walks backward through the chain, starting at tips, invoking cb() at each height.
func (store *DefaultStore) walkChain(ctx context.Context, tips []*types.Block, cb func(tips []*types.Block) (cont bool, err error)) error {
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
			p, err := store.GetBlock(ctx, pid)
			if err != nil {
				return errors.Wrap(err, "error retrieving block from store")
			}
			tips = append(tips, p)
		}
	}

	return nil
}

// GenesisCid returns the genesis cid of the chain tracked by the default store.
func (store *DefaultStore) GenesisCid() *cid.Cid {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.genesis
}

// Stop stops all activities and cleans up.
func (store *DefaultStore) Stop() {
	store.headEvents.Shutdown()
}
