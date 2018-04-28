package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/types"
)

// VMContext is the only thing exposed to an actor while executing.
// All methods on the VMContext are ABI methods exposed to actors.
type VMContext struct {
	from    *types.Actor
	to      *types.Actor
	message *types.Message
	state   types.StateTree

	returnVal  types.ReturnValue
	returnSize uint32
	exitCode   uint8
}

// NewVMContext returns an initialized context.
func NewVMContext(from, to *types.Actor, msg *types.Message, st types.StateTree) *VMContext {
	return &VMContext{
		from:    from,
		to:      to,
		message: msg,
		state:   st,
	}
}

// Message retrieves the message associated with this context.
func (ctx *VMContext) Message() *types.Message {
	return ctx.message
}

// ReadStorage reads the storage from the associated to actor.
func (ctx *VMContext) ReadStorage() []byte {
	return ctx.to.ReadStorage()
}

// WriteStorage writes to the storage of the associated to actor.
func (ctx *VMContext) WriteStorage(memory []byte) error {
	ctx.to.WriteStorage(memory)
	return ctx.state.SetActor(context.Background(), ctx.message.To, ctx.to)
}

// Revert sets the current return value and marks the call as failed with an exitCode.
// TODO: use ptr + size once allocations are implemented.
func (ctx *VMContext) Revert(exitCode uint8, ret []byte) error {
	fmt.Printf("revert %d (%s)\n", len(ret), ret)
	if exitCode < 1 {
		return fmt.Errorf("invalid exit code: %d, must be > 0", exitCode)
	}

	if len(ret) > types.ReturnValueLength {
		return fmt.Errorf("return value too large: expected < %d, got %d", types.ReturnValueLength, len(ret))
	}

	ctx.exitCode = exitCode
	ctx.setReturnVal(ret)

	return nil
}

// Return sets the current return value and markrs the call as successfull with
// exitCode `0`.
// Multiple calls will overwrite the past values
// TODO: use ptr + size once allocations are implemented.
func (ctx *VMContext) Return(ret []byte) error {
	fmt.Printf("return %d (%s)\n", len(ret), ret)
	if len(ret) > types.ReturnValueLength {
		return fmt.Errorf("return value too large: expected < %d, got %d", types.ReturnValueLength, len(ret))
	}

	ctx.exitCode = 0
	ctx.setReturnVal(ret)

	return nil
}

func (ctx *VMContext) setReturnVal(ret []byte) {
	copy(ctx.returnVal[:], ret)
	ctx.returnSize = uint32(len(ret))

	for i := len(ret); i < types.ReturnValueLength; i++ {
		ctx.returnVal[i] = 0
	}
}

// Send sends a message to another actor.
// This method assumes to be called from inside the `to` actor.
func (ctx *VMContext) Send(to types.Address, method string, value *types.TokenAmount, params []interface{}) (*types.MessageReceipt, error) {
	deps := vmContextSendDeps{
		EncodeValues:     abi.EncodeValues,
		GetOrCreateActor: ctx.state.GetOrCreateActor,
		Send:             Send,
		SetActor:         ctx.state.SetActor,
		ToValues:         abi.ToValues,
	}

	return ctx.send(deps, to, method, value, params)
}

type vmContextSendDeps struct {
	EncodeValues     func([]*abi.Value) ([]byte, error)
	GetOrCreateActor func(context.Context, types.Address, func() (*types.Actor, error)) (*types.Actor, error)
	Send             func(context.Context, *types.Actor, *types.Actor, *types.Message, types.StateTree) (*types.MessageReceipt, error)
	SetActor         func(context.Context, types.Address, *types.Actor) error
	ToValues         func([]interface{}) ([]*abi.Value, error)
}

// send sends a message to another actor. It exists alongside send so that we can inject its dependencies during test.
func (ctx *VMContext) send(deps vmContextSendDeps, to types.Address, method string, value *types.TokenAmount, params []interface{}) (*types.MessageReceipt, error) {
	// the message sender is the `to` actor, so this is what we set as `from` in the new message
	from := ctx.Message().To
	fromActor := ctx.to

	vals, err := deps.ToValues(params)
	if err != nil {
		return nil, faultErrorWrap(err, "failed to convert inputs to abi values")
	}

	paramData, err := deps.EncodeValues(vals)
	if err != nil {
		return nil, revertErrorWrap(err, "encoding params failed")
	}

	msg := types.NewMessage(from, to, 0, value, method, paramData)
	if msg.From == msg.To {
		// TODO: handle this
		return nil, newFaultErrorf("unhandled: sending to self (%s)", msg.From)
	}

	toActor, err := deps.GetOrCreateActor(context.TODO(), msg.To, func() (*types.Actor, error) {
		return NewAccountActor(nil)
	})
	if err != nil {
		return nil, faultErrorWrapf(err, "failed to get or create To actor %s", msg.To)
	}

	// TODO(fritz) de-dup some of the logic between here and core.Send
	receipt, err := deps.Send(context.Background(), fromActor, toActor, msg, ctx.state)
	if err != nil {
		return nil, err
	}

	if receipt.ExitCode > 0 {
		return receipt, revertErrorWrapf(err, "non zero exit code: %d", receipt.ExitCode)
	}

	return receipt, nil
}

// AddressForNewActor creates computes the address for a new actor in the same
// way that ethereum does.  Note that this will not work if we allow the
// creation of multiple contracts in a given invocation (nonce will remain the
// same, resulting in the same address back)
func (ctx *VMContext) AddressForNewActor() (types.Address, error) {
	return computeActorAddress(ctx.message.From, ctx.from.Nonce)
}

func computeActorAddress(creator types.Address, nonce uint64) (types.Address, error) {
	buf := new(bytes.Buffer)

	if _, err := buf.Write(creator.Bytes()); err != nil {
		return types.Address{}, err
	}

	if err := binary.Write(buf, binary.BigEndian, nonce); err != nil {
		return types.Address{}, err
	}

	hash, err := types.AddressHash(buf.Bytes())
	if err != nil {
		return types.Address{}, err
	}

	return types.NewMainnetAddress(hash), nil
}
