package impl

import (
	"context"

	"gx/ipfs/QmZFbDTY9jfSBms2MchvYM9oYRbAF19K7Pby47yDBfpPrb/go-cid"

	"github.com/filecoin-project/go-filecoin/chain"
)

type nodeBlock struct {
	api *nodeAPI
}

func newNodeBlock(api *nodeAPI) *nodeBlock {
	return &nodeBlock{api: api}
}

func (api *nodeBlock) Get(ctx context.Context, id *cid.Cid) (*chain.Block, error) {
	return api.api.node.ChainMgr.FetchBlock(ctx, id)
}
