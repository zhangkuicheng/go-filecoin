package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-floodsub"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
)

var ProtocolID = protocol.ID("/fil/0.0.0")
var TxsTopic = "/fil/tx"
var BlocksTopic = "/fil/blks"

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
		newBlocks: make(chan *Block),
		blockCallback: func(b *Block) error {
			return fcn.SendNewBlock(b)
		},
		currentBlock: fcn.bestBlock,
	}
	fcn.miner = m

	go m.Run(context.Background())

	fsub := floodsub.NewFloodSub(context.Background(), h)
	txsub, err := fsub.SubscribeWithOpts(&floodsub.TopicDescriptor{Name: &TxsTopic}, nil)
	if err != nil {
		return nil, err
	}

	blksub, err := fsub.SubscribeWithOpts(&floodsub.TopicDescriptor{Name: &BlocksTopic}, nil)
	if err != nil {
		return nil, err
	}

	go fcn.processNewBlocks(blksub)

	h.SetStreamHandler(ProtocolID, fcn.handleNewStream)

	fcn.txsub = txsub
	fcn.bsub = blksub
	fcn.pubsub = fsub

	return fcn, nil
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
			fmt.Printf("new block from %s: score %d\n", msg.GetFrom(), blk.Score())
			fcn.bestBlock = &blk
			fcn.miner.newBlocks <- &blk
		}
	}
}

type NetMessage struct {
	Head        *cid.Cid
	Block       *Block
	BlockHeight uint64
	Msgs        []*Message

	MsgType uint64
}

func (nm *NetMessage) WriteTo(w io.Writer) error {
	// TODO: this is crap and you know it
	return json.NewEncoder(w).Encode(nm)
}

func (nm *NetMessage) ReadFrom(r io.Reader) error {
	return json.NewDecoder(r).Decode(nm)
}

// TODO: protobuf? cbor? gonna json the hell out of this for now in hopes
// someone else gets mad enough to change it
// (why): protobuf faster and smaller, but not 'easily inspectable'. My
// argument is that easy inspection doesnt matter here since you have to know
// what youre looking at anyways
func (fcn *FilecoinNode) handleNewStream(s net.Stream) {
	defer s.Close()
	var msg NetMessage
	for {
		if err := msg.ReadFrom(s); err != nil {
			log.Warning("message decoding failed: ", err)
			s.Reset()
			return
		}

		// TODO: should the transaction pool take care of sending new
		// transactions to our other peers? or should we have some explicit
		// method "gossipTransactions" that we call here?
		for _, tx := range msg.Msgs {
			if err := fcn.txPool.Add(tx); err != nil {
				log.Warning("got invalid transaction: ", err)
			}
		}

		if !msg.Head.Equals(fcn.Head) {
			log.Error("we got a new block! do something...")
			// we gotta do a thing
		}
	}
}

func (fcn *FilecoinNode) validateBlock(b *Block) error {
	return nil
}

func (fcn *FilecoinNode) SendNewBlock(b *Block) error {

	// TODO: add block to blockservice?
	//fcn.Head = cbn.Cid()

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	return fcn.pubsub.Publish(BlocksTopic, data)
}

type peerHandle struct {
	send       chan *NetMessage
	disconnect chan error
	s          net.Stream
}

func (ph *peerHandle) shutdown() {
	// TODO: unregister ourselves from somewhere?
	ph.s.Close()
}

func (ph *peerHandle) run(ctx context.Context) {
	defer ph.shutdown()
	for {
		select {
		case msg := <-ph.send:
			// SEND IT
			if err := msg.WriteTo(ph.s); err != nil {
				log.Warning("message write failed: ", err)
				return
			}
		case err := <-ph.disconnect:
			_ = err
			// Send them the disconnect message, its not me, its you.
		case <-ctx.Done():
			// context cancel
			// TODO: should we still send them a disconnect message?
		}
	}
}

type fcnNotifiee FilecoinNode

var _ net.Notifiee = (*fcnNotifiee)(nil)

func (n *fcnNotifiee) ClosedStream(_ net.Network, s net.Stream)  {}
func (n *fcnNotifiee) OpenedStream(_ net.Network, s net.Stream)  {}
func (n *fcnNotifiee) Connected(_ net.Network, c net.Conn)       {}
func (n *fcnNotifiee) Disconnected(_ net.Network, c net.Conn)    {}
func (n *fcnNotifiee) Listen(_ net.Network, a ma.Multiaddr)      {}
func (n *fcnNotifiee) ListenClose(_ net.Network, a ma.Multiaddr) {}
