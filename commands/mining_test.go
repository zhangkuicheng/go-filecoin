package commands

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/filecoin-project/go-filecoin/types"
	"github.com/stretchr/testify/assert"
)

func parseInt(assert *assert.Assertions, s string) *big.Int {
	i := new(big.Int)
	i, err := i.SetString(strings.TrimSpace(s), 10)
	assert.True(err, "couldn't parse as big.Int %q", s)
	return i
}

func TestMinerGenBlock(t *testing.T) {
	assert := assert.New(t)
	d := NewDaemon(t).Start()
	defer d.ShutdownSuccess()

	t.Log("[success] address in local wallet")
	// TODO: use `config` cmd once it exists
	addr := d.Config().Mining.RewardAddress.String()

	s := d.RunSuccess("wallet", "balance", addr)
	beforeBalance := parseInt(assert, s.ReadStdout())
	d.RunSuccess("mining", "once")
	s = d.RunSuccess("wallet", "balance", addr)
	afterBalance := parseInt(assert, s.ReadStdout())
	sum := new(big.Int)
	fmt.Println(beforeBalance, afterBalance)
	assert.True(sum.Add(beforeBalance, big.NewInt(1000)).Cmp(afterBalance) == 0)
}

func TestMinerForceWinningTicket(t *testing.T) {
	assert := assert.New(t)
	d := NewDaemon(t).Start()
	defer d.ShutdownSuccess()

	mineAndReturnHeight := func() uint64 {
		bh := runSuccessFirstLine(d, "mining once --force-winning-ticket")
		out := d.RunSuccess("--enc=json show block " + bh)
		var b types.Block
		err := json.Unmarshal(out.stdout, &b)
		assert.NoError(err)
		return b.Height
	}

	heightBefore := mineAndReturnHeight()
	heightAfter := mineAndReturnHeight()
	assert.Equal(heightBefore+1, heightAfter)
}
