package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/go-filecoin/types"
	"github.com/stretchr/testify/require"
)

func TestDagDaemon(t *testing.T) {
	t.Parallel()
	t.Run("dag get <cid> returning the genesis block", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		d := NewDaemon(t).Start()
		defer d.ShutdownSuccess()

		// get the CID of the genesis block from the "chain ls" command output

		op1 := d.RunSuccess("chain", "ls", "--enc", "json")
		result1 := op1.readStdoutTrimNewlines()

		genesisBlockJSONStr := bytes.Split([]byte(result1), []byte{'\n'})[0]

		var expected types.Block
		json.Unmarshal(genesisBlockJSONStr, &expected)

		// get an IPLD node from the DAG by its CID

		op2 := d.RunSuccess("dag", "get", expected.Cid().String(), "--enc", "json")
		result2 := op2.readStdoutTrimNewlines()

		var actual types.Block
		err := json.Unmarshal([]byte(result2), &actual)
		require.NoError(err)

		// CIDs should be equal

		types.AssertHaveSameCid(assert, &expected, &actual)
	})
}
