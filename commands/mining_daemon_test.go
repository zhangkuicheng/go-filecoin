package commands

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/stretchr/testify/assert"
)

func parseInt(assert *assert.Assertions, s string) *big.Int {
	i := new(big.Int)
	i, err := i.SetString(strings.TrimSpace(s), 10)
	assert.True(err, "couldn't parse as big.Int %q", s)
	return i
}

func TestMiningGenBlock(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	d := th.NewDaemon(t, th.GenesisFile(th.GenesisFilePath())).Start()
	defer d.ShutdownSuccess()

	t.Log("[success] address in local wallet")
	// TODO: use `config` cmd once it exists
	addr := th.TestAddress1

	s := d.RunSuccess("wallet", "balance", addr)
	beforeBalance := parseInt(assert, s.ReadStdout())

	s = d.RunSuccess("actor", "ls")
	fmt.Println("LSED AND GOT BACK OUT OF: ")
	fmt.Print(s.ReadStdout())
	fmt.Println("LSED AND GOT BACK ERR OF: ")
	fmt.Print(s.ReadStderr())

	s = d.RunSuccess("mining", "once")
	fmt.Print("MINED AND GOT BACK OUT OF: ", s.ReadStdout())
	fmt.Print("MINED AND GOT BACK ERR OF: ", s.ReadStderr())

	s = d.RunSuccess("actor", "ls")
	fmt.Println("LSED AND GOT BACK OUT OF: ")
	fmt.Print(s.ReadStdout())
	fmt.Println("LSED AND GOT BACK ERR OF: ")
	fmt.Print(s.ReadStderr())

	s = d.RunSuccess("wallet", "balance", addr)
	afterBalance := parseInt(assert, s.ReadStdout())
	sum := new(big.Int)
	assert.Equal(sum.Add(beforeBalance, big.NewInt(1000)), afterBalance)
}
