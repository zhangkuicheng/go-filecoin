package types

type Signer interface {
	SignBytes(addr Address, data []byte) (Signature, error)
}
