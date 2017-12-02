package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmRS46AyqtpJBsf1zmQdeizSDEzo1qkWR7rdEuPFAv8237/go-libp2p-host"
	"gx/ipfs/QmVNv1WV6XxzQV4MBuiLX5729wMazaf8TNzm2Sq6ejyHh7/go-libp2p-floodsub"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"

	hamt "github.com/ipfs/go-hamt-ipld"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	dag "github.com/ipfs/go-ipfs/merkledag"
)

var ProtocolID = protocol.ID("/fil/0.0.0")

type FilecoinNode struct {
	h host.Host

	txPool *TransactionPool

	peersLk sync.Mutex
	peers   map[peer.ID]*peerHandle

	Addresses []Address

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	dag   dag.DAGService
	bswap *bitswap.Bitswap

	// Head is the cid of the head of the blockchain as far as this node knows
	Head *cid.Cid

	miner *Miner

	stateRoot *State
	cs        *hamt.CborIpldStore

	bestBlock *Block

	knownGoodBlocks *cid.Set
}

func NewFilecoinNode(h host.Host, fs *floodsub.PubSub, dag dag.DAGService, bs bserv.BlockService, bswap *bitswap.Bitswap) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:               h,
		knownGoodBlocks: cid.NewSet(),
		dag:             dag,
		bswap:           bswap,
		cs:              &hamt.CborIpldStore{bs},
		txPool:          NewTransactionPool(),
	}
	baseAddr := createNewAddress()
	fcn.Addresses = []Address{baseAddr}
	fmt.Println("my mining address is ", baseAddr)

	genesis, err := CreateGenesisBlock(fcn.cs)
	if err != nil {
		return nil, err
	}
	fcn.bestBlock = genesis

	c, err := fcn.dag.Add(genesis.ToNode())
	if err != nil {
		return nil, err
	}
	fmt.Println("genesis block cid is: ", c)
	fcn.knownGoodBlocks.Add(c)

	// TODO: better miner construction and delay start until synced
	m := &Miner{
		newBlocks:     make(chan *Block),
		blockCallback: fcn.SendNewBlock,
		currentBlock:  fcn.bestBlock,
		address:       baseAddr,
		fcn:           fcn,
		txPool:        fcn.txPool,
	}
	fcn.miner = m

	// Run miner
	go m.Run(context.Background())

	txsub, err := fs.Subscribe(TxsTopic)
	if err != nil {
		return nil, err
	}

	blksub, err := fs.Subscribe(BlocksTopic)
	if err != nil {
		return nil, err
	}

	go fcn.processNewBlocks(blksub)
	go fcn.processNewTransactions(txsub)

	h.SetStreamHandler(ProtocolID, fcn.handleNewStream)

	fcn.txsub = txsub
	fcn.bsub = blksub
	fcn.pubsub = fs

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

func createNewAddress() Address {
	buf := make([]byte, 20)
	rand.Read(buf)
	return Address(buf)
}

func (fcn *FilecoinNode) processNewBlocks(blksub *floodsub.Subscription) {
	// TODO: this function should really just be a validator function for the pubsub subscription
	for {
		msg, err := blksub.Next(context.Background())
		if err != nil {
			panic(err)
		}
		if msg.GetFrom() == fcn.h.ID() {
			continue
		}

		blk, err := DecodeBlock(msg.GetData())
		if err != nil {
			panic(err)
		}

		if err := fcn.processNewBlock(context.Background(), blk); err != nil {
			log.Error(err)
			continue
		}
		fcn.miner.newBlocks <- blk
	}
}

func (fcn *FilecoinNode) processNewBlock(ctx context.Context, blk *Block) error {
	if err := fcn.validateBlock(ctx, blk); err != nil {
		return fmt.Errorf("validate block failed: %s", err)
	}

	if blk.Score() > fcn.bestBlock.Score() {
		return fcn.acceptNewBlock(blk)
	}

	return fmt.Errorf("new block not better than current block (%d <= %d)",
		blk.Score(), fcn.bestBlock.Score())
}

// acceptNewBlock sets the given block as our current 'best chain' block
func (fcn *FilecoinNode) acceptNewBlock(blk *Block) error {
	_, err := fcn.dag.Add(blk.ToNode())
	if err != nil {
		return fmt.Errorf("failed to put block to disk: %s", err)
	}

	fcn.knownGoodBlocks.Add(blk.Cid())
	fcn.bestBlock = blk

	// TODO: actually go through transactions for each block back to the last
	// common block and remove transactions/re-add transactions in blocks we
	// had but arent in the new chain
	for _, tx := range blk.Txs {
		c, err := tx.Cid()
		if err != nil {
			return err
		}

		fcn.txPool.ClearTx(c)
	}

	s, err := LoadState(context.Background(), fcn.cs, blk.StateRoot)
	if err != nil {
		return fmt.Errorf("failed to get newly approved state: %s", err)
	}
	fcn.stateRoot = s

	fmt.Printf("accepted new block, [s=%d, h=%s, st=%s]\n", blk.Score(), blk.Cid(), blk.StateRoot)
	return nil

}

func (fcn *FilecoinNode) fetchBlock(ctx context.Context, c *cid.Cid) (*Block, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var blk Block
	if err := fcn.cs.Get(ctx, c, &blk); err != nil {
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

func (fcn *FilecoinNode) checkBlockStateChangeValid(ctx context.Context, s *State, b *Block) error {
	if err := s.ApplyTransactions(ctx, b.Txs); err != nil {
		return err
	}

	c, err := s.Flush(ctx)
	if err != nil {
		return err
	}

	if !c.Equals(b.StateRoot) {
		return fmt.Errorf("state root failed to validate! (%s != %s)", c, b.StateRoot)
	}

	return nil
}

func (fcn *FilecoinNode) validateBlock(ctx context.Context, b *Block) error {
	if err := fcn.checkBlockValid(ctx, b); err != nil {
		return fmt.Errorf("check block valid failed: %s", err)
	}

	if b.Score() <= fcn.bestBlock.Score() {
		return fmt.Errorf("new block is not better than our current block")
	}

	var validating []*Block
	baseBlk := b
	for !fcn.knownGoodBlocks.Has(baseBlk.Cid()) { // probably should be some sort of limit here
		validating = append(validating, baseBlk)

		next, err := fcn.fetchBlock(ctx, baseBlk.Parent)
		if err != nil {
			return fmt.Errorf("fetch block failed: %s", err)
		}

		if err := fcn.checkBlockValid(ctx, next); err != nil {
			return err
		}

		baseBlk = next
	}

	s, err := LoadState(ctx, fcn.cs, baseBlk.StateRoot)
	if err != nil {
		return fmt.Errorf("load state failed: %s", err)
	}

	for i := len(validating) - 1; i >= 0; i-- {
		if err := fcn.checkBlockStateChangeValid(ctx, s, validating[i]); err != nil {
			return err
		}
		fcn.knownGoodBlocks.Add(validating[i].Cid())
	}

	fmt.Println("new known block: ", b.Cid())
	return nil
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {
	nd := b.ToNode()
	_, err := fcn.dag.Add(nd)
	if err != nil {
		return err
	}

	if err := fcn.processNewBlock(context.Background(), b); err != nil {
		return err
	}

	return fcn.pubsub.Publish(BlocksTopic, nd.RawData())
}

func (fcn *FilecoinNode) SendNewTransaction(tx *Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return fcn.pubsub.Publish(TxsTopic, data)
}
