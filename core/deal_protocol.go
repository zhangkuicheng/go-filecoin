package core

import (
	"context"
	"encoding/json"
	"fmt"

	inet "gx/ipfs/QmQm7WmgYCa4RSz76tKEYpRjApjnRw8ZTUVQC15b8JM4a2/go-libp2p-net"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	contract "github.com/filecoin-project/go-filecoin/contract"
	types "github.com/filecoin-project/go-filecoin/types"
	dag "github.com/ipfs/go-ipfs/merkledag"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
)

/*
The deal protocol

To make a deal, the storage client first selects a bid of their own making, and
an ask from a miner they wish to make a deal with.

The client then sends a 'DealMessage' to the miner, containing:
  - The bid
  - The ask
  - A reference to the data
  - The size of the data (TODO: should we require that len(data) == bid.size ?)
  - A signature over that data

The miner receives this, and decides whether or not to accept it.
If it does not accept, it sends a 'DealResponse' indicating that it does not
accept the deal, and terminates the protocol.

If the miner does accept the deal, it sends back a 'DealResponse' indicating
the deal has been accepted.  The miner then starts fetching the data from the
client out of band (TODO: figure this out better) Once the miner has
successfully retrieved the data, it creates a 'MakeDeal' transaction which it
signs and posts on chain.

Once the miner has successfully fetched the data and posted the deal to the
chain, they send a 'DealResult' message back to the client indicating the txid
(its hash).  If the transfer or the posting of the deal fails, the miner sends
back a 'DealResult' message indicating the error.
*/

var MakeDealProtocol = protocol.ID("/fil/deal/1.0.0")

type DealMessage struct {
	AskId, BidId uint64
	Data         *cid.Cid
	ClientSig    string
}

type DealResponse struct {
	OK bool
}

type DealResult struct {
	TxHash *cid.Cid
	Error  string
}

func (fcn *FilecoinNode) HandleMakeDeal(s inet.Stream) {
	fmt.Println("====== HANDLING MAKE DEAL!")
	ctx := context.TODO()
	defer s.Close()
	dec := json.NewDecoder(s)
	enc := json.NewEncoder(s)

	var m DealMessage
	if err := dec.Decode(&m); err != nil {
		log.Error("failed to decode incoming deal message: ", err)
		return
	}

	storage, cst, err := fcn.LoadStorageContract(ctx)
	if err != nil {
		log.Error("failed to load storage contract: ", err)
		return
	}

	cctx := &contract.CallContext{
		ContractState: cst,
		Ctx:           ctx,
	}

	ask, err := storage.GetAsk(cctx, m.AskId)
	if err != nil {
		// TODO: write back an error message?
		log.Error(errors.Wrap(err, "failed to get ask"))
		return
	}

	bid, err := storage.GetBid(cctx, m.BidId)
	if err != nil {
		// TODO: write back an error message?
		log.Error(errors.Wrap(err, "failed to get bid"))
		return
	}

	_ = bid

	if !fcn.IsOurAddress(ask.MinerID) {
		log.Error("ask in deal is not ours")
		return
	}

	// TODO: validate bid/ask pair (is there enough space, price, balances, etc)

	// TODO: don't always auto-accept. Need manual acceptance too
	if err := enc.Encode(&DealResponse{OK: true}); err != nil {
		log.Error("failed to write deal response: ", err)
		return
	}

	fmt.Println("====== Fetching Data")

	// Receive file...
	// TODO: do the fancy thing that nico and juan were talking about
	if err := dag.FetchGraph(ctx, m.Data, fcn.DAG); err != nil {
		log.Error("fetching data failed: ", err)
		return
	}

	fmt.Println("======= Data fetched!")

	nonce, err := fcn.StateMgr.StateRoot.NonceForActor(ctx, ask.MinerID)
	if err != nil {
		log.Error(errors.Wrap(err, "getting nonce failed"))
		return
	}

	tx := &types.Transaction{
		From:   ask.MinerID,
		To:     contract.StorageContractAddress,
		Method: "makeDeal",
		Nonce:  nonce,
		// TODO: also need dataref, and other actual deal fields
		Params: []interface{}{m.AskId, m.BidId, m.ClientSig},
	}

	txcid, err := tx.Cid()
	if err != nil {
		log.Error("getting cid from tx we created: ", err)
		return
	}

	fmt.Println("====== SENDING DEAL TRANSACTION!")
	if err := fcn.SendNewTransaction(tx); err != nil {
		log.Error("sending transaction confirming deal:", err)
		return
	}

	res := &DealResult{
		TxHash: txcid,
	}

	if err := enc.Encode(res); err != nil {
		log.Error(errors.Wrap(err, "failed to write back deal response"))
		return
	}
}

func ClientMakeDeal(ctx context.Context, fcn *FilecoinNode, askId, bidId uint64, data *cid.Cid) (*cid.Cid, error) {
	fmt.Println("====== Lets make a deal!")
	storage, cst, err := fcn.LoadStorageContract(ctx)
	if err != nil {
		return nil, err
	}

	cctx := &contract.CallContext{
		ContractState: cst,
		Ctx:           ctx,
	}

	ask, err := storage.GetAsk(cctx, askId)
	if err != nil {
		return nil, err
	}

	bid, err := storage.GetBid(cctx, bidId)
	if err != nil {
		return nil, err
	}

	if !fcn.IsOurAddress(bid.Owner) {
		return nil, fmt.Errorf("not our bid")
	}

	// TODO: validation...?
	deal := &DealMessage{
		AskId:     askId,
		BidId:     bidId,
		Data:      data,
		ClientSig: "foobar", // TODO: crypto
	}

	fmt.Println("====== Finding that pesky miner")
	minerPid, err := fcn.Lookup.Lookup(ask.MinerID)
	if err != nil {
		return nil, err
	}
	fmt.Println("====== Found the miner!", minerPid)

	fmt.Println("====== Opening new stream to miner!")
	s, err := fcn.Host.NewStream(ctx, minerPid, MakeDealProtocol)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(s)
	if err := enc.Encode(deal); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(s)

	var dealResp DealResponse
	if err := dec.Decode(&dealResp); err != nil {
		return nil, err
	}
	fmt.Println("====== Got a deal response", dealResp.OK)

	if !dealResp.OK {
		return nil, fmt.Errorf("miner rejected deal")
	}

	var res DealResult
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	fmt.Println("====== Got a deal result!", res)
	if res.Error != "" {
		return nil, fmt.Errorf("deal result error: %s", res.Error)
	}

	return res.TxHash, nil
}
