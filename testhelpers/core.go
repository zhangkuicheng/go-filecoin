package testhelpers

import (
	"context"

	"gx/ipfs/QmSkuaNgyGmV8c1L3cZNWcUxRJV6J3nsD96JVQPcWcwtyW/go-hamt-ipld"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	"gx/ipfs/QmcD7SqfyQyA91TZUQ7VPRYbGarxmY7EsQewVYMuN5LNSv/go-ipfs-blockstore"
	"gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	"gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/actor/builtin/account"
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/filnet"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/stretchr/testify/require"
)

// RequireMakeStateTree takes a map of addresses to actors and stores them on
// the state tree, requiring that all its steps succeed.
func RequireMakeStateTree(require *require.Assertions, cst *hamt.CborIpldStore, acts map[types.Address]*types.Actor) (*cid.Cid, state.Tree) {
	ctx := context.Background()
	t := state.NewEmptyStateTreeWithActors(cst, builtin.Actors)

	for addr, act := range acts {
		err := t.SetActor(ctx, addr, act)
		require.NoError(err)
	}

	c, err := t.Flush(ctx)
	require.NoError(err)

	return c, t
}

// RequireNewEmptyActor creates a new empty actor with the given starting
// value and requires that its steps succeed.
func RequireNewEmptyActor(require *require.Assertions, value *types.AttoFIL) *types.Actor {
	return &types.Actor{Balance: value}
}

// RequireNewAccountActor creates a new account actor with the given starting
// value and requires that its steps succeed.
func RequireNewAccountActor(require *require.Assertions, value *types.AttoFIL) *types.Actor {
	act, err := account.NewActor(value)
	require.NoError(err)
	return act
}

// RequireNewMinerActor creates a new miner actor with the given owner, pledge, and collateral,
// and requires that its steps succeed.
func RequireNewMinerActor(require *require.Assertions, vms vm.StorageMap, addr types.Address, owner types.Address, key []byte, pledge *types.BytesAmount, pid peer.ID, coll *types.AttoFIL) *types.Actor {
	act := types.NewActor(types.MinerActorCodeCid, types.NewZeroAttoFIL())
	storage := vms.NewStorage(addr, act)
	initializerData := miner.NewState(owner, key, pledge, pid, coll)
	err := (&miner.Actor{}).InitializeState(storage, initializerData)
	require.NoError(storage.Flush())
	require.NoError(err)
	return act
}

// RequireNewFakeActor instantiates and returns a new fake actor and requires
// that its steps succeed.
func RequireNewFakeActor(require *require.Assertions, vms vm.StorageMap, addr types.Address, codeCid *cid.Cid) *types.Actor {
	return RequireNewFakeActorWithTokens(require, vms, addr, codeCid, types.NewAttoFILFromFIL(100))
}

// RequireNewFakeActorWithTokens instantiates and returns a new fake actor and requires
// that its steps succeed.
func RequireNewFakeActorWithTokens(require *require.Assertions, vms vm.StorageMap, addr types.Address, codeCid *cid.Cid, amt *types.AttoFIL) *types.Actor {
	act := types.NewActor(codeCid, amt)
	store := vms.NewStorage(addr, act)
	err := (&actor.FakeActor{}).InitializeState(store, &actor.FakeActorStorage{})
	require.NoError(err)
	require.NoError(vms.Flush())
	return act
}

// RequireRandomPeerID returns a new libp2p peer ID or panics.
func RequireRandomPeerID() peer.ID {
	pid, err := filnet.RandPeerID()
	if err != nil {
		panic(err)
	}

	return pid
}

// VMStorage creates a new storage object backed by an in memory datastore
func VMStorage() vm.StorageMap {
	return vm.NewStorageMap(blockstore.NewBlockstore(datastore.NewMapDatastore()))
}

// MustSign signs a given address with the provided mocksigner or panics if it
// cannot.
func MustSign(s types.MockSigner, msgs ...*types.Message) []*types.SignedMessage {
	var smsgs []*types.SignedMessage
	for _, m := range msgs {
		sm, err := types.NewSignedMessage(*m, &s)
		if err != nil {
			panic(err)
		}
		smsgs = append(smsgs, sm)
	}
	return smsgs
}

// // MustGetNonce returns the next nonce for an actor at the given address or panics.
// func MustGetNonce(st state.Tree, a types.Address) uint64 {
// 	mp := NewMessagePool()
// 	nonce, err := NextNonce(context.Background(), st, mp, a)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return nonce
// }

// // MustConvertParams abi encodes the given parameters into a byte array (or panics)
// func MustConvertParams(params ...interface{}) []byte {
// 	vals, err := abi.ToValues(params)
// 	if err != nil {
// 		panic(err)
// 	}

// 	out, err := abi.EncodeValues(vals)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return out
// }

// // MustPut stores the thingy in the store or panics if it cannot.
// func MustPut(store *hamt.CborIpldStore, thingy interface{}) *cid.Cid {
// 	cid, err := store.Put(context.Background(), thingy)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return cid
// }

// // MustDecodeCid decodes a string to a Cid pointer, panicking on error
// func MustDecodeCid(cidStr string) *cid.Cid {
// 	decode, err := cid.Decode(cidStr)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return decode
// }
