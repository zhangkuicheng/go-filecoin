package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	testnode "github.com/filecoin-project/go-filecoin/testhelpers/iptbTestWrapper"
)

func TestAddrsNewAndListTestNode(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	tns, err := testnode.NewTestNodes(t, 1)
	assert.NoError(err)

	tn := tns[0]
	tn.MustInit(ctx).MustStart(ctx)
	defer tn.MustStop(ctx)

	var removeAddr addressListResult
	tn.MustRunCmdJSON(ctx, &removeAddr, "go-filecoin", "wallet", "addrs", "ls")

	var newAddr addressResult
	tn.MustRunCmdJSON(ctx, &newAddr, "go-filecoin", "wallet", "addrs", "new")

	var listAddr addressListResult
	tn.MustRunCmdJSON(ctx, &listAddr, "go-filecoin", "wallet", "addrs", "ls")

	// Since everynode starts with an address remove that from the list we compare against
	for i, a := range listAddr.Addresses {
		if a.String() == removeAddr.Addresses[0].String() {
			listAddr.Addresses = append(listAddr.Addresses[:i], listAddr.Addresses[i+1:]...)
		}
	}
	t.Logf("NewAddress: %v", newAddr)
	t.Logf("ListAddress: %v", listAddr)
	assert.Equal(newAddr.Address.Bytes(), listAddr.Addresses[0].Bytes())
}
