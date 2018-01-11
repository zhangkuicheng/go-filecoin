package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func getPath() string {
	base := os.Getenv("FIL_PATH")
	if base == "" {
		base = os.Getenv("HOME")
	}
	return filepath.Join(base, "filecoin-daemon-lock")
}

func writeDaemonLock() error {
	p := getPath()
	return ioutil.WriteFile(p, []byte("its locked, don't worry"), 0440)
}

func removeDaemonLock() {
	_ = os.Remove(getPath())
}

func DaemonIsRunning() (bool, error) {
	p := getPath()
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
