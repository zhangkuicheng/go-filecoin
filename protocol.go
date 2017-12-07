package main

import (
	"context"
	"encoding/json"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmbD5yKbXahNvoMqzeuNyKQA9vAs9fUvJg2GXeWU1fVqY5/go-libp2p-net"
)

var TxsTopic = "/fil/tx"
var BlocksTopic = "/fil/blks"

const HelloProtocol = "/filecoin/hello/0.0.1"

type HelloMessage struct {
	// Could just put the full block header here
	Head        *cid.Cid
	BlockHeight uint64

	// Maybe add some other info to gossip
}

func (fcn *FilecoinNode) HelloPeer(p peer.ID) {
	ctx := context.Background() // TODO: add appropriate timeout
	s, err := fcn.h.NewStream(ctx, p, HelloProtocol)
	if err != nil {
		log.Error("failed to open stream to new peer for hello: ", err)
		return
	}
	defer s.Close()

	hello := &HelloMessage{
		Head:        fcn.stateMgr.headCid,
		BlockHeight: fcn.stateMgr.bestBlock.Score(),
	}

	if err := json.NewEncoder(s).Encode(hello); err != nil {
		log.Error("marshaling hello message to new peer: ", err)
		return
	}
}

func (fcn *FilecoinNode) handleHelloStream(s net.Stream) {
	var hello HelloMessage
	if err := json.NewDecoder(s).Decode(&hello); err != nil {
		log.Error("decoding hello message: ", err)
		return
	}

	// TODO: inform the syncer
}

type fcnNotifiee FilecoinNode

var _ net.Notifiee = (*fcnNotifiee)(nil)

func (n *fcnNotifiee) ClosedStream(_ net.Network, s net.Stream) {}
func (n *fcnNotifiee) OpenedStream(_ net.Network, s net.Stream) {}
func (n *fcnNotifiee) Connected(_ net.Network, c net.Conn) {
}

func (n *fcnNotifiee) Disconnected(_ net.Network, c net.Conn)    {}
func (n *fcnNotifiee) Listen(_ net.Network, a ma.Multiaddr)      {}
func (n *fcnNotifiee) ListenClose(_ net.Network, a ma.Multiaddr) {}
