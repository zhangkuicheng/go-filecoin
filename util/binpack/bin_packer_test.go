package binpack

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-filecoin/types"
	"github.com/stretchr/testify/assert"
)

func TestNaivePacker(t *testing.T) {
	assert := assert.New(t)

	binner := &testBinner{binSize: types.NewBytesAmount(20)}
	packer, _, _ := NewNaivePacker(binner)

	newItem := func(size uint64) testItem {
		return testItem{size: types.NewBytesAmount(size)}
	}

	_, err := packer.AddItem(context.Background(), newItem(10))
	assert.NoError(err)
	assert.Equal(types.NewBytesAmount(10), binner.currentBinUsed.size)
	assert.Equal(0, binner.closeCount)

	_, err = packer.AddItem(context.Background(), newItem(8))
	assert.NoError(err)
	assert.Equal(types.NewBytesAmount(18), binner.currentBinUsed.size)
	assert.Equal(0, binner.closeCount)

	_, err = packer.AddItem(context.Background(), newItem(2))
	assert.NoError(err)
	assert.Equal(types.ZeroBytes, binner.currentBinUsed.size)
	assert.Equal(1, binner.closeCount)

	_, err = packer.AddItem(context.Background(), newItem(5))
	assert.NoError(err)
	assert.Equal(types.NewBytesAmount(5), binner.currentBinUsed.size)
	assert.Equal(1, binner.closeCount)

	_, err = packer.AddItem(context.Background(), newItem(25))
	assert.EqualError(err, "item too large for bin")
	assert.Equal(types.NewBytesAmount(5), binner.currentBinUsed.size)
	assert.Equal(1, binner.closeCount)
}

// Binner implementation for tests.

type testItem struct {
	size *types.BytesAmount
}

type testBin struct {
	size *types.BytesAmount
}

func (tb *testBin) GetID() uint64 {
	return 0
}

type testBinner struct {
	binSize        *types.BytesAmount
	currentBinUsed *testBin
	closeCount     int
}

var _ Binner = &testBinner{}

func (tb *testBinner) GetCurrentBin() Bin {
	return tb.currentBinUsed
}

func (tb *testBinner) AddItem(ctx context.Context, item Item, bin Bin) error {
	if tb.currentBinUsed == nil {
		tb.currentBinUsed = &testBin{size: types.NewBytesAmount(0)}
	}
	tb.currentBinUsed.size = tb.currentBinUsed.size.Add(item.(testItem).size)
	return nil
}

func (tb *testBinner) BinSize() *types.BytesAmount {
	return tb.binSize
}

func (tb *testBinner) CloseBin(Bin) {
	tb.currentBinUsed.size = types.ZeroBytes
	tb.closeCount++
}

func (tb *testBinner) ItemSize(item Item) *types.BytesAmount {
	return item.(testItem).size
}

func (tb *testBinner) NewBin() (Bin, error) {
	return &testBin{size: tb.binSize}, nil
}

func (tb *testBinner) SpaceAvailable(bin Bin) *types.BytesAmount {
	if tb.currentBinUsed == nil {
		return tb.binSize
	}
	return tb.binSize.Sub(tb.currentBinUsed.size)
}
