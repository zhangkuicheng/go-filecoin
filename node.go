package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"gx/ipfs/QmRS46AyqtpJBsf1zmQdeizSDEzo1qkWR7rdEuPFAv8237/go-libp2p-host"
	"gx/ipfs/QmVNv1WV6XxzQV4MBuiLX5729wMazaf8TNzm2Sq6ejyHh7/go-libp2p-floodsub"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"

	hamt "github.com/ipfs/go-hamt-ipld"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	dag "github.com/ipfs/go-ipfs/merkledag"
)

var ProtocolID = protocol.ID("/fil/0.0.0")

type FilecoinNode struct {
	h host.Host

	Addresses []Address

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	dag   dag.DAGService
	bswap *bitswap.Bitswap

	stateMgr *StateManager

	miner *Miner

	cs *hamt.CborIpldStore
}

func NewFilecoinNode(h host.Host, fs *floodsub.PubSub, dag dag.DAGService, bs bserv.BlockService, bswap *bitswap.Bitswap) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:     h,
		dag:   dag,
		bswap: bswap,
		cs:    &hamt.CborIpldStore{bs},
	}

	s := &StateManager{
		knownGoodBlocks: cid.NewSet(),
		txPool:          NewTransactionPool(),
		cs:              fcn.cs,
		dag:             fcn.dag,
	}

	fcn.stateMgr = s

	baseAddr := createNewAddress()
	fcn.Addresses = []Address{baseAddr}
	fmt.Println("my mining address is ", baseAddr)

	genesis, err := CreateGenesisBlock(fcn.cs)
	if err != nil {
		return nil, err
	}
	s.bestBlock = genesis

	c, err := fcn.dag.Add(genesis.ToNode())
	if err != nil {
		return nil, err
	}
	fmt.Println("genesis block cid is: ", c)
	s.knownGoodBlocks.Add(c)

	// TODO: better miner construction and delay start until synced
	m := &Miner{
		newBlocks:     make(chan *Block),
		blockCallback: fcn.SendNewBlock,
		currentBlock:  s.bestBlock,
		address:       baseAddr,
		fcn:           fcn,
		txPool:        s.txPool,
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

	h.SetStreamHandler(ProtocolID, fcn.handleHelloStream)

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

		if err := fcn.stateMgr.txPool.Add(&txmsg); err != nil {
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

		if err := fcn.stateMgr.processNewBlock(context.Background(), blk); err != nil {
			log.Error(err)
			continue
		}
		fcn.miner.newBlocks <- blk
	}
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {
	nd := b.ToNode()
	_, err := fcn.dag.Add(nd)
	if err != nil {
		return err
	}

	if err := fcn.stateMgr.processNewBlock(context.Background(), b); err != nil {
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
