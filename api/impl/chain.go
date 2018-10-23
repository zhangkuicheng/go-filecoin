package impl

import (
	"context"

	"github.com/filecoin-project/go-filecoin/consensus"	
	"github.com/filecoin-project/go-filecoin/types"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmZFbDTY9jfSBms2MchvYM9oYRbAF19K7Pby47yDBfpPrb/go-cid"
)

type nodeChain struct {
	api *nodeAPI
}

func newNodeChain(api *nodeAPI) *nodeChain {
	return &nodeChain{api: api}
}

func (api *nodeChain) Head() ([]*cid.Cid, error) {
	ts := api.api.node.ChainReader.Head()
	if len(ts) == 0 {
		return nil, ErrHeaviestTipSetNotFound
	}
	tsSlice := ts.ToSlice()
	out := make([]*cid.Cid, len(tsSlice))
	for i, b := range tsSlice {
		out[i] = b.Cid()
	}

	return out, nil
}

func (api *nodeChain) Ls(ctx context.Context) <-chan interface{} {
	return api.api.node.ChainReader.BlockHistory(ctx)
}

func (api *nodeChain) LsCids(ctx context.Context) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for maybeTS := range api.api.node.ChainReader.BlockHistory(ctx){
			switch maybeTS.(type) {
			case consensus.TipSet:
				var cids types.SortedCidSet
				ts, ok := maybeTS.(consensus.TipSet)
				if !ok {
					out<-errors.New("tipset could not be cast")
					continue
				}
				for _, blk := range ts {
					(&cids).Add(blk.Cid())
				}
				out<-cids
			default:
				out<-maybeTS
			}
		}
	}()
	return out
}
