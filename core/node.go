package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"gx/ipfs/QmP1T1SGU6276R2MHKP2owbck37Fnzd6ZkpyNJvnG2LoTG/go-libp2p-floodsub"
	"gx/ipfs/QmP46LGWhzVZTMmt5akNNLfoV8qL4h5wTwmzQxLyDafggd/go-libp2p-host"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"

	hamt "github.com/ipfs/go-hamt-ipld"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	dag "github.com/ipfs/go-ipfs/merkledag"
)

var log = logging.Logger("core")

var ProtocolID = protocol.ID("/fil/0.0.0")

type FilecoinNode struct {
	h host.Host

	Addresses []Address

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	DAG     dag.DAGService
	Bitswap *bitswap.Bitswap
	cs      *hamt.CborIpldStore

	StateMgr *StateManager
}

func NewFilecoinNode(h host.Host, fs *floodsub.PubSub, dag dag.DAGService, bs bserv.BlockService, bswap *bitswap.Bitswap) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:       h,
		DAG:     dag,
		Bitswap: bswap,
		cs:      &hamt.CborIpldStore{bs},
	}

	s := &StateManager{
		knownGoodBlocks: cid.NewSet(),
		txPool:          NewTransactionPool(),
		cs:              fcn.cs,
		dag:             fcn.DAG,
	}

	fcn.StateMgr = s

	baseAddr := CreateNewAddress()
	fcn.Addresses = []Address{baseAddr}
	fmt.Println("my mining address is ", baseAddr)

	genesis, err := CreateGenesisBlock(fcn.cs)
	if err != nil {
		return nil, err
	}
	s.bestBlock = genesis

	c, err := fcn.DAG.Add(genesis.ToNode())
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
	s.miner = m

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

	h.SetStreamHandler(HelloProtocol, fcn.handleHelloStream)
	h.Network().Notify((*fcnNotifiee)(fcn))

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

		if err := fcn.StateMgr.txPool.Add(&txmsg); err != nil {
			panic(err)
		}
	}
}

func CreateNewAddress() Address {
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

		fcn.StateMgr.Inform(msg.GetFrom(), blk)
	}
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {
	nd := b.ToNode()
	_, err := fcn.DAG.Add(nd)
	if err != nil {
		return err
	}

	if err := fcn.StateMgr.processNewBlock(context.Background(), b); err != nil {
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
