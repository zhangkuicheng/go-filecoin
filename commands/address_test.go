package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/stretchr/testify/assert"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
)

func TestAddrsNewAndList(t *testing.T) {
	assert := assert.New(t)

	d := NewTestDaemon(t).Start()
	defer d.ShutdownSuccess()

	var err error
	addrs := make([]string, 10)
	for i := 0; i < 10; i++ {
		addrs[i], err = d.CreateWalletAddr()
		assert.NoError(err)
	}

	list := d.RunSuccess("wallet", "addrs", "ls").ReadStdout()
	for _, addr := range addrs {
		assert.Contains(list, addr)
	}
}

func TestWalletBalance(t *testing.T) {
	assert := assert.New(t)

	d := NewTestDaemon(t).Start()
	defer d.ShutdownSuccess()
	addr, err := d.CreateWalletAddr()
	assert.NoError(err)

	t.Log("[success] not found, zero")
	balance := d.RunSuccess("wallet", "balance", addr)
	assert.Equal("0", balance.ReadStdoutTrimNewlines())

	t.Log("[success] balance 10000000")
	balance = d.RunSuccess("wallet", "balance", address.NetworkAddress.String())
	assert.Equal("10000000", balance.ReadStdoutTrimNewlines())

	t.Log("[success] newly generated one")
	addrNew := d.RunSuccess("wallet addrs new")
	balance = d.RunSuccess("wallet", "balance", addrNew.ReadStdoutTrimNewlines())
	assert.Equal("0", balance.ReadStdoutTrimNewlines())
}

func TestAddrsLookup(t *testing.T) {
	assert := assert.New(t)

	//Define 2 nodes, each with an address
	d1 := NewTestDaemon(t, th.SwarmAddr("/ip4/127.0.0.1/tcp/6000")).Start()
	defer d1.ShutdownSuccess()
	d1.CreateWalletAddr()

	d2 := NewTestDaemon(t, th.SwarmAddr("/ip4/127.0.0.1/tcp/6001")).Start()
	defer d2.ShutdownSuccess()
	d2.CreateWalletAddr()

	//Connect daemons
	d1.ConnectSuccess(d2)

	d1Raw := d1.RunSuccess("address ls")
	d1Addrs := strings.Split(strings.Trim(d1Raw.ReadStdout(), "\n"), "\n")
	d1WalletAddr := d1Addrs[len(d1Addrs)-1]
	t.Logf("D1 Wallet Address: %s", d1WalletAddr)
	assert.NotEmpty(d1WalletAddr)

	d2Raw := d2.RunSuccess("address ls")
	d2Addrs := strings.Split(strings.Trim(d2Raw.ReadStdout(), "\n"), "\n")
	d2WalletAddr := d2Addrs[len(d2Addrs)-1]
	t.Logf("D2 Wallet Address: %s", d2WalletAddr)
	assert.NotEmpty(d2WalletAddr)

	isD2IdRaw := d1.RunSuccess(fmt.Sprintf("address lookup %s", d2WalletAddr))
	isD1IdRaw := d2.RunSuccess(fmt.Sprintf("address lookup %s", d1WalletAddr))

	isD1Id := strings.Trim(isD1IdRaw.ReadStdout(), "\n")
	isD2Id := strings.Trim(isD2IdRaw.ReadStdout(), "\n")

	d1ID, err := d1.GetID()
	assert.NoError(err)
	d2ID, err := d2.GetID()
	assert.NoError(err)

	assert.Equal(d1ID, isD1Id)
	assert.Equal(d2ID, isD2Id)
}
