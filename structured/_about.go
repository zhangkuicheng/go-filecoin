// package structured defines the structured storage API that is used by actors
// in Filecoin.
//
// The API is currently implemented by Noms, but that should be viewed as an
// opaque implementation detail. We want to maintain the ability to implement
// this API with other backends should that become desirable.
//
// Changes to this API should be carefully considered as to whether they bind
// Filecoin permanently to those features and semantics.
//
// Example Usage:
//
// ds := NewDatastore(myChunkStoreImpl)
// m := NewMap(ds).Set(String("foo"), Uint8(42))
// ds.Commit(m, ds.Head())
package structured
