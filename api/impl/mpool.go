package impl

import (
	"context"

	"github.com/filecoin-project/go-filecoin/chain"
	"github.com/filecoin-project/go-filecoin/node"
)

type nodeMpool struct {
	api *nodeAPI
}

func newNodeMpool(api *nodeAPI) *nodeMpool {
	return &nodeMpool{api: api}
}

func (api *nodeMpool) View(ctx context.Context, messageCount uint) ([]*chain.SignedMessage, error) {
	nd := api.api.node

	pending := nd.MsgPool.Pending()
	if len(pending) < int(messageCount) {
		subscription, err := nd.PubSub.Subscribe(node.MessageTopic)
		if err != nil {
			return nil, err
		}

		for len(pending) < int(messageCount) {
			_, err = subscription.Next(ctx)
			if err != nil {
				return nil, err
			}
			pending = nd.MsgPool.Pending()
		}
	}

	return pending, nil
}
