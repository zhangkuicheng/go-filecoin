package pluginlocalfilecoin

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	peer "gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	"github.com/filecoin-project/go-filecoin/repo"
)

func (l *Localfilecoin) isAlive() (bool, error) {
	pid, err := l.getPID()
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}

	return false, nil
}

func (l *Localfilecoin) getPID() (int, error) {
	b, err := ioutil.ReadFile(filepath.Join(l.dir, "daemon.pid"))
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(b))
}

func (l *Localfilecoin) env() ([]string, error) {
	envs := os.Environ()
	filecoinpath := "FIL_PATH=" + l.dir

	for i, e := range envs {
		if strings.HasPrefix(e, "FIL_PATH=") {
			envs[i] = filecoinpath
			return envs, nil
		}
	}

	return append(envs, filecoinpath), nil
}

func (l *Localfilecoin) signalAndWait(p *os.Process, waitch <-chan struct{}, signal os.Signal, t time.Duration) error {
	err := p.Signal(signal)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", l.dir, err)
	}

	select {
	case <-waitch:
		return nil
	case <-time.After(t):
		return errTimeout
	}
}

func (l *Localfilecoin) readerFor(file string) (io.ReadCloser, error) {
	return os.OpenFile(filepath.Join(l.dir, file), os.O_RDONLY, 0)
}

func (l *Localfilecoin) cachePeerID() error {
	// save the daemons peerID to a file
	rep, err := repo.OpenFSRepo(l.dir)
	if err != nil {
		return err
	}
	defer rep.Close()

	sk, err := rep.Keystore().Get("self")
	if err != nil {
		return err
	}

	peerID, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return err
	}

	db, err := l.db()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Put(key_peerID, []byte(peerID.Pretty()))
}
