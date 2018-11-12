package testhelpers

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/filecoin-project/go-filecoin/util/project"
)

// GetFreePort gets a free port from the kernel
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
	defer l.Close() // nolint: errcheck
	return l.Addr().(*net.TCPAddr).Port, nil
}

// MustGetFilecoinBinary returns the path where the filecoin binary will be if it has been built and panics otherwise.
func MustGetFilecoinBinary() string {
	bin := project.Root("go-filecoin")
	_, err := os.Stat(bin)
	if err != nil {
		panic(err)
	}

	if os.IsNotExist(err) {
		panic("You are missing the filecoin binary...try building'")
	}

	return bin
}

// WaitForIt waits until the given callback returns true.
func WaitForIt(count int, delay time.Duration, cb func() (bool, error)) error {
	var done bool
	var err error
	for i := 0; i < count; i++ {
		done, err = cb()
		if err != nil {
			return err
		}
		if done {
			break
		}
		time.Sleep(delay)
	}

	if !done {
		return fmt.Errorf("timeout waiting for it")
	}

	return nil
}
