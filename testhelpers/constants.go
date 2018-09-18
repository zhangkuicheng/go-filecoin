package testhelpers

import (
	"path/filepath"
)

// The file used to build these addresses can be found in:
// $GOPATH/src/github.com/filecoin-project/go-filecoin/testhelpers/testfiles/setup.json
//
// If said file is modified these addresses will need to change as well
//

const TestAddress1 = "fcq3d04er6tkfaa02j8jw444a43p8d2s0c98z232x" // nolint: deadcode, golint
const TestKey1 = "a.key"                                         // nolint: deadcode, golint

const TestAddress2 = "fcqn5h3lcwskp6gguzd8r4v9e4yhp9xcyud40jz8k" // nolint: deadcode, golint
const TestKey2 = "b.key"                                         // nolint: deadcode, golint

const TestAddress3 = "fcq0e8r5huklp54zwh6s6t6r3d4cnf0drq93l6jjz" // nolint: deadcode, golint
const TestKey3 = "c.key"                                         // nolint: deadcode, golint

const TestAddress4 = "fcqkmzkc3442ydkqmpecy62cr5srzmcyh0wdltgv" // nolint: deadcode, varcheck, megacheck, golint
const TestKey4 = "d.key"                                        // nolint: deadcode, golint

const TestAddress5 = "fcqzx983czlkuyzr69uaw33s89qv4j62lc6ep4h5c" // nolint: deadcode, varcheck, megacheck, golint
const TestKey5 = "e.key"                                         // nolint: deadcode, golint

const TestMinerAddress = "fcqn3lue7efxjejcd3dx6lejvmr426n02rd6kj6zn" // nolint: deadcode, varcheck, megacheck, golint

// GenesisFilePath returns the path of the WalletFile
// Head after running with setup.json is: zdpuAkgCshuhMj8nB5nHW3HFboWpvz8JxKvHxMBfDCaKeV2np
func GenesisFilePath() string {
	gopath, err := getGoPath()
	if err != nil {
		panic(err)
	}

	return filepath.Join(gopath, "/src/github.com/filecoin-project/go-filecoin/testhelpers/testfiles/genesis.car")
}

// KeyFilePaths returns the paths to the wallets of the testaddresses
func KeyFilePaths() []string {
	gopath, err := getGoPath()
	if err != nil {
		panic(err)
	}
	folder := "/src/github.com/filecoin-project/go-filecoin/testhelpers/testfiles/"

	return []string{
		filepath.Join(gopath, folder, "a.key"),
		filepath.Join(gopath, folder, "b.key"),
		filepath.Join(gopath, folder, "c.key"),
		filepath.Join(gopath, folder, "d.key"),
		filepath.Join(gopath, folder, "e.key"),
	}
}

func keyFilePath(kf string) string {
	gopath, err := getGoPath()
	if err != nil {
		panic(err)
	}
	folder := "/src/github.com/filecoin-project/go-filecoin/testhelpers/testfiles/"

	return filepath.Join(gopath, folder, kf)
}
