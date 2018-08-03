package chain

import (
	"context"

	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/types"
)

// NewHeadTopic is the topic used to publish new heads.
const NewHeadTopic = "new-head"

type Store interface {
	// Put persists a block to disk.
	Put(ctx context.Context, block *types.Block) error
	// Get gets a block by cid. In the future there is caching here.
	Get(ctx context.Context, c *cid.Cid) (types.Block, error)
	// Has indicates whether the block is in the store.
	Has(ctx context.Context, c *cid.Cid) bool

	HeadEvents() *pubsub.PubSub
	SetHead(s core.TipSet) error
	Head() core.TipSet

	BlockHistory(ctx context.Context) <-chan interface{}
	WaitForMessage(ctx context.Context, msgCid *cid.Cid, cb func(*types.Block, *types.SignedMessage, *types.MessageReceipt) error) error
	GetTipSetByBlock(blk *types.Block) (core.TipSet, error)
	GetTipSetsByHeight(height uint64) []core.TipSet
	WalkChain(tips []*types.Block, cb func(tips []*types.Block) (cont bool, err error)) error
	GenesisCid() *cid.Cid
}
