package commands

import (
	"strings"
	"testing"
)

// TODO add more tests cases when filecoin-project/go-filecoin#479 is resolved
func TestSignErrorCases(t *testing.T) {
	// Used to generate address's
	ephemeral := NewDaemon(t).Start()
	defer func() { t.Log(ephemeral.ReadStderr()) }()
	defer ephemeral.ShutdownSuccess()

	// Make a valid address that doesn't belong to below daemon
	ephemAddr := strings.Trim(ephemeral.RunSuccess("wallet", "addrs", "new").ReadStdout(), "\n")

	d := NewDaemon(t).Start()
	defer func() { t.Log(d.ReadStderr()) }()
	defer d.ShutdownSuccess()

	addr := strings.Trim(d.RunSuccess("wallet", "addrs", "new").ReadStdout(), "\n")

	// Sign data with unknown encoding.
	d.RunFail("encoding not supported", "wallet", "sign", addr, "123")

	// Sign with known but unsupported encoding (base32hexUpper).
	d.RunFail("Encoding must be base32hex", "wallet", "sign", addr, "V123")

	// Sign with invalid address.
	d.RunFail("invalid checksum", "wallet", "sign", "fqcc", "123")

	// Sign with an address that's not owned by different daemon
	d.RunFail("unknown address", "wallet", "sign", ephemAddr, "v123")

}
