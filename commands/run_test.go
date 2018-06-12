package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
)

type TestOutput struct {
	*th.Output
	Test *testing.T
}

func (o *TestOutput) AssertSuccess() *TestOutput {
	o.Test.Helper()
	assert.NoError(o.Test, o.Error)
	oErr := o.ReadStderr()

	assert.Equal(o.Test, o.Code, 0, oErr)
	assert.NotContains(o.Test, oErr, "CRITICAL")
	assert.NotContains(o.Test, oErr, "ERROR")
	assert.NotContains(o.Test, oErr, "WARNING")
	return o

}

func (o *TestOutput) AssertFail(err string) *TestOutput {
	o.Test.Helper()
	assert.NoError(o.Test, o.Error)
	assert.Equal(o.Test, 1, o.Code)
	assert.Empty(o.Test, o.ReadStdout())
	assert.Contains(o.Test, o.ReadStderr(), err)
	return o
}

type TestDaemon struct {
	*th.Daemon
	Test *testing.T
}

// NewTestDaemon needs a comment
func NewTestDaemon(t *testing.T, options ...func(*th.Daemon)) *TestDaemon {
	newDaemon, err := th.NewDaemon(options...)
	require.NoError(t, err)

	return &TestDaemon{
		Daemon: newDaemon,
		Test:   t,
	}
}

// Start needs a comment
func (td *TestDaemon) Start() *TestDaemon {
	_, err := td.Daemon.Start()
	require.NoError(td.Test, err)
	require.NoError(td.Test, td.WaitForAPI(), "Daemon failed to start")
	return td
}

// ShutdownSuccess needs a comment
func (td *TestDaemon) ShutdownSuccess() {
	err := td.Shutdown()
	require.NoError(td.Test, err)

	tdOut := td.ReadStderr()
	require.NoError(td.Test, err, tdOut)
	require.NotContains(td.Test, tdOut, "CRITICAL")
	require.NotContains(td.Test, tdOut, "ERROR")
	require.NotContains(td.Test, tdOut, "WARNING")
}

// ShutdownEasy needs comments
// TODO don't panic be happy
func (td *TestDaemon) ShutdownEasy() {
	err := td.Daemon.Shutdown()
	assert.NoError(td.Test, err)
	tdOut := td.ReadStderr()
	assert.NoError(td.Test, err, tdOut)
	os.RemoveAll(td.RepoDir)
}

func (td *TestDaemon) Run(args ...string) (*TestOutput, error) {
	td.Test.Helper()
	output, err := td.RunWithStdin(nil, args...)
	return &TestOutput{
		Output: output,
		Test:   td.Test,
	}, err
}

func (td *TestDaemon) RunSuccess(args ...string) *TestOutput {
	td.Test.Helper()
	o, err := td.Run(args...)
	assert.NoError(td.Test, err)
	return o.AssertSuccess()
}

func (td *TestDaemon) RunFail(err string, args ...string) *TestOutput {
	td.Test.Helper()
	o, _ := td.Run(args...)
	return o.AssertFail(err)
}

func (td *TestDaemon) ConfigExists(dir string) bool {
	_, err := os.Stat(filepath.Join(td.RepoDir, "config.toml"))
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func (td *TestDaemon) ConnectSuccess(remote *TestDaemon) *TestOutput {
	// Connect the nodes
	remoteAddr, err := remote.GetAddress()
	assert.NoError(td.Test, err)
	out := td.RunSuccess("swarm", "connect", remoteAddr)
	peers1 := td.RunSuccess("swarm", "peers")
	peers2 := remote.RunSuccess("swarm", "peers")

	remoteID, err := remote.GetID()
	assert.NoError(td.Test, err)
	td.Test.Log("[success] 1 -> 2")
	require.Contains(td.Test, peers1.ReadStdout(), remoteID)

	daemonID, err := td.GetID()
	assert.NoError(td.Test, err)
	td.Test.Log("[success] 2 -> 1")
	require.Contains(td.Test, peers2.ReadStdout(), daemonID)

	return out
}

func runSuccessFirstLine(td *TestDaemon, args ...string) string {
	return runSuccessLines(td, args...)[0]
}

func runSuccessLines(td *TestDaemon, args ...string) []string {
	output := td.RunSuccess(args...)
	result := output.ReadStdoutTrimNewlines()
	return strings.Split(result, "\n")
}

func RunInit(opts ...string) ([]byte, error) {
	return RunCommand("init", opts...)
}

func RunCommand(cmd string, opts ...string) ([]byte, error) {
	filecoinBin, err := th.GetFilecoinBinary()
	if err != nil {
		return nil, err
	}

	process := exec.Command(filecoinBin, append([]string{cmd}, opts...)...)
	return process.CombinedOutput()
}

func ConfigExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "config.toml"))
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
