package commands

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
)

func TestId(t *testing.T) {
	assert := assert.New(t)

	d := NewTestDaemon(t).Start()
	defer d.ShutdownSuccess()

	id := d.RunSuccess("id")

	idContent := id.ReadStdout()
	assert.Containsf(idContent, d.SwarmAddr, "default addr")
	assert.Contains(idContent, "ID")

}

func TestIdFormat(t *testing.T) {
	assert := assert.New(t)

	d := NewTestDaemon(t).Start()
	defer d.ShutdownSuccess()

	idContent := d.RunSuccess("id",
		"--format=\"<id>\\t<aver>\\t<pver>\\t<pubkey>\\n<addrs>\"",
	).ReadStdout()

	assert.Contains(idContent, "\t")
	assert.Contains(idContent, "\n")
	assert.Containsf(idContent, d.SwarmAddr, "default addr")
	assert.NotContains(idContent, "ID")
}

func TestPersistId(t *testing.T) {
	assert := assert.New(t)

	// we need to control this
	dir, err := ioutil.TempDir("", "go-fil-test")
	require.NoError(t, err)

	// Start a demon in dir
	d1 := NewTestDaemon(t, th.RepoDir(dir)).Start()

	// get the id and kill it
	id1, err := d1.GetID()
	assert.NoError(err)
	d1.ShutdownSuccess()

	// restart the daemon
	d2 := NewTestDaemon(t, th.RepoDir(dir), th.ShouldInit(false)).Start()

	// get the id and compare to previous
	id2, err := d2.GetID()
	assert.NoError(err)
	d2.ShutdownSuccess()
	t.Logf("d1: %s", d1.ReadStdout())
	t.Logf("d2: %s", d2.ReadStdout())
	assert.Equal(id1, id2)

}
