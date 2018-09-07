package testhelpers

import (
	"path/filepath"
)

// The file used to build these addresses can be found in:
// $GOPATH/src/github.com/filecoin-project/go-filecoin/testhelpers/testfiles/setup.json
//
// If said file is modified these addresses will need to change as well
//

const TestAddress1 = "fcqdp2qp4yun0envsv0wl8pyjl97vuvsfre3sct6" // nolint: deadcode, golint
const TestKey1 = "a.key"                                         // nolint: deadcode, golint

const TestAddress2 = "fcqlru4l28u62g7lz99dx2254k52sf4nx6f5tqweq" // nolint: deadcode, golint
const TestKey2 = "b.key"                                         // nolint: deadcode, golint

const TestAddress3 = "fcqgg0c9npe5anrv4ytuyg0yggkwqx5q2h8laqec" // nolint: deadcode, golint
const TestKey3 = "c.key"                                         // nolint: deadcode, golint

const TestAddress4 = "fcqj3sknp5famqmnl9jd4nnq9c48cnrvmccx3cx7g" // nolint: deadcode, varcheck, megacheck, golint
const TestKey4 = "d.key"                                         // nolint: deadcode, golint

const TestAddress5 = "fcqjm6qw5fecm3tlv57sjkycuqt3d5h7m6v2c2v6a" // nolint: deadcode, varcheck, megacheck, golint
const TestKey5 = "e.key"                                         // nolint: deadcode, golint

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
