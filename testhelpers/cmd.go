package testhelpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	errors "gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/filecoin-project/go-filecoin/types"
)

// GetID gets the peerid of the daemon
func (d *Daemon) GetID() (string, error) {
	out, err := d.Run("id", "--enc=json")
	if err != nil {
		return "", err
	}

	var parsed map[string]interface{}
	fmt.Println(out.ReadStdout())
	err = json.Unmarshal([]byte(out.ReadStdoutTrimNewlines()), &parsed)
	if err != nil {
		return "", err
	}

	s, ok := parsed["ID"].(string)
	if !ok {
		return "", errors.New("id format incorrect")
	}
	return s, nil
}

// GetAddress gets the libp2p address of the daemon
func (d *Daemon) GetAddress() (string, error) {
	out, err := d.Run("id", "--enc=json")
	if err != nil {
		return "", err
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(out.ReadStdout()), &parsed)
	if err != nil {
		return "", err
	}

	adders, ok := parsed["Addresses"].([]interface{})
	if !ok {
		return "", errors.New("address format incorrect")
	}

	s, ok := adders[0].(string)
	if !ok {
		return "", errors.New("address format incorrect")
	}
	return s, nil
}

// ConnectSuccess connects 2 daemons and pacnis if it fails
func (d *Daemon) Connect(remote *Daemon) (*Output, error) {
	// Connect the nodes
	addr, err := remote.GetAddress()
	if err != nil {
		return nil, err
	}

	out, err := d.Run("swarm", "connect", addr)
	if err != nil {
		return out, err
	}
	peers1, err := d.Run("swarm", "peers")
	if err != nil {
		return out, err
	}
	peers2, err := remote.Run("swarm", "peers")
	if err != nil {
		return out, err
	}

	rid, err := remote.GetID()
	if err != nil {
		return out, err
	}
	lid, err := d.GetID()
	if err != nil {
		return out, err
	}

	if !strings.Contains(peers1.ReadStdout(), rid) {
		return out, errors.New("failed to connect (2->1)")
	}
	if !strings.Contains(peers2.ReadStdout(), lid) {
		return out, errors.New("failed to connect (1->2)")
	}

	return out, nil
}

// CreateMinerAddr issues a new message to the network, mines the message
// and returns the address of the new miner
// equivalent to:
//     `go-filecoin miner create --from $TEST_ACCOUNT 100000 20`
// TODO don't panic be happy
func (d *Daemon) CreateMinerAddr() (types.Address, error) {
	// need money
	_, err := d.Run("mining", "once")
	if err != nil {
		panic(err)
		return types.Address{}, err
	}

	nodeCfg, err := d.Config()
	if err != nil {
		panic(err)
		return types.Address{}, err
	}
	addr := nodeCfg.Mining.RewardAddress

	var wg sync.WaitGroup
	var minerAddr types.Address

	errchan := make(chan error)
	wg.Add(1)
	go func(errchan chan error) {
		miner, err := d.Run("miner", "create", "--from", addr.String(), "1000000", "1000")
		if err != nil {
			errchan <- err
			d.Error(err)
			return
		}
		addr, err := types.NewAddressFromString(strings.Trim(miner.ReadStdout(), "\n"))
		if err != nil {
			fmt.Println(addr.String())
			errchan <- err
			d.Error(err)
			return
		}
		if addr.Empty() {
			errchan <- err
			d.Error(err)
			return
		}
		minerAddr = addr
		wg.Done()
	}(errchan)

	_, err = d.Run("mining", "once")
	if err != nil {
		return types.Address{}, err
	}
	if len(errchan) > 0 {
		return types.Address{}, errors.Errorf("%d errors happened during miner create", len(errchan))
	}

	wg.Wait()
	return minerAddr, nil
}

// CreateWalletAddr adds a new address to the daemons wallet and
// returns it.
// equivalent to:
//     `go-filecoin wallet addrs new`
func (d *Daemon) CreateWalletAddr() (string, error) {
	outNew, err := d.Run("wallet", "addrs", "new")
	if err != nil {
		return "", err
	}
	addr := strings.Trim(outNew.ReadStdout(), "\n")
	if addr == "" {
		return "", errors.New("got empty address")

	}
	return addr, nil
}

// GetMainWalletAddress does thats
func (d *Daemon) GetMainWalletAddress() (string, error) {
	out, err := d.Run("address", "ls")
	if err != nil {
		return "", err
	}

	addr := strings.Trim(out.ReadStdout(), "\n ")
	return addr, nil
}

// MustHaveChainHeadBy ensures all `peers` have the same chain head as `d`, by
// duration `wait`
func (d *Daemon) MustHaveChainHeadBy(wait time.Duration, peers []*Daemon) error {
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
		return nil
	case <-time.After(wait):
		// TODO don't panic be happy
		return errors.New("timeout exceeded waiting for chain head to sync")
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

func (d *Daemon) ProposeDeal(askID, bidID uint64, dataRef string) (*Output, error) {
	out, err := d.Run("client", "propose-deal",
		fmt.Sprintf("--ask=%d", askID),
		fmt.Sprintf("--bid=%d", bidID),
		dataRef,
	)
	return out, err
}

// MakeDeal will make a deal with the miner `miner`, using data `dealData`.
// MakeDeal will return the cid of `dealData`
// TODO don't panic be happy
func (d *Daemon) MakeDeal(dealData string, miner *Daemon) (string, error) {

	// The daemons need 2 monies each.
	d.MakeMoney(2)
	miner.MakeMoney(2)

	// How long to wait for miner blocks to propagate to other nodes
	propWait := time.Second * 3

	m, err := miner.CreateMinerAddr()
	if err != nil {
		return "", err
	}

	minerCfg, err := miner.Config()
	if err != nil {
		return "", err
	}
	minerAddr := minerCfg.Mining.RewardAddress.String()

	askO, err := miner.Run(
		"miner", "add-ask",
		"--from", minerAddr,
		m.String(), "1200", "1",
	)
	if err != nil {
		return "", err
	}

	miner.MineAndPropagate(propWait, d)
	_, err = miner.Run("message", "wait", "--return", strings.TrimSpace(askO.ReadStdout()))
	if err != nil {
		return "", err
	}

	clientCfg, err := d.Config()
	if err != nil {
		return "", err
	}
	clientAddr := clientCfg.Mining.RewardAddress.String()
	_, err = d.Run(
		"client", "add-bid",
		"--from", clientAddr,
		"500", "1",
	)
	if err != nil {
		return "", err
	}
	d.MineAndPropagate(propWait, miner)

	buf := strings.NewReader(dealData)
	o, err := d.RunWithStdin(buf, "client", "import")
	if err != nil {
		return "", err
	}
	ddCid := strings.TrimSpace(o.ReadStdout())

	negidO, err := d.Run("client", "propose-deal", "--ask=0", "--bid=0", ddCid)
	if err != nil {
		return "", err
	}
	time.Sleep(time.Millisecond * 20)

	miner.MineAndPropagate(propWait, d)

	negid := strings.Split(strings.Split(negidO.ReadStdout(), "\n")[1], " ")[1]
	// ensure we have made the deal
	_, err = d.Run("client", "query-deal", negid)
	if err != nil {
		return "", err
	}
	// return the cid for the dealData (ddCid)
	return ddCid, nil
}

func (d *Daemon) EventLogStream() io.Reader {
	r, w := io.Pipe()

	go func() {
		defer w.Close()

		url := fmt.Sprintf("http://127.0.0.1%s/api/log/tail", d.CmdAddr)
		res, err := http.Get(url)
		if err != nil {
			return
		}
		io.Copy(w, res.Body)
		defer res.Body.Close()
	}()

	return r
}

func (td *Daemon) MiningOnce() error {
	_, err := td.Run("mining", "once")
	return err
}

/*****************************************************************************/
/***************************** Suspect bad methods ***************************/

// SendFilecoin does that
func (d *Daemon) SendFilecoin(ctx context.Context, from, to string, amt int) error {
	out, err := d.Run("message", "send",
		fmt.Sprintf("--value=%d", amt),
		fmt.Sprintf("--from=%s", from),
		to)
	if err != nil {
		return err
	}

	cid, err := cid.Parse(strings.Trim(out.ReadStdout(), "\n"))
	if err != nil {
		return err
	}

	_, err = d.MineForMessage(ctx, cid.String())
	return err
}

func (d *Daemon) MineForMessage(ctx context.Context, msg string) (*Output, error) {

	d.Info("message wait: mining for message ", msg)
	var outErr error
	var out *Output

	wait := make(chan struct{})
	go func() {
		out, outErr = d.WaitForMessage(ctx, msg)
		d.Info("message wait: mined message ", msg)
		close(wait)
	}()

	var mineErr error
	if d.WaitMining() { // if disabled, skip (for realistic network sim)
		mineErr = d.MiningOnce()
	}

	<-wait

	if mineErr != nil {
		return out, mineErr
	}
	return out, outErr
}

func (d *Daemon) WaitForMessage(ctx context.Context, msg string) (out *Output, err error) {
	d.Info("message wait: waiting for message ", msg)

	// do it async to allow "canceling out" via context.
	done := make(chan struct{})

	go func() {
		// sets the return vars
		out, err = d.Run("message", "wait",
			"--return",
			"--message=false",
			"--receipt=false",
			msg,
		)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return out, err
	}
}

func (td *Daemon) WalletBalance(addr string) (int, error) {
	out, err := td.Run("wallet", "balance", addr)
	if err != nil {
		return 0, err
	}

	balance, err := strconv.Atoi(strings.Trim(out.ReadStdout(), "\n"))
	if err != nil {
		return balance, err
	}
	return balance, err
}

func (td *Daemon) MinerAddAsk(ctx context.Context, from string, size, price int) error {
	out, err := td.Run("miner", "add-ask", from,
		strconv.Itoa(size), strconv.Itoa(price))
	if err != nil {
		return err
	}

	cid, err := cid.Parse(strings.Trim(out.ReadStdout(), "\n"))
	if err != nil {
		return err
	}

	_, err = td.MineForMessage(ctx, cid.String())
	return err
}

func (td *Daemon) ClientAddBid(ctx context.Context, from string, size, price int) error {
	out, err := td.Run("client", "add-bid", fmt.Sprintf("--from=%s", from),
		strconv.Itoa(size), strconv.Itoa(price))
	if err != nil {
		return err
	}

	cid, err := cid.Parse(strings.Trim(out.ReadStdout(), "\n"))
	if err != nil {
		return err
	}

	_, err = td.MineForMessage(ctx, cid.String())
	return err
}

func (td *Daemon) OrderbookGetAsks(ctx context.Context) (*Output, error) {
	return td.Run("orderbook", "asks", "--enc=json")
}

func (td *Daemon) OrderbookGetBids(ctx context.Context) (*Output, error) {
	return td.Run("orderbook", "bids")
}

func (td *Daemon) OrderbookGetDeals(ctx context.Context) (*Output, error) {
	return td.Run("orderbook", "deals")
}
