package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmUUSLfvihARhCxxgnjW4hmycJpPvzNu12Aaz6JWVdfnLg/go-libp2p-floodsub"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"

	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	bstore "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/blocks/blockstore"
	bserv "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/blockservice"
	bitswap "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/exchange/bitswap"
	bsnet "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/exchange/bitswap/network"
	dag "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/merkledag"
	none "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/routing/none"
)

var ProtocolID = protocol.ID("/fil/0.0.0")

var GenesisBlock = dag.NewRawNode([]byte(`{"genesis":"yay"}`))

type FilecoinNode struct {
	h host.Host

	txPool TransactionPool

	peersLk sync.Mutex
	peers   map[peer.ID]*peerHandle

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	dag dag.DAGService

	// Head is the cid of the head of the blockchain as far as this node knows
	Head *cid.Cid

	miner *Miner

	bestBlock    *Block
	bestBlockCid *cid.Cid // doesnt need to be separate, but since i'm kinda cheating with the whole serialization thing it works for now

	knownGoodBlocks *cid.Set
}

func NewFilecoinNode(h host.Host) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:               h,
		bestBlock:       new(Block),
		knownGoodBlocks: cid.NewSet(),
	}

	m := &Miner{
		newBlocks:     make(chan *Block),
		blockCallback: fcn.SendNewBlock,
		currentBlock:  fcn.bestBlock,
	}
	fcn.miner = m

	// Run miner
	go m.Run(context.Background())

	// TODO: maybe this gets passed in?
	fsub := floodsub.NewFloodSub(context.Background(), h)

	// Also should probably pass in the dagservice instance
	bs := bstore.NewBlockstore(ds.NewMapDatastore())
	nilr, _ := none.ConstructNilRouting(nil, nil, nil)
	bsnet := bsnet.NewFromIpfsHost(h, nilr)
	bswap := bitswap.New(context.Background(), h.ID(), bsnet, bs, true)
	bserv := bserv.New(bs, bswap)
	fcn.dag = dag.NewDAGService(bserv)

	c, err := fcn.dag.Add(GenesisBlock)
	if err != nil {
		return nil, err
	}
	fmt.Println("genesis block cid is: ", c)
	fcn.knownGoodBlocks.Add(c)

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
	// TODO: this function should really just be a validator function for the pubsub subscription
	for {
		msg, err := txsub.Next(context.Background())
		if err != nil {
			panic(err)
		}

		var txmsg Transaction
		if err := json.Unmarshal(msg.GetData(), &txmsg); err != nil {
			panic(err)
		}

		if err := fcn.txPool.Add(&txmsg); err != nil {
			panic(err)
		}
	}
}

func (fcn *FilecoinNode) processNewBlocks(blksub *floodsub.Subscription) {
	// TODO: this function should really just be a validator function for the pubsub subscription
	for {
		msg, err := blksub.Next(context.Background())
		if err != nil {
			panic(err)
		}

		var blk Block
		if err := json.Unmarshal(msg.GetData(), &blk); err != nil {
			panic(err)
		}

		if err := fcn.validateBlock(&blk); err != nil {
			log.Error("invalid block: ", err)
			continue
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

func (fcn *FilecoinNode) fetchBlock(ctx context.Context, c *cid.Cid) (*Block, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	nd, err := fcn.dag.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	var blk Block
	if err := json.Unmarshal(nd.RawData(), &blk); err != nil {
		return nil, err
	}

	return &blk, nil
}

// checkSingleBlock verifies that this block, on its own, is structurally and
// cryptographically valid. This means checking that all of its fields are
// properly filled out and its signature is correct. Checking the validity of
// state changes must be done separately and only once the state of the
// previous block has been validated.
func (fcn *FilecoinNode) checkBlockValid(ctx context.Context, b *Block) error {
	return nil
}

func (fcn *FilecoinNode) checkBlockStateChangeValid(ctx context.Context, b *Block /* and argument for current state */) error {
	// TODO
	return nil
}

func (fcn *FilecoinNode) validateBlock(ctx context.Context, b *Block) error {
	if err := fcn.checkBlockValid(ctx, b); err != nil {
		return err
	}

	if b.Score() <= fcn.bestBlock.Score() {
		return fmt.Errorf("new block is not better than our current block")
	}
	if fcn.knownGoodBlocks.Has(b.Parent) {
		return nil
	}
	validating := []*Block{b}
	cur := b
	for { // probably should be some sort of limit here
		next, err := fcn.fetchBlock(ctx, cur.Parent)
		if err != nil {
			return err
		}

		if err := fcn.checkSingleBlock(next); err != nil {
			return err
		}

		if fcn.knownGoodBlocks.Has(next.Parent) {
			// we have a known good root
			break
		}
		validating = append(validating, next)
		cur = next
	}

	for i := len(validating) - 1; i >= 0; i-- {
		if err := fcn.checkBlockStateChangeValid(ctx, validating[i]); err != nil {
			return err
		}
	}

	// do we set this as our 'best block' here? Or should the caller handle that?
	fcn.knownGoodBlocks.Add(b.Cid())

	return nil
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	// this is a hack... but... whatever...
	c, err := fcn.dag.Add(dag.NewRawNode(data))
	if err != nil {
		return err
	}
	_ = c

	return fcn.pubsub.Publish(BlocksTopic, data)
}
