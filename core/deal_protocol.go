package core

import (
	"context"
	"encoding/json"
	"fmt"

	inet "gx/ipfs/QmU4vCDZTPLDqSDKguWbHCiUe46mZUtmM2g2suBZ9NE8ko/go-libp2p-net"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"

	contract "github.com/filecoin-project/playground/go-filecoin/contract"

	"github.com/pkg/errors"
)

var MakeDealProtocol = protocol.ID("/fil/deal/1.0.0")

type DealMessage struct {
	AskId, BidId uint64
	Data         *cid.Cid
}

func (fcn *FilecoinNode) HandleMakeDeal(s inet.Stream) {
	defer s.Close()
	dec := json.NewDecoder(s)

	var m DealMessage
	if err := dec.Decode(&m); err != nil {
		log.Error("failed to decode incoming deal message: ", err)
		return
	}

	storage, cst, err := fcn.LoadStorageContract(context.TODO())
	if err != nil {
		log.Error("failed to load storage contract: ", err)
		return
	}

	cctx := &contract.CallContext{
		ContractState: cst,
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

	if !fcn.IsOurAddress(ask.MinerID) {
		log.Error("ask in deal is not ours")
		return
	}

	_ = bid
}

func clientMakeDeal(ctx context.Context, fcn *FilecoinNode, askId, bidId uint64, data *cid.Cid) (interface{}, error) {
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
		AskId: askId,
		BidId: bidId,
		Data:  data,
	}

	minerPid, err := fcn.Lookup.Lookup(ask.MinerID)
	if err != nil {
		return nil, err
	}

	s, err := fcn.Host.NewStream(ctx, minerPid, MakeDealProtocol)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(s)
	if err := enc.Encode(deal); err != nil {
		return nil, err
	}

	_ = bid
	panic("NYI")
}
