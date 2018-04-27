package core

import (
	"context"

	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/filecoin-project/go-filecoin/types"
)

// Send executes a message pass inside the VM. If error is set it
// will always satisfy either ShouldRevert() or IsFault().
func Send(ctx context.Context, from, to *types.Actor, msg *types.Message, st types.StateTree) (*types.MessageReceipt, error) {
	deps := sendDeps{
		transfer: transfer,
		LoadCode: LoadCode,
	}

	return send(ctx, deps, from, to, msg, st)
}

type sendDeps struct {
	transfer func(*types.Actor, *types.Actor, *types.TokenAmount) error
	LoadCode func(*cid.Cid) (ExecutableActor, error)
}

// send executes a message pass inside the VM. It exists alongside Send so that we can inject its dependencies during test.
func send(ctx context.Context, deps sendDeps, from, to *types.Actor, msg *types.Message, st types.StateTree) (*types.MessageReceipt, error) {
	vmCtx := NewVMContext(from, to, msg, st)

	c, err := msg.Cid()
	if err != nil {
		return nil, faultErrorWrap(err, "could not generate cid for message")
	}

	if msg.Value != nil {
		if err := deps.transfer(from, to, msg.Value); err != nil {
			return nil, err
		}
	}

	// save balance changes
	if err := st.SetActor(ctx, msg.From, from); err != nil {
		return nil, faultErrorWrap(err, "could not set from actor after send")
	}
	if err := st.SetActor(ctx, msg.To, to); err != nil {
		return nil, faultErrorWrap(err, "could not set to actor after send")
	}

	if msg.Method == "" {
		// if only tokens are transferred there is no need for a method
		// this means we can shortcircuit execution
		return types.NewMessageReceipt(c, 0, types.ReturnValue{}), nil
	}

	toExecutable, err := deps.LoadCode(to.Code)
	if err != nil {
		return nil, faultErrorWrap(err, "unable to load code for To actor")
	}

	if !toExecutable.Exports().Has(msg.Method) {
		return nil, newRevertErrorf("missing export: %s", msg.Method)
	}

	err = MakeTypedExport(toExecutable, msg.Method)(vmCtx)
	if err != nil {
		return nil, faultErrorWrapf(err, "failed to execute method")
	}

	return types.NewMessageReceipt(c, vmCtx.exitCode, vmCtx.returnVal), nil
}

func transfer(fromActor, toActor *types.Actor, value *types.TokenAmount) error {
	if value.IsNegative() {
		return ErrCannotTransferNegativeValue
	}

	if fromActor.Balance.LessThan(value) {
		return ErrInsufficientBalance
	}

	if toActor.Balance == nil {
		toActor.Balance = types.ZeroToken // This would be unsafe if TokenAmount could be mutated.
	}
	fromActor.Balance = fromActor.Balance.Sub(value)
	toActor.Balance = toActor.Balance.Add(value)

	return nil
}
