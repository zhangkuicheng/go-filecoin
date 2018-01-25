package contract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"

	types "github.com/filecoin-project/playground/go-filecoin/types"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	hamt "gx/ipfs/QmeEgzPRAjisT3ndLSR8jrrZAZyWd3nx2mpZU4S7mCQzYi/go-hamt-ipld"
)

var StorageContractCodeCid = identCid("storageContract")
var StorageContractAddress = types.Address("storageContract")

const asksArrKey = "activeAsks"
const bidsArrKey = "activeBids"
const dealsArrKey = "activeDeals"

// Only Duration or BlockHeight are allowed to be defined
type Bid struct {
	Expiry uint64

	Price *big.Int

	Size uint64

	// number of blocks, from the deal being commited to the chain
	Duration uint64

	// fixed block height
	BlockHeight uint64

	Collateral *big.Int

	Owner types.Address

	//Coding      ErasureCoding
}

type Ask struct {
	Expiry uint64

	Price *big.Int

	Size uint64

	MinerID types.Address
}

type StorageContract struct{}

func (sc *StorageContract) getBids(cctx *CallContext) (uint64, error) {
	return sc.loadUint64(cctx, "bids")
}

func (sc *StorageContract) getAsks(cctx *CallContext) (uint64, error) {
	return sc.loadUint64(cctx, "asks")
}

func (sc *StorageContract) getDeals(cctx *CallContext) (uint64, error) {
	return sc.loadUint64(cctx, "deals")
}

func (sc *StorageContract) loadUint64(cctx *CallContext, k string) (uint64, error) {
	asksd, err := cctx.ContractState.Get(cctx.Ctx, k)
	if err != nil && err != hamt.ErrNotFound {
		return 0, err
	}

	return big.NewInt(0).SetBytes(asksd).Uint64(), nil
}

func (sc *StorageContract) storeUint64(cctx *CallContext, k string, v uint64) error {
	return cctx.ContractState.Set(cctx.Ctx, k, big.NewInt(0).SetUint64(v).Bytes())
}

func (sc *StorageContract) Call(ctx *CallContext, method string, args []interface{}) (interface{}, error) {
	switch method {
	case "addAsk":
		return mustTypedCallClosure(sc.addAsk)(ctx, args)
	case "addBid":
		return mustTypedCallClosure(sc.addBid)(ctx, args)
	case "createMiner":
		return sc.createMiner(ctx, args)
	case "getAsks":
		return sc.loadArray(ctx, asksArrKey)
	case "getBids":
		return sc.loadArray(ctx, bidsArrKey)
	case "makeDeal":
		return mustTypedCallClosure(sc.makeDealCall)(ctx, args)
	default:
		return nil, ErrMethodNotFound
	}
}

func (sc *StorageContract) ListAsks(cctx *CallContext) ([]*Ask, error) {
	ids, err := sc.loadArray(cctx, asksArrKey)
	if err != nil {
		return nil, err
	}
	var asks []*Ask
	for _, id := range ids {
		a, err := sc.GetAsk(cctx, id)
		if err != nil {
			return nil, err
		}
		asks = append(asks, a)
	}
	return asks, nil
}

func (sc *StorageContract) GetAsk(cctx *CallContext, id uint64) (*Ask, error) {
	data, err := cctx.ContractState.Get(cctx.Ctx, "a"+fmt.Sprint(id))
	if err != nil {
		return nil, err
	}

	var a Ask
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

func (sc *StorageContract) GetBid(cctx *CallContext, id uint64) (*Bid, error) {
	data, err := cctx.ContractState.Get(cctx.Ctx, "b"+fmt.Sprint(id))
	if err != nil {
		return nil, err
	}

	var b Bid
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (sc *StorageContract) ListBids(cctx *CallContext) ([]*Bid, error) {
	ids, err := sc.loadArray(cctx, bidsArrKey)
	if err != nil {
		return nil, err
	}
	var bids []*Bid
	for _, id := range ids {
		b, err := sc.GetBid(cctx, id)
		if err != nil {
			return nil, err
		}
		bids = append(bids, b)
	}
	return bids, nil
}

func castBid(i interface{}) (*Bid, error) {
	switch i := i.(type) {
	case *Bid:
		return i, nil
	case map[string]interface{}:
		d, err := json.Marshal(i)
		if err != nil {
			return nil, err
		}

		var b Bid
		if err := json.Unmarshal(d, &b); err != nil {
			return nil, err
		}

		return &b, nil
	default:
		fmt.Printf("bid: %#v\n", i)
		panic("halten sie!")
	}
}

func castAsk(i interface{}) (*Ask, error) {
	switch i := i.(type) {
	case *Ask:
		return i, nil
	case map[string]interface{}:
		d, err := json.Marshal(i)
		if err != nil {
			return nil, err
		}

		var a Ask
		if err := json.Unmarshal(d, &a); err != nil {
			return nil, err
		}

		return &a, nil
	default:
		fmt.Printf("ask: %#v\n", i)
		panic("halten sie!")
	}
}

func (sc *StorageContract) addBid(ctx *CallContext, price, size uint64) (interface{}, error) {
	b := &Bid{
		Owner: ctx.From,
		Price: big.NewInt(0).SetUint64(price),
		Size:  size,
	}
	if err := sc.validateBid(b); err != nil {
		return nil, err
	}

	bidID, err := sc.getBids(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if err := ctx.ContractState.Set(ctx.Ctx, "b"+fmt.Sprint(bidID), data); err != nil {
		return nil, err
	}

	if err := sc.addActiveBid(ctx, bidID); err != nil {
		return nil, err
	}

	return bidID, nil
}

func (sc *StorageContract) addActiveAsk(ctx *CallContext, id uint64) error {
	asks, err := sc.loadArray(ctx, asksArrKey)
	if err != nil {
		return err
	}

	asks = append(asks, id)

	return sc.storeArray(ctx, asksArrKey, asks)
}

func (sc *StorageContract) storeArray(ctx *CallContext, k string, arr []uint64) error {
	// TODO: find a better structure for arrays
	data, err := json.Marshal(arr)
	if err != nil {
		return err
	}

	return ctx.ContractState.Set(ctx.Ctx, k, data)

}

func (sc *StorageContract) loadArray(ctx *CallContext, k string) ([]uint64, error) {
	// TODO: find a better structure for arrays
	data, err := ctx.ContractState.Get(ctx.Ctx, k)
	switch err {
	case hamt.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	case nil:
		// noop
	}

	var out []uint64
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	return out, nil

}

func (sc *StorageContract) addActiveBid(ctx *CallContext, id uint64) error {
	bids, err := sc.loadArray(ctx, bidsArrKey)
	if err != nil {
		return err
	}

	bids = append(bids, id)

	return sc.storeArray(ctx, bidsArrKey, bids)
}

func (sc *StorageContract) removeActiveAsk(ctx *CallContext, id uint64) error {
	return sc.removeFromArray(ctx, asksArrKey, id)
}
func (sc *StorageContract) removeActiveBid(ctx *CallContext, id uint64) error {
	return sc.removeFromArray(ctx, bidsArrKey, id)
}

func (sc *StorageContract) removeFromArray(ctx *CallContext, k string, id uint64) error {
	arr, err := sc.loadArray(ctx, k)
	if err != nil {
		return err
	}

	for i, v := range arr {
		if v == id {
			arr = append(arr[:i], arr[i+1:]...)
			break
		}
	}

	return sc.storeArray(ctx, k, arr)
}

func (sc *StorageContract) validateBid(b *Bid) error {
	// check all the fields look good

	// need to check client has enough filecoin to lock up

	return nil
}

func (sc *StorageContract) getAsk(ctx *CallContext, id uint64) (*Ask, error) {
	d, err := ctx.ContractState.Get(ctx.Ctx, fmt.Sprintf("a%d", id))
	if err != nil {
		return nil, err
	}

	var a Ask
	if err := json.Unmarshal(d, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

func (sc *StorageContract) getBid(ctx *CallContext, id uint64) (*Bid, error) {
	d, err := ctx.ContractState.Get(ctx.Ctx, fmt.Sprintf("b%d", id))
	if err != nil {
		return nil, err
	}

	var b Bid
	if err := json.Unmarshal(d, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (sc *StorageContract) addAsk(ctx *CallContext, miner types.Address, price, size uint64) (interface{}, error) {

	tx := &types.Transaction{
		To:     miner,
		From:   ctx.Address,
		Method: "addAsk",
		Params: []interface{}{
			ctx.From,
			price,
			size,
		},
	}

	ask := &Ask{
		MinerID: ctx.From,
		Price:   big.NewInt(0).SetUint64(price),
		Size:    size,
	}

	// validate the ask with the miners contract
	_, err := ctx.State.ActorExec(ctx.Ctx, tx)
	if err != nil {
		return nil, err
	}

	id, err := sc.getAsks(ctx)
	if err != nil {
		return nil, err
	}

	if err := sc.storeUint64(ctx, "asks", id+1); err != nil {
		return nil, err
	}

	if err := sc.putOrder(ctx, fmt.Sprintf("a%d", id), ask); err != nil {
		return nil, err
	}

	if err := sc.addActiveAsk(ctx, id); err != nil {
		return nil, err
	}

	return id, nil
}

func (mn *MinerContract) AddAsk(ctx *CallContext, from types.Address, price, size uint64) (interface{}, error) {
	if err := mn.LoadState(ctx.ContractState); err != nil {
		return nil, fmt.Errorf("load state: %s", err)
	}

	if mn.Owner != ctx.From {
		return nil, &revertError{fmt.Errorf("not authorized to access that miner (%s != %s)", mn.Owner, ctx.From)}
	}

	s := big.NewInt(int64(size))
	total := big.NewInt(0).Set(mn.Pledge)
	total.Sub(total, mn.LockedStorage)

	if total.Cmp(s) < 0 {
		return nil, &revertError{fmt.Errorf("not enough available pledge")}
	}

	mn.LockedStorage = mn.LockedStorage.Add(mn.LockedStorage, s)

	if err := mn.Flush(ctx.Ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

func (sc *StorageContract) putOrder(ctx *CallContext, k string, o interface{}) error {
	d, err := json.Marshal(o)
	if err != nil {
		return err
	}

	return ctx.ContractState.Set(ctx.Ctx, k, d)
}

func (sc *StorageContract) addActiveDeal(ctx *CallContext, id uint64) error {
	arr, err := sc.loadArray(ctx, dealsArrKey)
	if err != nil {
		return err
	}

	arr = append(arr, id)
	return sc.storeArray(ctx, dealsArrKey, arr)
}

type Deal struct {
	MinerSig types.Address // using an address as the signature for now because i don't feel like adding crypto stuff yet
	Expiry   uint64
	DataRef  *cid.Cid

	Ask uint64
	Bid uint64
}

func (sc *StorageContract) makeDealCall(ctx *CallContext, ask, bid uint64, sig types.Address) (interface{}, error) {
	return sc.makeDeal(ctx, &Deal{Ask: ask, Bid: bid, MinerSig: sig})
}

func (sc *StorageContract) makeDeal(ctx *CallContext, d *Deal) (uint64, error) {
	ask, err := sc.getAsk(ctx, d.Ask)
	if err != nil {
		return 0, errors.Wrap(err, "get ask")
	}

	bid, err := sc.getBid(ctx, d.Bid)
	if err != nil {
		return 0, errors.Wrap(err, "get bid")
	}

	if ask.Size < bid.Size {
		return 0, &revertError{fmt.Errorf("not enough space in ask for bid")}
	}

	// Miner should take care not to sign a deal until they have the data
	if ask.MinerID != d.MinerSig { // make sure signature in deal matches miner ID of the ask
		return 0, &revertError{fmt.Errorf("signature in deal does not match minerID of ask")}
	}

	if ctx.From != bid.Owner {
		return 0, &revertError{fmt.Errorf("cannot create a deal for someone elses bid")}
	}

	id, err := sc.getDeals(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "get deals")
	}

	if err := sc.storeUint64(ctx, "deals", id+1); err != nil {
		return 0, errors.Wrap(err, "store updated deal count")
	}

	data, err := json.Marshal(d)
	if err != nil {
		return 0, err
	}

	// DEBATABLE:
	ask.Size -= bid.Size
	if err := sc.putOrder(ctx, fmt.Sprintf("a%d", d.Ask), ask); err != nil {
		return 0, err
	}

	if err := ctx.ContractState.Set(ctx.Ctx, fmt.Sprintf("d%d", id), data); err != nil {
		return 0, err
	}

	if err := sc.addActiveDeal(ctx, id); err != nil {
		return 0, err
	}

	return id, nil
}

func (sc *StorageContract) createMiner(ctx *CallContext, args []interface{}) (interface{}, error) {
	pledge, err := numberCast(args[0])
	if err != nil {
		return nil, err
	}

	nminer := &MinerContract{
		Owner:         ctx.From,
		Pledge:        pledge,
		LockedStorage: big.NewInt(0),
		Power:         big.NewInt(0),
		s:             ctx.State.NewContractState(),
	}

	if err := nminer.Flush(ctx.Ctx); err != nil {
		return nil, err
	}

	mem, err := nminer.s.Flush(ctx.Ctx)
	if err != nil {
		return nil, err
	}

	ca := newContractAddress(ctx)

	act := &Actor{
		Code:   MinerContractCodeHash,
		Memory: mem,
	}

	if err := ctx.State.SetActor(ctx.Ctx, ca, act); err != nil {
		return nil, err
	}

	return ca, nil
}

func newContractAddress(cctx *CallContext) types.Address {
	b, err := json.Marshal([]interface{}{cctx.From, cctx.FromNonce, cctx.Address})
	if err != nil {
		panic(err)
	}

	h := sha256.Sum256(b)
	return types.Address(h[:20])
}
