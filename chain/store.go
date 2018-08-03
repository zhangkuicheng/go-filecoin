package chain

import (
	"context"

	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"

	"github.com/filecoin-project/go-filecoin/types"
)

// NewHeadTopic is the topic used to publish new heads.
const NewHeadTopic = "new-head"

type Store interface {
	// Load loads the chain from disk.
	Load(ctx context.Context) error
	// Stop stops all activities and cleans up.
	Stop()

	// Put persists a block to disk.
	Put(ctx context.Context, block *types.Block) error
	// Get gets a block by cid. In the future there is caching here.
	Get(ctx context.Context, c *cid.Cid) (*types.Block, error)
	// Has indicates whether the block is in the store.
	Has(ctx context.Context, c *cid.Cid) bool
	// GetForIDs returns the blocks in the input cid set.
	GetForIDs(ctx context.Context, ids types.SortedCidSet) ([]*types.Block, error)
	GetParents(ctx context.Context, ts types.TipSet) ([]*types.Block, error)

	HeadEvents() *pubsub.PubSub
	SetHead(ctx context.Context, s types.TipSet) error
	Head() types.TipSet

	BlockHistory(ctx context.Context) <-chan interface{}
	GetTipSetByBlock(blk *types.Block) (types.TipSet, error)
	GetTipSetsByHeight(height uint64) []types.TipSet
	WalkChain(ctx context.Context, tips []*types.Block, cb func(tips []*types.Block) (cont bool, err error)) error
	GenesisCid() *cid.Cid
}
