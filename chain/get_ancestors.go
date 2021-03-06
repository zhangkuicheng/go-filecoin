package chain

import (
	"context"

	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/sampling"
	"github.com/filecoin-project/go-filecoin/types"
)

// GetRecentAncestorsOfHeaviestChain returns the ancestors of a `TipSet` with
// height `descendantBlockHeight` in the heaviest chain.
func GetRecentAncestorsOfHeaviestChain(ctx context.Context, chainReader ReadStore, descendantBlockHeight *types.BlockHeight) ([]types.TipSet, error) {
	head := chainReader.GetHead()
	headTipSetAndState, err := chainReader.GetTipSetAndState(head)
	if err != nil {
		return nil, err
	}
	return GetRecentAncestors(ctx, headTipSetAndState.TipSet, chainReader, descendantBlockHeight, consensus.AncestorRoundsNeeded, sampling.LookbackParameter)
}

// GetRecentAncestors returns the ancestors of base as a slice of TipSets.
//
// In order to validate post messages, randomness from the chain is required.
// This function collects that randomess: all tipsets with height greater than
// childBH - ancestorRounds, and the lookback tipsets that precede them.
//
// The return slice is a concatenation of two slices: append(provingPeriodAncestors, extraRandomnessAncestors...)
//   provingPeriodAncestors: all ancestor tipsets with height greater than childBH - ancestorRoundsNeeded
//   extraRandomnessAncestors: the lookback number of tipsets directly preceding tipsets in provingPeriodAncestors
//
// The last tipset of provingPeriodAncestors is the earliest possible tipset to
// begin a proving period that is still "live", i.e it is valid to accept PoSts
// over this proving period when processing a tipset at childBH.  The last
// tipset of extraRandomnessAncestors is the tipset used to sample randomness
// for any PoSts with a proving period beginning at the last tipset of
// provingPeriodAncestors.  By including ancestors as far back as the last tipset
// of extraRandomnessAncestors, the consensus state transition function can sample
// the randomness used by all live PoSts to correctly process all valid
// 'submitPoSt' messages.
//
// Because null blocks increase chain height but do not have associated tipsets
// the length of provingPeriodAncestors may vary (more null blocks -> shorter length).  The
// length of slice extraRandomnessAncestors is a constant (at least once the
// chain is longer than lookback tipsets).
func GetRecentAncestors(ctx context.Context, base types.TipSet, chainReader ReadStore, childBH, ancestorRoundsNeeded *types.BlockHeight, lookback uint) ([]types.TipSet, error) {
	if lookback == 0 {
		return nil, errors.New("lookback must be greater than 0")
	}
	earliestAncestorHeight := childBH.Sub(ancestorRoundsNeeded)
	if earliestAncestorHeight.LessThan(types.NewBlockHeight(0)) {
		earliestAncestorHeight = types.NewBlockHeight(uint64(0))
	}

	// Step 1 -- gather all tipsets with a height greater than the earliest
	// possible proving period start still in scope for the given head.
	iterator := IterAncestors(ctx, chainReader, base)
	provingPeriodAncestors, err := CollectTipSetsOfHeightAtLeast(ctx, iterator, earliestAncestorHeight)
	if err != nil {
		return nil, err
	}
	firstExtraRandomnessAncestorsCids, err := provingPeriodAncestors[len(provingPeriodAncestors)-1].Parents()
	if err != nil {
		return nil, err
	}
	// no parents means hit genesis so return the whole chain
	if firstExtraRandomnessAncestorsCids.Len() == 0 {
		return provingPeriodAncestors, nil
	}

	// Step 2 -- gather the lookback tipsets directly preceding provingPeriodAncestors.
	tsas, err := chainReader.GetTipSetAndState(firstExtraRandomnessAncestorsCids)
	if err != nil {
		return nil, err
	}
	iterator = IterAncestors(ctx, chainReader, tsas.TipSet)
	extraRandomnessAncestors, err := CollectAtMostNTipSets(ctx, iterator, lookback)
	if err != nil {
		return nil, err
	}
	return append(provingPeriodAncestors, extraRandomnessAncestors...), nil
}

// CollectTipSetsOfHeightAtLeast collects all tipsets with a height greater
// than or equal to minHeight from the input tipset.
func CollectTipSetsOfHeightAtLeast(ctx context.Context, iterator *TipsetIterator, minHeight *types.BlockHeight) ([]types.TipSet, error) {
	var ret []types.TipSet
	var err error
	var h uint64
	for ; !iterator.Complete(); err = iterator.Next() {
		if err != nil {
			return nil, err
		}
		h, err = iterator.Value().Height()
		if err != nil {
			return nil, err
		}
		if types.NewBlockHeight(h).LessThan(minHeight) {
			return ret, nil
		}
		ret = append(ret, iterator.Value())
	}
	return ret, nil
}

// CollectAtMostNTipSets collect N tipsets from the input channel.  If there
// are fewer than n tipsets in the channel it returns all of them.
func CollectAtMostNTipSets(ctx context.Context, iterator *TipsetIterator, n uint) ([]types.TipSet, error) {
	var ret []types.TipSet
	var err error
	for i := uint(0); i < n && !iterator.Complete(); i++ {
		ret = append(ret, iterator.Value())
		if err = iterator.Next(); err != nil {
			return nil, err
		}
	}
	return ret, nil
}
