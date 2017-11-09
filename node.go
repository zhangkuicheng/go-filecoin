package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-floodsub"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
)

var ProtocolID = protocol.ID("/fil/0.0.0")

type FilecoinNode struct {
	h host.Host

	txPool TransactionPool

	peersLk sync.Mutex
	peers   map[peer.ID]*peerHandle

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	// Head is the cid of the head of the blockchain as far as this node knows
	Head *cid.Cid

	miner *Miner

	bestBlock *Block
}

func NewFilecoinNode(h host.Host) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:         h,
		bestBlock: new(Block),
	}

	m := &Miner{
		newBlocks:     make(chan *Block),
		blockCallback: fcn.SendNewBlock,
		currentBlock:  fcn.bestBlock,
	}
	fcn.miner = m

	go m.Run(context.Background())

	fsub := floodsub.NewFloodSub(context.Background(), h)
	txsub, err := fsub.Subscribe(TxsTopic)
	if err != nil {
		return nil, err
	}

	blksub, err := fsub.Subscribe(BlocksTopic)
	if err != nil {
		return nil, err
	}

	go fcn.processNewBlocks(blksub)
	go fcn.processNewTransactions(txsub)

	h.SetStreamHandler(ProtocolID, fcn.handleNewStream)

	fcn.txsub = txsub
	fcn.bsub = blksub
	fcn.pubsub = fsub

	return fcn, nil
}

func (fcn *FilecoinNode) processNewTransactions(txsub *floodsub.Subscription) {
	for {
		msg, err := txsub.Next(context.Background())
		if err != nil {
			panic(err)
		}

		var txmsg Message
		if err := json.Unmarshal(msg.GetData(), &txmsg); err != nil {
			panic(err)
		}

		if err := fcn.txPool.Add(&txmsg); err != nil {
			panic(err)
		}
	}
}

func (fcn *FilecoinNode) processNewBlocks(blksub *floodsub.Subscription) {
	for {
		msg, err := blksub.Next(context.Background())
		if err != nil {
			panic(err)
		}

		var blk Block
		if err := json.Unmarshal(msg.GetData(), &blk); err != nil {
			panic(err)
		}

		if blk.Score() > fcn.bestBlock.Score() {
			fcn.bestBlock = &blk
			if msg.GetFrom() != fcn.h.ID() {
				fmt.Printf("new block from %s: score %d\n", msg.GetFrom(), blk.Score())
				fcn.miner.newBlocks <- &blk
			}
		}
	}
}

func (fcn *FilecoinNode) validateBlock(b *Block) error {
	return nil
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	return fcn.pubsub.Publish(BlocksTopic, data)
}
