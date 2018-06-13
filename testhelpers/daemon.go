package testhelpers

import (
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
)

// Daemon is a daemon
type Daemon struct {
	CmdAddr    string
	SwarmAddr  string
	RepoDir    string
	Init       bool
	waitMining bool

	// The filecoin daemon process
	process *exec.Cmd

	lk     sync.Mutex
	Stdin  io.Writer
	Stdout io.Reader
	Stderr io.Reader

	insecureApi bool
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
		CmdAddr:     fmt.Sprintf(":%d", cmdPort),
		SwarmAddr:   fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", swarmPort),
		RepoDir:     dir,
		Init:        true, // we want to init unless told otherwise
		insecureApi: false,
	}

	// configure TestDaemon options
	for _, option := range options {
		option(d)
	}

	repodirFlag := fmt.Sprintf("--repodir=%s", d.RepoDir)
	if d.Init {
		out, err := runInit(repodirFlag)
		if err != nil {
			panic(err)
			d.Info(string(out))
			return nil, err
		}
	}

	args := []string{
		"daemon",
		fmt.Sprintf("--repodir=%s", d.RepoDir),
		fmt.Sprintf("--cmdapiaddr=%s", d.CmdAddr),
		fmt.Sprintf("--swarmlisten=%s", d.SwarmAddr),
	}

	if d.insecureApi {
		args = append(args, fmt.Sprintf("--insecureapi"))
	}

	// define filecoin daemon process
	d.process = exec.Command(filecoinBin, args...)

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

	stderrFile, err := os.Create(filepath.Join(d.RepoDir, "stderr.daemon"))
	if err != nil {
		return nil, err
	}

	go func() {
		io.Copy(stderrFile, d.Stderr)
	}()

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
func (d *Daemon) Logf(format string, a ...interface{}) {
	fmt.Printf("[%s]\t%s\n", d.CmdAddr, fmt.Sprintf(format, a...))
}

// Log is a daemon logger
func (d *Daemon) Info(msg ...string) {
	fmt.Printf("[%s]\t %s\n", d.CmdAddr, msg)
}

// Log is a daemon logger
func (d *Daemon) Error(err error) {
	fmt.Errorf("[%s]\t %s\n", d.CmdAddr, err)
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

	d.Logf("run: %q\n", strings.Join(finalArgs, " "))
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

// Shutdown shuts things down
func (d *Daemon) Shutdown() error {
	if err := d.process.Process.Signal(syscall.SIGTERM); err != nil {
		d.Logf("Daemon Stderr:\n%s", d.ReadStderr())
		d.Logf("Failed to kill daemon %s", err)
		return err
	}

	if d.RepoDir == "" {
		panic("daemon had no repodir set")
	}

	return nil
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

// Config is a helper to read out the config of the deamon
func (d *Daemon) Config() (*config.Config, error) {
	cfg, err := config.ReadFile(filepath.Join(d.RepoDir, "config.toml"))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// MineAndPropagate mines a block and ensure the block has propogated to all `peers`
// by comparing the current head block of `d` with the head block of each peer in `peers`
func (d *Daemon) MineAndPropagate(wait time.Duration, peers ...*Daemon) error {
	_, err := d.Run("mining", "once")
	if err != nil {
		return err
	}
	// short circuit
	if peers == nil {
		return nil
	}
	// ensure all peers have same chain head as `d`
	return d.MustHaveChainHeadBy(wait, peers)
}

// TryAPICheck will check if the daemon is ready
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

func (d *Daemon) SetWaitMining(t bool) {
	d.lk.Lock()
	defer d.lk.Unlock()
	d.waitMining = t
}

func (d *Daemon) WaitMining() bool {
	d.lk.Lock()
	defer d.lk.Unlock()
	return d.waitMining
}
