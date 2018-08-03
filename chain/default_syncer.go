package chain

import (
	"context"

	"gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

type DefaultSyncer struct {
	// blockstore is the raw on storage for blocks (connect to the network with bitswap)
	blockstore *hamt.CborIpldStore
}

func NewDefaultSyncer(blockstore *hamt.CborIpldStore) DefaultSyncer {
	return DefaultSyncer{
		blockstore: blockstore,
	}
}

// HandleNewTipsetFromNetwork hears about new tipsets from the network and then decides what to do with them.
func (syncer *DefaultSyncer) HandleNewBlocksFromNetwork(ctx context.Context, cids []*cid.Cid, height uint64) {
	// First implementation is something like:
	//     var wantBlocks []types.Block
	//     for cid := range cids {
	//       if !syncer.chainStore.Has(cid) && syncer.expectedConsensus.WantBlock(cid, height) {
	//         wantBlocks = append(wantBlocks, cid)
	//       }
	//     }
	//     blocks, err := fetcher.fetch(wantBlocks) // Note: probably better to use an iterator for streaming
	//     if !err {
	//       for block := range blocks {
	//         if syncer.expectedConsensus.IsValidBlock(block) {
	//           // Note: here we can see why we want separate IsValidBlock and MaybeAddNewHead
	//           // calls: because we don't want to save invalid blocks, but we also need to save valid blocks
	//           // before trying to extend the chain, so they are available through the chain.Store when the
	//           // NewHead event fires.
	//           syncer.chainStore.Put(block)
	//           syncer.expectedConsensus.MaybeAddNewHead(block)
	//         }
	//       }
	//     }
}
