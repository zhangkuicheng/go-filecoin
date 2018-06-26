package types

type Recoverer interface {
	Ecrecover(data []byte, sig Signature) ([]byte, error)
}
