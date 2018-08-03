package types

// TipIndex tracks tipsets by height and parent set, mainly for use in expected consensus.
type TipIndex map[uint64]tipSetsByParents

func (ti TipIndex) addBlock(b *Block) error {
	tsbp, ok := ti[uint64(b.Height)]
	if !ok {
		tsbp = tipSetsByParents{}
		ti[uint64(b.Height)] = tsbp
	}
	return tsbp.addBlock(b)
}

type tipSetsByParents map[string]TipSet

func (tsbp tipSetsByParents) addBlock(b *Block) error {
	key := KeyForParentSet(b.Parents)
	ts := tsbp[key]
	if ts == nil {
		ts = TipSet{}
	}
	err := ts.AddBlock(b)
	if err != nil {
		return err
	}
	tsbp[key] = ts
	return nil
}

func KeyForParentSet(parents SortedCidSet) string {
	var k string
	for it := parents.Iter(); !it.Complete(); it.Next() {
		k += it.Value().String()
	}
	return k
}
