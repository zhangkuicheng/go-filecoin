package commands

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
)

func runSuccessFirstLine(td *th.TestDaemon, args ...string) string {
	return runSuccessLines(td, args...)[0]
}

func runSuccessLines(td *th.TestDaemon, args ...string) []string {
	output := td.RunSuccess(args...)
	result := output.ReadStdoutTrimNewlines()
	return strings.Split(result, "\n")
}

// Credit: https://github.com/phayes/freeport
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
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
