package core

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	logging "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log"
	"gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	"gx/ipfs/QmdbxjQWogRCHRaxhhGnYdT1oQJzL9GdqSKzCdqWr85AP2/pubsub"
	"gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"

	statetree "github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("chain")

var (
	// ErrStateRootMismatch is returned when the computed state root doesn't match the expected result.
	ErrStateRootMismatch = errors.New("blocks state root does not match computed result")
	// ErrInvalidBase is returned when the chain doesn't connect back to a known good block.
	ErrInvalidBase = errors.New("block does not connect to a known good chain")
	// ErrDifferentGenesis is returned when processing a chain with a different genesis block.
	ErrDifferentGenesis = fmt.Errorf("chain had different genesis")
	// ErrBadTipSet is returned when processing a tipset containing blocks of different heights or different parent sets
	ErrBadTipSet = errors.New("tipset contains blocks of different heights or different parent sets")
	// ErrUninit is returned when the chain manager is called to process a block but does not have a genesis block
	ErrUninit = errors.New("the chain manager cannot process blocks without a genesis block")
)

var heaviestTipSetKey = datastore.NewKey("/chain/heaviestTipSet")

// HeaviestTipSetTopic is the topic used to publish new best tipsets.
const HeaviestTipSetTopic = "heaviest-tipset"

// BlockProcessResult signifies the outcome of processing a given block.
type BlockProcessResult int

const (
	// Unknown implies there was an error that made it impossible to process the block.
	Unknown = BlockProcessResult(iota)

	// ChainAccepted implies the chain was valid, and is now our current best
	// chain.
	ChainAccepted

	// ChainValid implies the chain was valid, but not better than our current
	// best chain.
	ChainValid

	// InvalidBase implies the chain does not connect back to any previously
	// known good block.
	InvalidBase

	// Uninit implies that the chain manager does not have a genesis block
	// and therefore cannot process new blocks.
	Uninit
)

func (bpr BlockProcessResult) String() string {
	switch bpr {
	case ChainAccepted:
		return "accepted"
	case ChainValid:
		return "valid"
	case Unknown:
		return "unknown"
	case InvalidBase:
		return "invalid"
	}
	return "" // never hit
}

// ChainManager manages the current state of the chain and handles validating
// and applying updates.
// Safe for concurrent access
type ChainManager struct {
	// heaviestTipSet is the set of blocks at the head of the best known chain
	heaviestTipSet struct {
		sync.Mutex
		ts TipSet
	}

	blockProcessor  Processor
	tipSetProcessor TipSetProcessor

	// genesisCid holds the cid of the chains genesis block for later access
	genesisCid *cid.Cid

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

	// Tracks state by tipset identifier
	stateCache map[string]*cid.Cid

	cstore *hamt.CborIpldStore

	ds datastore.Datastore

	// PwrTableView provides miner and total power for the EC chain weight
	// computation.
	PwrTableView powerTableView

	// HeaviestTipSetPubSub is a pubsub channel that publishes all best tipsets.
	// We operate under the assumption that tipsets published to this channel
	// will always be queued and delivered to subscribers in the order discovered.
	// Successive published tipsets may be supersets of previously published tipsets.
	HeaviestTipSetPubSub *pubsub.PubSub

	FetchBlock        func(context.Context, *cid.Cid) (*types.Block, error)
	GetHeaviestTipSet func() TipSet
}

// NewChainManager creates a new filecoin chain manager.
func NewChainManager(ds datastore.Datastore, cs *hamt.CborIpldStore) *ChainManager {
	cm := &ChainManager{
		cstore:          cs,
		ds:              ds,
		blockProcessor:  ProcessBlock,
		tipSetProcessor: ProcessTipSet,
		knownGoodBlocks: cid.NewSet(),
		tips:            tipIndex{},
		stateCache:      make(map[string]*cid.Cid),

		PwrTableView:         &marketView{},
		HeaviestTipSetPubSub: pubsub.New(128),
	}
	cm.FetchBlock = cm.fetchBlock
	cm.GetHeaviestTipSet = cm.getHeaviestTipSet

	return cm
}

// maybeAcceptBlock attempts to accept blk if its score is greater than the current best block,
// otherwise returning ChainValid.
func (cm *ChainManager) maybeAcceptBlock(ctx context.Context, blk *types.Block) (BlockProcessResult, error) {
	// We have to hold the lock at this level to avoid TOCTOU problems
	// with the new heaviest tipset.
	log.LogKV(ctx, "maybeAcceptBlock", blk.Cid().String())
	cm.heaviestTipSet.Lock()
	defer cm.heaviestTipSet.Unlock()
	ts, err := cm.GetTipSetByBlock(blk)
	if err != nil {
		return Unknown, err
	}
	// Calculate weights of TipSets for comparison.
	heaviestWeight, err := cm.weight(ctx, cm.heaviestTipSet.ts)
	if err != nil {
		return Unknown, err
	}
	newWeight, err := cm.weight(ctx, ts)
	if err != nil {
		return Unknown, err
	}
	heaviestTicket, err := cm.heaviestTipSet.ts.MinTicket()
	if err != nil {
		return Unknown, err
	}
	newTicket, err := ts.MinTicket()
	if err != nil {
		return Unknown, err
	}
	if newWeight.Cmp(heaviestWeight) == -1 ||
		(newWeight.Cmp(heaviestWeight) == 0 &&
			// break ties by choosing tipset with smaller ticket
			bytes.Compare(newTicket, heaviestTicket) >= 0) {
		return ChainValid, nil
	}

	// set the given tipset as our current heaviest tipset
	if err := cm.setHeaviestTipSet(ctx, ts); err != nil {
		return Unknown, err
	}
	log.Infof("new heaviest tipset, [s=%s, hs=%s]", newWeight.RatString(), ts.String())
	log.LogKV(ctx, "maybeAcceptBlock", ts.String())
	return ChainAccepted, nil
}

// NewBlockProcessor is the signature for a function which processes a new block.
type NewBlockProcessor func(context.Context, *types.Block) (BlockProcessResult, error)

// ProcessNewBlock sends a new block to the chain manager. If the block is in a
// tipset heavier than our current heaviest, this tipset is accepted as our
// heaviest tipset. Otherwise an error is returned explaining why it was not accepted.
func (cm *ChainManager) ProcessNewBlock(ctx context.Context, blk *types.Block) (bpr BlockProcessResult, err error) {
	ctx = log.Start(ctx, "ChainManager.ProcessNewBlock")
	defer func() {
		log.SetTag(ctx, "result", bpr.String())
		log.FinishWithErr(ctx, err)
	}()
	log.Infof("processing block [s=%d, cid=%s]", blk.Score(), blk.Cid())
	if cm.genesisCid == nil {
		return Uninit, ErrUninit
	}

	switch _, err := cm.state(ctx, []*types.Block{blk}); err {
	default:
		return Unknown, errors.Wrap(err, "validate block failed")
	case ErrInvalidBase:
		return InvalidBase, ErrInvalidBase
	case nil:
		return cm.maybeAcceptBlock(ctx, blk)
	}
}

func (cm *ChainManager) addBlock(b *types.Block, id *cid.Cid) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.knownGoodBlocks.Add(id)
	if err := cm.tips.addBlock(b); err != nil {
		panic("Invalid block added to tipset.  Validation should have caught earlier")
	}
}

// AggregateStateTreeComputer is the signature for a function used to get the state of a tipset.
type AggregateStateTreeComputer func(context.Context, TipSet) (statetree.Tree, error)

// InformNewTipSet informs the chainmanager that we learned about a potentially
// new tipset from the given peer. It fetches that tipset's blocks and
// passes them to the block processor (which fetches the rest of the chain on
// demand). In the (near) future we will want a better protocol for
// synchronizing the blockchain and downloading it efficiently.
// TODO: sync logic should be decoupled and off in a separate worker. This
// method should not block
func (cm *ChainManager) InformNewTipSet(from peer.ID, cids []*cid.Cid, h uint64) {
	// Naive sync.
	// TODO: more dedicated sync protocols, like "getBlockHashes(range)"
	ctx := context.TODO()

	for _, c := range cids {
		blk, err := cm.FetchBlock(ctx, c)
		if err != nil {
			log.Error("failed to fetch block: ", err)
			return
		}
		_, err = cm.ProcessNewBlock(ctx, blk)
		if err != nil {
			log.Error("processing new block: ", err)
			return
		}
	}
}
