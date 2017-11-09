package main

import (
	"context"
	"encoding/json"
	"io"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
)

var TxsTopic = "/fil/tx"
var BlocksTopic = "/fil/blks"

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
