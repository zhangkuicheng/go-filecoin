package testhelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/filecoin-project/go-filecoin/config"
	"github.com/filecoin-project/go-filecoin/types"
)

// Daemon is a daemon
type Daemon struct {
	CmdAddr   string
	SwarmAddr string
	RepoDir   string
	Init      bool

	// The filecoin daemon process
	process *exec.Cmd

	lk     sync.Mutex
	Stdin  io.Writer
	Stdout io.Reader
	Stderr io.Reader
}

// NewDaemon makes a new daemon
func NewDaemon(options ...func(*Daemon)) (*Daemon, error) {
	// Ensure we have the actual binary
	filecoinBin, err := GetFilecoinBinary()
	if err != nil {
		return nil, err
	}

	//Ask the kernel for a port to avoid conflicts
	cmdPort, err := GetFreePort()
	if err != nil {
		return nil, err
	}
	swarmPort, err := GetFreePort()
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "go-fil-test")
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		CmdAddr:   fmt.Sprintf(":%d", cmdPort),
		SwarmAddr: fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", swarmPort),
		RepoDir:   dir,
		Init:      true, // we want to init unless told otherwise
	}

	// configure TestDaemon options
	for _, option := range options {
		option(d)
	}

	repodirFlag := fmt.Sprintf("--repodir=%s", d.RepoDir)
	if d.Init {
		out, err := runInit(repodirFlag)
		if err != nil {
			d.Log(string(out))
			return nil, err
		}
	}

	// define filecoin daemon process
	d.process = exec.Command(filecoinBin, "daemon",
		fmt.Sprintf("--repodir=%s", d.RepoDir),
		fmt.Sprintf("--cmdapiaddr=%s", d.CmdAddr),
		fmt.Sprintf("--swarmlisten=%s", d.SwarmAddr),
	)

	// setup process pipes
	d.Stdout, err = d.process.StdoutPipe()
	if err != nil {
		return nil, err
	}
	d.Stderr, err = d.process.StderrPipe()
	if err != nil {
		return nil, err
	}
	d.Stdin, err = d.process.StdinPipe()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func runInit(opts ...string) ([]byte, error) {
	return runCommand("init", opts...)
}

func runCommand(cmd string, opts ...string) ([]byte, error) {
	filecoinBin, err := GetFilecoinBinary()
	if err != nil {
		return nil, err
	}

	process := exec.Command(filecoinBin, append([]string{cmd}, opts...)...)
	return process.CombinedOutput()
}

// Logf is a daemon logger
// TODO print the daemon api like `Log` see below
func (d *Daemon) Logf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

// Log is a daemon logger
func (d *Daemon) Log(msg ...string) {
	fmt.Printf("[%s]\t %s", d.CmdAddr, msg)
}

// Run runs commands on the daemon
func (d *Daemon) Run(args ...string) (*Output, error) {
	return d.RunWithStdin(nil, args...)
}

// RunWithStdin runs things with stdin
func (d *Daemon) RunWithStdin(stdin io.Reader, args ...string) (*Output, error) {
	bin, err := GetFilecoinBinary()
	if err != nil {
		return nil, err
	}

	// handle Run("cmd subcmd")
	if len(args) == 1 {
		args = strings.Split(args[0], " ")
	}

	finalArgs := append(args, "--repodir="+d.RepoDir, "--cmdapiaddr="+d.CmdAddr)

	d.Logf("run: %q", strings.Join(finalArgs, " "))
	cmd := exec.Command(bin, finalArgs...)

	if stdin != nil {
		cmd.Stdin = stdin
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	o := &Output{
		Args:   args,
		Stdout: stdout,
		stdout: stdoutBytes,
		Stderr: stderr,
		stderr: stderrBytes,
	}

	err = cmd.Wait()

	switch err := err.(type) {
	case *exec.ExitError:
		// TODO: its non-trivial to get the 'exit code' cross platform...
		o.Code = 1
	default:
		o.Error = err
	case nil:
		// okay
	}

	return o, nil
}

// GetID gets the peerid of the daemon
// TODO don't panic be happy
func (d *Daemon) GetID() string {
	out, err := d.Run("id")
	if err != nil {
		panic(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out.ReadStdout()), &parsed); err != nil {
		panic(err)
	}

	return parsed["ID"].(string)
}

// GetAddress gets the libp2p address of the daemon
// TODO don't panic be happy
func (d *Daemon) GetAddress() string {
	out, err := d.Run("id")
	if err != nil {
		panic(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out.ReadStdout()), &parsed); err != nil {
		panic(err)
	}

	adders := parsed["Addresses"].([]interface{})
	return adders[0].(string)
}

// ConnectSuccess connects 2 daemons and pacnis if it fails
// TODO don't panic be happy
func (d *Daemon) ConnectSuccess(remote *Daemon) *Output {
	// Connect the nodes
	out, err := d.Run("swarm", "connect", remote.GetAddress())
	if err != nil {
		panic(err)
	}
	peers1, err := d.Run("swarm", "peers")
	if err != nil {
		panic(err)
	}
	peers2, err := remote.Run("swarm", "peers")
	if err != nil {
		panic(err)
	}

	if !strings.Contains(peers1.ReadStdout(), remote.GetID()) {
		panic("failed to connect (2->1)")
	}
	if !strings.Contains(peers2.ReadStdout(), d.GetID()) {
		panic("failed to connect (1->2)")
	}

	return out
}

// ReadStdout reads that
func (d *Daemon) ReadStdout() string {
	d.lk.Lock()
	defer d.lk.Unlock()
	out, err := ioutil.ReadAll(d.Stdout)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// ReadStderr reads that
func (d *Daemon) ReadStderr() string {
	d.lk.Lock()
	defer d.lk.Unlock()
	out, err := ioutil.ReadAll(d.Stderr)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// Start starts the daemon process
func (d *Daemon) Start() (*Daemon, error) {
	if err := d.process.Start(); err != nil {
		return nil, err
	}
	if err := d.WaitForAPI(); err != nil {
		return nil, err
	}
	return d, nil
}

// Shutdown suts things down
// TODO don't panic be happy
func (d *Daemon) Shutdown() {
	if err := d.process.Process.Signal(syscall.SIGTERM); err != nil {
		d.Logf("Daemon Stderr:\n%s", d.ReadStderr())
		d.Logf("Failed to kill daemon %s", err)
		panic(err)
	}

	if d.RepoDir == "" {
		panic("testdaemon had no repodir set")
	}

	_ = os.RemoveAll(d.RepoDir)
}

// ShutdownSuccess needs comments
// TODO don't panic be happy
func (d *Daemon) ShutdownSuccess() {
	if err := d.process.Process.Signal(syscall.SIGTERM); err != nil {
		panic(err)
	}
	dOut := d.ReadStderr()
	if strings.Contains(dOut, "ERROR") {
		panic("Daemon has error messages")
	}
}

// ShutdownEasy needs comments
// TODO don't panic be happy
func (d *Daemon) ShutdownEasy() {
	if err := d.process.Process.Signal(syscall.SIGINT); err != nil {
		panic(err)
	}
	dOut := d.ReadStderr()
	if strings.Contains(dOut, "ERROR") {
		d.Log("Daemon has error messages")
	}
}

// WaitForAPI waits for the daemon to be running by hitting the http endpoint
func (d *Daemon) WaitForAPI() error {
	for i := 0; i < 100; i++ {
		err := TryAPICheck(d)
		if err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 100)
	}
	return fmt.Errorf("filecoin node failed to come online in given time period (20 seconds)")
}

// CreateMinerAddr issues a new message to the network, mines the message
// and returns the address of the new miner
// equivalent to:
//     `go-filecoin miner create --from $TEST_ACCOUNT 100000 20`
// TODO don't panic be happy
func (d *Daemon) CreateMinerAddr() types.Address {
	// need money
	_, err := d.Run("mining", "once")
	if err != nil {
		panic(err)
	}

	addr := d.Config().Mining.RewardAddress

	var wg sync.WaitGroup
	var minerAddr types.Address

	wg.Add(1)
	go func() {
		miner, err := d.Run("miner", "create", "--from", addr.String(), "1000000", "1000")
		if err != nil {
			panic(err)
		}
		addr, err := types.NewAddressFromString(strings.Trim(miner.ReadStdout(), "\n"))
		if err != nil {
			panic(err)
		}
		if addr.Empty() {
			panic("got back empty address")
		}
		minerAddr = addr
		wg.Done()
	}()

	_, err = d.Run("mining", "once")
	if err != nil {
		panic(err)
	}

	wg.Wait()
	return minerAddr
}

// CreateWalletAddr adds a new address to the daemons wallet and
// returns it.
// equivalent to:
//     `go-filecoin wallet addrs new`
// TODO don't panic be happy
func (d *Daemon) CreateWalletAddr() string {
	outNew, err := d.Run("wallet", "addrs", "new")
	if err != nil {
		panic(err)
	}
	addr := strings.Trim(outNew.ReadStdout(), "\n")
	if addr == "" {
		panic("address is empty")
	}
	return addr
}

// Config is a helper to read out the config of the deamon
// TODO don't panic be happy
func (d *Daemon) Config() *config.Config {
	cfg, err := config.ReadFile(filepath.Join(d.RepoDir, "config.toml"))
	if err != nil {
		panic(err)
	}
	return cfg
}

// MineAndPropagate mines a block and ensure the block has propogated to all `peers`
// by comparing the current head block of `d` with the head block of each peer in `peers`
// TODO don't panic be happy
func (d *Daemon) MineAndPropagate(wait time.Duration, peers ...*Daemon) {
	_, err := d.Run("mining", "once")
	if err != nil {
		panic(err)
	}
	// short circuit
	if peers == nil {
		return
	}
	// ensure all peers have same chain head as `d`
	d.MustHaveChainHeadBy(wait, peers)
}

// MustHaveChainHeadBy ensures all `peers` have the same chain head as `d`, by
// duration `wait`
func (d *Daemon) MustHaveChainHeadBy(wait time.Duration, peers []*Daemon) {
	// will signal all nodes have completed check
	done := make(chan struct{})
	var wg sync.WaitGroup

	expHead := d.GetChainHead()

	for _, p := range peers {
		wg.Add(1)
		go func(p *Daemon) {
			for {
				actHead := p.GetChainHead()
				if expHead.Cid().Equals(actHead.Cid()) {
					wg.Done()
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}(p)
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return
	case <-time.After(wait):
		// TODO don't panic be happy
		panic("Timeout waiting for chains to sync")
	}
}

// GetChainHead returns the head block from `d`
// TODO don't panic be happy
func (d *Daemon) GetChainHead() types.Block {
	out, err := d.Run("chain", "ls", "--enc=json")
	if err != nil {
		panic(err)
	}
	bc := d.MustUnmarshalChain(out.ReadStdout())
	return bc[0]
}

// MustUnmarshalChain unmarshals the chain from `input` into a slice of blocks
// TODO don't panic be happy
func (d *Daemon) MustUnmarshalChain(input string) []types.Block {
	chain := strings.Trim(input, "\n")
	var bs []types.Block

	for _, line := range bytes.Split([]byte(chain), []byte{'\n'}) {
		var b types.Block
		if err := json.Unmarshal(line, &b); err != nil {
			panic(err)
		}
		bs = append(bs, b)
	}

	return bs
}

// MakeMoney mines a block and receives the block reward
// TODO don't panic be happy
func (d *Daemon) MakeMoney(rewards int) {
	for i := 0; i < rewards; i++ {
		d.MineAndPropagate(time.Second * 1)
	}
}

// MakeDeal will make a deal with the miner `miner`, using data `dealData`.
// MakeDeal will return the cid of `dealData`
// TODO don't panic be happy
func (d *Daemon) MakeDeal(dealData string, miner *Daemon) string {

	// The daemons need 2 monies each.
	d.MakeMoney(2)
	miner.MakeMoney(2)

	// How long to wait for miner blocks to propagate to other nodes
	propWait := time.Second * 3

	m := miner.CreateMinerAddr()

	askO, err := miner.Run(
		"miner", "add-ask",
		"--from", miner.Config().Mining.RewardAddress.String(),
		m.String(), "1200", "1",
	)
	if err != nil {
		panic(err)
	}
	miner.MineAndPropagate(propWait, d)
	_, err = miner.Run("message", "wait", "--return", strings.TrimSpace(askO.ReadStdout()))
	if err != nil {
		panic(err)
	}

	_, err = d.Run(
		"client", "add-bid",
		"--from", d.Config().Mining.RewardAddress.String(),
		"500", "1",
	)
	if err != nil {
		panic(err)
	}
	d.MineAndPropagate(propWait, miner)

	buf := strings.NewReader(dealData)
	o, err := d.RunWithStdin(buf, "client", "import")
	if err != nil {
		panic(err)
	}
	ddCid := strings.TrimSpace(o.ReadStdout())

	negidO, err := d.Run("client", "propose-deal", "--ask=0", "--bid=0", ddCid)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Millisecond * 20)

	miner.MineAndPropagate(propWait, d)

	negid := strings.Split(strings.Split(negidO.ReadStdout(), "\n")[1], " ")[1]
	// ensure we have made the deal
	_, err = d.Run("client", "query-deal", negid)
	if err != nil {
		panic(err)
	}
	// return the cid for the dealData (ddCid)
	return ddCid
}

// TryAPICheck will check if the daemon is ready
// TODO don't panic be happy
func TryAPICheck(d *Daemon) error {
	url := fmt.Sprintf("http://127.0.0.1%s/api/id", d.CmdAddr)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return fmt.Errorf("liveness check failed: %s", err)
	}

	_, ok := out["ID"]
	if !ok {
		return fmt.Errorf("liveness check failed: ID field not present in output")
	}

	return nil
}
