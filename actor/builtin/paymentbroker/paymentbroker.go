package paymentbroker

import (
	cbor "gx/ipfs/QmRiRJhn427YVuufBEHofLreKWNw7P7BWNq86Sb9kzqdbd/go-ipld-cbor"

	"github.com/attic-labs/noms/go/marshal"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/filecoin-project/go-filecoin/vm/errors"

	noms "github.com/attic-labs/noms/go/types"
)

const (
	// ErrNonAccountActor indicates an non-account actor attempted to create a payment channel
	ErrNonAccountActor = 33
	// ErrDuplicateChannel indicates an attempt to create a payment channel with an existing id
	ErrDuplicateChannel = 34
	// ErrEolTooLow indicates an attempt to lower the Eol of a payment channel
	ErrEolTooLow = 35
	// ErrReclaimBeforeEol indicates an attempt to reclaim funds before the eol of the channel
	ErrReclaimBeforeEol = 36
	// ErrInsufficientChannelFunds indicates an attempt to take more funds than the channel contains
	ErrInsufficientChannelFunds = 37
	// ErrUnknownChannel indicates an invalid channel id
	ErrUnknownChannel = 38
	// ErrWrongTarget indicates attempt to redeem from wrong target account
	ErrWrongTarget = 39
	// ErrExpired indicates the block height has exceeded the eol
	ErrExpired = 40
	// ErrAlreadyWithdrawn indicates amount of the voucher has already been withdrawn
	ErrAlreadyWithdrawn = 41
)

// Errors map error codes to revert errors this actor may return
var Errors = map[uint8]error{
	ErrNonAccountActor:          errors.NewCodedRevertError(ErrNonAccountActor, "Only account actors may create payment channels"),
	ErrDuplicateChannel:         errors.NewCodedRevertError(ErrDuplicateChannel, "Duplicate create channel attempt"),
	ErrEolTooLow:                errors.NewCodedRevertError(ErrEolTooLow, "payment channel eol may not be decreased"),
	ErrReclaimBeforeEol:         errors.NewCodedRevertError(ErrReclaimBeforeEol, "payment channel may not reclaimed before eol"),
	ErrInsufficientChannelFunds: errors.NewCodedRevertError(ErrInsufficientChannelFunds, "voucher amount exceeds amount in channel"),
	ErrUnknownChannel:           errors.NewCodedRevertError(ErrUnknownChannel, "payment channel is unknown"),
	ErrWrongTarget:              errors.NewCodedRevertError(ErrWrongTarget, "attempt to redeem channel from wrong target account"),
	ErrExpired:                  errors.NewCodedRevertError(ErrExpired, "block height has exceeded channel's end of life"),
	ErrAlreadyWithdrawn:         errors.NewCodedRevertError(ErrAlreadyWithdrawn, "update amount has already been redeemed"),
}

func init() {
	cbor.RegisterCborType(PaymentChannel{})
	cbor.RegisterCborType(Storage{})
	cbor.RegisterCborType(PaymentVoucher{})
}

// Signature signs an update request
type Signature = []byte

// PaymentChannel records the intent to pay funds to a target account.
type PaymentChannel struct {
	Target         types.Address
	Amount         types.AttoFIL
	AmountRedeemed types.AttoFIL
	Eol            types.BlockHeight `noms:"eol"`
}

// PaymentVoucher is a voucher for a payment channel that can be transferred off-chain but guarantees a future payment.
type PaymentVoucher struct {
	Channel   types.ChannelID `json:"channel"`
	Payer     types.Address   `json:"payer"`
	Target    types.Address   `json:"target"`
	Amount    types.AttoFIL   `json:"amount"`
	Signature Signature       `json:"signature"`
}

// Actor provides a mechanism for off chain payments.
// It allows the creation of payment Channels that hold funds for a target account
// and permits that account to withdraw funds only with a voucher signed by the
// channel's creator.
type Actor struct{}

// Storage is the payment broker's storage
type Storage struct {
	Channels noms.Map
}

// NewStorage returns an empty Storage struct
func (pb *Actor) NewStorage() interface{} {
	return &Storage{}
}

// Exports returns the actor's exports
func (pb *Actor) Exports() exec.Exports {
	return paymentBrokerExports
}

var _ exec.ExecutableActor = (*Actor)(nil)

// NewPaymentBrokerActor returns a new payment broker actor.
func NewPaymentBrokerActor() (*types.Actor, error) {
	vs := actor.NewValueStore()
	initStorage := Storage{
		Channels: noms.NewMap(vs),
	}

	storageBytes, err := actor.MarshalStorageNoms(initStorage, vs)
	if err != nil {
		return nil, err
	}
	return types.NewActorWithMemory(types.PaymentBrokerActorCodeCid, types.NewAttoFILFromFIL(0), storageBytes), nil
}

var paymentBrokerExports = exec.Exports{
	"close": &exec.FunctionSignature{
		Params: []abi.Type{abi.Address, abi.ChannelID, abi.AttoFIL, abi.Bytes},
		Return: nil,
	},
	"createChannel": &exec.FunctionSignature{
		Params: []abi.Type{abi.Address, abi.BlockHeight},
		Return: []abi.Type{abi.ChannelID},
	},
	"extend": &exec.FunctionSignature{
		Params: []abi.Type{abi.ChannelID, abi.BlockHeight},
		Return: nil,
	},
	"ls": &exec.FunctionSignature{
		Params: []abi.Type{abi.Address},
		Return: []abi.Type{abi.Bytes},
	},
	"reclaim": &exec.FunctionSignature{
		Params: []abi.Type{abi.ChannelID},
		Return: nil,
	},
	"update": &exec.FunctionSignature{
		Params: []abi.Type{abi.Address, abi.ChannelID, abi.AttoFIL, abi.Bytes},
		Return: nil,
	},
	"voucher": &exec.FunctionSignature{
		Params: []abi.Type{abi.ChannelID, abi.AttoFIL},
		Return: []abi.Type{abi.Bytes},
	},
}

// CreateChannel creates a new payment channel from the caller to the target.
// The value attached to the invocation is used as the deposit, and the channel
// will expire and return all of its money to the owner after the given block height.
func (pb *Actor) CreateChannel(ctx *vm.Context, target types.Address, eol *types.BlockHeight) (channelID *types.ChannelID, ec uint8, err error) {
	var storage Storage
	err = actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {
		// require that from account be an account actor to ensure nonce is a valid id
		if !ctx.IsFromAccountActor() {
			return nil, Errors[ErrNonAccountActor]
		}

		payer := ctx.Message().From
		channelID = types.NewChannelID(uint64(ctx.Message().Nonce))
		_, err := findChannel(storage, payer, channelID)
		if err == nil {
			return nil, Errors[ErrDuplicateChannel]
		} else if err != Errors[ErrUnknownChannel] {
			return nil, err
		}

		paymentChannel := PaymentChannel{
			Target:         target,
			Amount:         *ctx.Message().Value,
			AmountRedeemed: *types.NewAttoFILFromFIL(0),
			Eol:            *eol,
		}

		insertChannel(vrw, &storage, paymentChannel, payer, channelID)
		return storage, nil
	})
	if err != nil {
		return nil, errors.CodeError(err), err
	}

	return channelID, 0, nil
}

// Update is called by the target account to withdraw funds with authorization from the payer.
// This method is exactly like Close except it doesn't close the channel.
// This is useful when you want to checkpoint the value in a payment, but continue to use the
// channel afterwards. The amt represents the total funds authorized so far, so that subsequent
// calls to Update will only transfer the difference between the given amt and the greatest
// amt taken so far. A series of channel transactions might look like this:
//                                Payer: 2000, Target: 0, Channel: 0
// payer createChannel(1000)   -> Payer: 1000, Target: 0, Channel: 1000
// target Update(100)          -> Payer: 1000, Target: 100, Channel: 900
// target Update(200)          -> Payer: 1000, Target: 200, Channel: 800
// target Close(500)           -> Payer: 1500, Target: 500, Channel: 0
//
func (pb *Actor) Update(ctx *vm.Context, payer types.Address, chid *types.ChannelID, amt *types.AttoFIL, sig Signature) (uint8, error) {
	var storage Storage
	err := actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {

		// TODO: check the signature against the other voucher components.
		channel, err := findChannel(storage, payer, chid)
		if err != nil {
			return nil, err
		}

		err = updateChannel(ctx, ctx.Message().From, &channel, amt)
		insertChannel(vrw, &storage, channel, payer, chid)
		return storage, err
	})
	if err != nil {
		return errors.CodeError(err), err
	}

	return 0, nil
}

func insertChannel(vrw noms.ValueReadWriter, storage *Storage, channel PaymentChannel, payer types.Address, chid *types.ChannelID) {
	var payerChannels noms.Map
	if v, ok := storage.Channels.MaybeGet(noms.String(payer.String())); ok {
		payerChannels = v.(noms.Map)
	} else {
		payerChannels = noms.NewMap(vrw)
	}

	payerChannels = payerChannels.Edit().Set(
		noms.String(chid.String()),
		marshal.MustMarshal(vrw, channel)).Map()
	storage.Channels = storage.Channels.Edit().Set(
		noms.String(payer.String()),
		payerChannels).Map()
}

// Close first executes the logic performed in the the Update method, then returns all
// funds remaining in the channel to the payer account and deletes the channel.
func (pb *Actor) Close(ctx *vm.Context, payer types.Address, chid *types.ChannelID, amt *types.AttoFIL, sig Signature) (uint8, error) {
	var storage Storage
	err := actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {

		// TODO: check the signature against the other voucher components.
		channel, err := findChannel(storage, payer, chid)
		if err != nil {
			return nil, err
		}

		err = updateChannel(ctx, ctx.Message().From, &channel, amt)
		if err != nil {
			return nil, err
		}

		// return funds to payer
		err = reclaim(ctx, &storage, payer, chid, &channel)
		return channel, err
	})
	if err != nil {
		return errors.CodeError(err), err
	}

	return 0, nil
}

// Extend can be used by the owner of a channel to add more funds to it and
// extend the Channels lifespan.
func (pb *Actor) Extend(ctx *vm.Context, chid *types.ChannelID, eol *types.BlockHeight) (uint8, error) {
	var storage Storage
	err := actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {
		channel, err := findChannel(storage, ctx.Message().From, chid)
		if err != nil {
			return nil, err
		}

		// eol can only be increased
		if channel.Eol.GreaterThan(eol) {
			return nil, Errors[ErrEolTooLow]
		}

		// set new eol
		channel.Eol = *eol

		// increment the value
		channel.Amount = *channel.Amount.Add(ctx.Message().Value)

		// return funds to payer
		return channel, err
	})
	if err != nil {
		return errors.CodeError(err), err
	}

	return 0, nil
}

// Reclaim is used by the owner of a channel to reclaim unspent funds in timed
// out payment Channels they own.
func (pb *Actor) Reclaim(ctx *vm.Context, chid *types.ChannelID) (uint8, error) {
	var storage Storage
	err := actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {
		channel, err := findChannel(storage, ctx.Message().From, chid)
		if err != nil {
			return nil, err
		}

		// reclaim may only be called at or after Eol
		if ctx.BlockHeight().LessThan(&channel.Eol) {
			return nil, Errors[ErrReclaimBeforeEol]
		}

		// return funds to payer
		err = reclaim(ctx, &storage, ctx.Message().From, chid, &channel)
		return channel, err
	})
	if err != nil {
		return errors.CodeError(err), err
	}

	return 0, nil
}

// Voucher takes a channel id and amount creates a new unsigned PaymentVoucher against the given channel.
// It errors if the channel doesn't exist or contains less than request amount.
func (pb *Actor) Voucher(ctx *vm.Context, chid *types.ChannelID, amount *types.AttoFIL) (ret []byte, ec uint8, err error) {
	var storage Storage
	err = actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {
		channel, err := findChannel(storage, ctx.Message().From, chid)
		if err != nil {
			return nil, err
		}

		// voucher must be for less than total amount in channel
		if channel.Amount.LessThan(amount) {
			return nil, Errors[ErrInsufficientChannelFunds]
		}

		// return voucher
		voucher := PaymentVoucher{
			Channel: *chid,
			Payer:   ctx.Message().From,
			Target:  channel.Target,
			Amount:  *amount,
		}

		ret, err = cbor.DumpObject(voucher)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		return nil, errors.CodeError(err), err
	}

	return ret, 0, nil
}

// Ls returns all payment channels for a given payer address.
// The slice of channels will be returned as cbor encoded map from string channelId to PaymentChannel.
func (pb *Actor) Ls(ctx *vm.Context, payer types.Address) (ret []byte, ec uint8, err error) {
	var storage Storage
	err = actor.WithStorageNoms(ctx, &storage, func(vrw noms.ValueReadWriter) (interface{}, error) {
		var byPayer = map[string]*PaymentChannel{}
		v, found := storage.Channels.MaybeGet(noms.String(payer.String()))
		if found {
			marshal.MustUnmarshal(v, &byPayer)
		}
		ret, err = cbor.DumpObject(byPayer)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return nil, errors.CodeError(err), err
	}

	return ret, 0, nil
}

func findChannel(storage Storage, payer types.Address, chid *types.ChannelID) (ret PaymentChannel, err error) {
	actorsChannels, found := storage.Channels.MaybeGet(noms.String(payer.String()))
	if !found {
		return ret, Errors[ErrUnknownChannel]
	}

	channel, found := actorsChannels.(noms.Map).MaybeGet(noms.String(chid.String()))
	if !found {
		return ret, Errors[ErrUnknownChannel]
	}

	marshal.MustUnmarshal(channel, &ret)
	return ret, nil
}

func updateChannel(ctx *vm.Context, target types.Address, channel *PaymentChannel, amt *types.AttoFIL) error {
	if target != channel.Target {
		return Errors[ErrWrongTarget]
	}

	if ctx.BlockHeight().GreaterEqual(&channel.Eol) {
		return Errors[ErrExpired]
	}

	if amt.GreaterThan(&channel.Amount) {
		return Errors[ErrInsufficientChannelFunds]
	}

	if amt.LessEqual(&channel.AmountRedeemed) {
		return Errors[ErrAlreadyWithdrawn]
	}

	// transfer funds to sender
	updateAmount := amt.Sub(&channel.AmountRedeemed)
	_, _, err := ctx.Send(ctx.Message().From, "", updateAmount, nil)
	if err != nil {
		return err
	}

	// update amount redeemed from this channel
	channel.AmountRedeemed = *amt

	return nil
}

func reclaim(ctx *vm.Context, storage *Storage, payer types.Address, chid *types.ChannelID, channel *PaymentChannel) error {
	amt := channel.Amount.Sub(&channel.AmountRedeemed)
	if amt.LessEqual(types.ZeroAttoFIL) {
		return nil
	}

	// clean up
	var actorsChannels noms.Map
	if v, found := storage.Channels.MaybeGet(noms.String(payer.String())); found {
		actorsChannels = v.(noms.Map)
	} else {
		return errors.NewRevertError("unexpected error closing channel")
	}

	actorsChannels = actorsChannels.Edit().Remove(noms.String(chid.String())).Map()
	if !actorsChannels.Empty() {
		storage.Channels = storage.Channels.Edit().Set(noms.String(payer.String()), actorsChannels).Map()
	} else {
		storage.Channels = storage.Channels.Edit().Remove(noms.String(payer.String())).Map()
	}

	// send funds
	_, _, err := ctx.Send(payer, "", amt, nil)
	if err != nil {
		return errors.RevertErrorWrap(err, "could not send update funds")
	}

	return nil
}
