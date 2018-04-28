package core

import (
	"bytes"
	"context"
	"testing"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmdtiofXbibTe6Day9ii5zjBZpSRm8vhfoerrNuY3sAQ7e/go-hamt-ipld"

	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/types"
)

func TestVMContextStorage(t *testing.T) {
	assert := assert.New(t)
	addrGetter := types.NewAddressForTestGetter()
	ctx := context.Background()

	cst := hamt.NewCborStore()
	state := types.NewEmptyStateTree(cst)

	toActor, err := NewAccountActor(nil)
	assert.NoError(err)
	toAddr := addrGetter()

	assert.NoError(state.SetActor(ctx, toAddr, toActor))

	msg := types.NewMessage(addrGetter(), toAddr, 0, nil, "hello", nil)

	vmCtx := NewVMContext(nil, toActor, msg, state)

	assert.NoError(vmCtx.WriteStorage([]byte("hello")))

	// make sure we can read it back
	toActorBack, err := state.GetActor(ctx, toAddr)
	assert.NoError(err)

	storage := NewVMContext(nil, toActorBack, msg, state).ReadStorage()
	assert.Equal(storage, []byte("hello"))
}

func TestVMContextSendFailures(t *testing.T) {
	actor1 := types.NewActor(nil, types.NewTokenAmount(100))
	actor2 := types.NewActor(nil, types.NewTokenAmount(50))
	newMsg := types.NewMessageForTestGetter()
	newAddress := types.NewAddressForTestGetter()

	t.Run("failure to convert to ABI values results in fault error", func(t *testing.T) {
		assert := assert.New(t)

		var calls []string
		deps := vmContextSendDeps{
			ToValues: func(_ []interface{}) ([]*abi.Value, error) {
				calls = append(calls, "ToValues")
				return nil, errors.New("error")
			},
		}

		ctx := NewVMContext(actor1, actor2, newMsg(), &types.MockStateTree{})

		receipt, err := ctx.send(deps, newAddress(), "foo", nil, []interface{}{})

		assert.Error(err)
		assert.Equal(uint8(1), receipt.ExitCode)
		assert.True(IsFault(err))
		assert.Equal([]string{"ToValues"}, calls)
	})

	t.Run("failure to encode ABI values to byte slice results in revert error", func(t *testing.T) {
		assert := assert.New(t)

		var calls []string
		deps := vmContextSendDeps{
			EncodeValues: func(_ []*abi.Value) ([]byte, error) {
				calls = append(calls, "EncodeValues")
				return nil, errors.New("error")
			},
			ToValues: func(_ []interface{}) ([]*abi.Value, error) {
				calls = append(calls, "ToValues")
				return nil, nil
			},
		}

		ctx := NewVMContext(actor1, actor2, newMsg(), &types.MockStateTree{})

		receipt, err := ctx.send(deps, newAddress(), "foo", nil, []interface{}{})

		assert.Error(err)
		assert.Equal(1, int(receipt.ExitCode))
		assert.True(shouldRevert(err))
		assert.Equal([]string{"ToValues", "EncodeValues"}, calls)
	})

	t.Run("refuse to send a message with identical from/to", func(t *testing.T) {
		assert := assert.New(t)

		to := newAddress()

		msg := newMsg()
		msg.To = to

		var calls []string
		deps := vmContextSendDeps{
			EncodeValues: func(_ []*abi.Value) ([]byte, error) {
				calls = append(calls, "EncodeValues")
				return nil, nil
			},
			ToValues: func(_ []interface{}) ([]*abi.Value, error) {
				calls = append(calls, "ToValues")
				return nil, nil
			},
		}

		ctx := NewVMContext(actor1, actor2, msg, &types.MockStateTree{})

		receipt, err := ctx.send(deps, to, "foo", nil, []interface{}{})

		assert.Error(err)
		assert.Equal(1, int(receipt.ExitCode))
		assert.True(IsFault(err))
		assert.Equal([]string{"ToValues", "EncodeValues"}, calls)
	})

	t.Run("returns a fault error if unable to create or find a recipient actor", func(t *testing.T) {
		assert := assert.New(t)

		var calls []string
		deps := vmContextSendDeps{
			EncodeValues: func(_ []*abi.Value) ([]byte, error) {
				calls = append(calls, "EncodeValues")
				return nil, nil
			},
			GetOrCreateActor: func(_ context.Context, _ types.Address, _ func() (*types.Actor, error)) (*types.Actor, error) {
				calls = append(calls, "GetOrCreateActor")
				return nil, errors.New("error")
			},
			ToValues: func(_ []interface{}) ([]*abi.Value, error) {
				calls = append(calls, "ToValues")
				return nil, nil
			},
		}

		ctx := NewVMContext(actor1, actor2, newMsg(), &types.MockStateTree{})

		receipt, err := ctx.send(deps, newAddress(), "foo", nil, []interface{}{})

		assert.Error(err)
		assert.Equal(1, int(receipt.ExitCode))
		assert.True(IsFault(err))
		assert.Equal([]string{"ToValues", "EncodeValues", "GetOrCreateActor"}, calls)
	})

	t.Run("propagates any error returned from Send", func(t *testing.T) {
		assert := assert.New(t)

		expectedVMSendErr := errors.New("error")

		var calls []string
		deps := vmContextSendDeps{
			EncodeValues: func(_ []*abi.Value) ([]byte, error) {
				calls = append(calls, "EncodeValues")
				return nil, nil
			},
			GetOrCreateActor: func(_ context.Context, _ types.Address, f func() (*types.Actor, error)) (*types.Actor, error) {
				calls = append(calls, "GetOrCreateActor")
				return f()
			},
			Send: func(ctx context.Context, from, to *types.Actor, msg *types.Message, st types.StateTree) (*types.MessageReceipt, error) {
				calls = append(calls, "Send")
				return &types.MessageReceipt{ExitCode: 123}, expectedVMSendErr
			},
			SetActor: func(_ context.Context, _ types.Address, _ *types.Actor) error {
				calls = append(calls, "SetActor")
				return nil
			},
			ToValues: func(_ []interface{}) ([]*abi.Value, error) {
				calls = append(calls, "ToValues")
				return nil, nil
			},
		}

		ctx := NewVMContext(actor1, actor2, newMsg(), &types.MockStateTree{})

		receipt, err := ctx.send(deps, newAddress(), "foo", nil, []interface{}{})

		assert.Error(err)
		assert.Equal(123, int(receipt.ExitCode))
		assert.Equal(expectedVMSendErr, err)
		assert.Equal([]string{"ToValues", "EncodeValues", "GetOrCreateActor", "Send"}, calls)
	})
}

func TestVMContextReturn(t *testing.T) {
	assert := assert.New(t)
	addrGetter := types.NewAddressForTestGetter()
	ctx := context.Background()

	cst := hamt.NewCborStore()
	state := types.NewEmptyStateTree(cst)

	toActor, err := NewAccountActor(nil)
	assert.NoError(err)
	toAddr := addrGetter()

	assert.NoError(state.SetActor(ctx, toAddr, toActor))

	msg := types.NewMessage(addrGetter(), toAddr, 0, nil, "hello", nil)

	vmCtx := NewVMContext(nil, toActor, msg, state)

	// initial write
	assert.NoError(vmCtx.Return([]byte("hello")))
	out := [types.ReturnValueLength]byte{}
	copy(out[:], []byte("hello"))
	assert.Equal(out, vmCtx.returnVal)

	// overwrite with something shorter
	assert.NoError(vmCtx.Return([]byte("foo")))
	out = [types.ReturnValueLength]byte{}
	copy(out[:], []byte("foo"))
	assert.Equal(out, vmCtx.returnVal)

	// overwrite with something filling all of it
	data := bytes.Repeat([]byte{9}, types.ReturnValueLength)
	assert.NoError(vmCtx.Return(data))
	assert.Equal(data, vmCtx.returnVal[:])
}

func TestVMContextRevert(t *testing.T) {
	assert := assert.New(t)
	addrGetter := types.NewAddressForTestGetter()
	ctx := context.Background()

	cst := hamt.NewCborStore()
	state := types.NewEmptyStateTree(cst)

	toActor, err := NewAccountActor(nil)
	assert.NoError(err)
	toAddr := addrGetter()

	assert.NoError(state.SetActor(ctx, toAddr, toActor))

	msg := types.NewMessage(addrGetter(), toAddr, 0, nil, "hello", nil)

	vmCtx := NewVMContext(nil, toActor, msg, state)

	// inivalid exit code of 0
	err = vmCtx.Revert(0, nil)
	assert.Error(err)
	assert.Contains(err.Error(), "invalid exit code")

	// valid exit code
	assert.NoError(vmCtx.Revert(12, nil))
	assert.Equal(uint8(12), vmCtx.exitCode)
}
