package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

<<<<<<< HEAD
	"gx/ipfs/QmQZadYTDF4ud9DdK85PH2vReJRzUM9YfVW4ReB1q2m51p/go-hamt-ipld"

	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
=======
	//	"gx/ipfs/QmQZadYTDF4ud9DdK85PH2vReJRzUM9YfVW4ReB1q2m51p/go-hamt-ipld"
	//	"gx/ipfs/QmcmpX42gtDv1fz24kau4wjS9hfwWj5VexWBKgGnWzsyag/go-ipfs-blockstore"
	"gx/ipfs/QmVG5gxteQNEMhrS8prJSmU2C9rebtFuTd3SYZ5kE3YZ5k/go-datastore"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/address"
	//	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/state"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	//	"github.com/filecoin-project/go-filecoin/types"
>>>>>>> The chain refactor -- no commit message can tell of the horror
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFakeChain(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	var length = 9
	ctx := context.Background()
	r := repo.NewInMemoryRepo()
	chainStore, _, _, _, err := build(ctx, r)
	require.NoError(err)

	assert.NoError(fake(ctx, length, false, chainStore))
	h, err := chainStore.Head().Height()
	assert.NoError(err)
	assert.Equal(9, h)
}

<<<<<<< HEAD
=======
func TestAddActors(t *testing.T) {
	if !*nerfTests {
		t.SkipNow()
	}
	assert := assert.New(t)
	require := require.New(t)

	ctx := context.Background()

	r := repo.NewInMemoryRepo()
	chainStore, bs, cst, con, err := build(ctx, r)
	require.NoError(err)

	bts := chainStore.Head()
	require.NotNil(bts)
	tsas, err := chainStore.GetTipSetAndState(ctx, bts.String())
	require.NoError(err)
	st, err := state.LoadStateTree(ctx, cst, tsas.TipSetStateRoot, builtin.Actors)
	require.NoError(err)

	_, allActors := state.GetAllActors(st)
	initialActors := len(allActors)

	err = fakeActors(ctx, cst, chainStore, con, bs, bts)
	assert.NoError(err)

	bts = chainStore.Head()
	require.NotNil(bts)
	tsas, err = chainStore.GetTipSetAndState(ctx, bts.String())
	require.NoError(err)
	st, err = state.LoadStateTree(ctx, cst, tsas.TipSetStateRoot, builtin.Actors)
	require.NoError(err)

	_, allActors = state.GetAllActors(st)
	assert.Equal(initialActors+2, len(allActors), "add a account and miner actors")

	sma, err := st.GetActor(ctx, address.StorageMarketAddress)
	require.NoError(err)

	var storageMkt storagemarket.State
	chunk, err := r.Datastore().Get(datastore.NewKey(sma.Head.KeyString()))
	require.NoError(err)
	chunkBytes, ok := chunk.([]byte)
	require.True(ok)
	err = actor.UnmarshalStorage(chunkBytes, &storageMkt)
	require.NoError(err)

	assert.Equal(1, len(storageMkt.Miners))
	assert.Equal(1, len(storageMkt.Orderbook.StorageAsks))
	assert.Equal(1, len(storageMkt.Orderbook.Bids))
}

>>>>>>> The chain refactor -- no commit message can tell of the horror
func GetFakecoinBinary() (string, error) {
	bin := filepath.FromSlash(fmt.Sprintf("%s/src/github.com/filecoin-project/go-filecoin/tools/go-fakecoin/go-fakecoin", os.Getenv("GOPATH")))
	_, err := os.Stat(bin)
	if err == nil {
		return bin, nil
	}

	if os.IsNotExist(err) {
		return "", fmt.Errorf("You are missing the fakecoin binary...try building, searched in '%s'", bin)
	}

	return "", err
}

var testRepoPath = filepath.FromSlash("/tmp/fakecoin/")

func TestCommandsSucceed(t *testing.T) {
	t.Skip("TODO: flaky test")
	require := require.New(t)

	os.RemoveAll(testRepoPath)       // go-filecoin init will fail if repo exists.
	defer os.RemoveAll(testRepoPath) // clean up when we're done.

	bin, err := GetFakecoinBinary()
	require.NoError(err)

	// 'go-fakecoin actors' completes without error. (runs init, so must be the first)
	cmdActors := exec.Command(bin, "actors", "-repodir", testRepoPath)
	out, err := cmdActors.CombinedOutput()
	require.NoError(err, string(out))

	// 'go-fakecoin fake' completes without error.
	cmdFake := exec.Command(bin, "fake", "-repodir", testRepoPath)
	out, err = cmdFake.CombinedOutput()

	require.NoError(err, string(out))
}
