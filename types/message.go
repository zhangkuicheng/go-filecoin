package types

import (
	"errors"

	cbor "gx/ipfs/QmRiRJhn427YVuufBEHofLreKWNw7P7BWNq86Sb9kzqdbd/go-ipld-cbor"
	errPkg "gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

func init() {
	cbor.RegisterCborType(Message{})
}

var (
	// ErrInvalidMessageLength is returned when the message length does not match the expected length.
	ErrInvalidMessageLength = errors.New("invalid message length")
)

// Message is an exchange of information between two actors modeled
// as a function call.
// Messages are the equivalent of transactions in Ethereum.
type Message struct {
	To   Address `json:"to"`
	From Address `json:"from"`
	// When receiving a message from a user account the nonce in
	// the message must match the expected nonce in the from actor.
	// This prevents replay attacks.
	Nonce Uint64 `json:"nonce"`

	Value *AttoFIL `json:"value"`

	Method string `json:"method"`
	Params []byte `json:"params"`

	Signature []byte `json:"Signature"`
}

// Unmarshal a message from the given bytes.
func (msg *Message) Unmarshal(b []byte) error {
	return cbor.DecodeInto(b, msg)
}

// Marshal the message into bytes.
func (msg *Message) Marshal() ([]byte, error) {
	return cbor.DumpObject(msg)
}

// Cid returns the canonical CID for the message.
// TODO: can we avoid returning an error?
func (msg *Message) Cid() (*cid.Cid, error) {
	obj, err := cbor.WrapObject(msg, DefaultHashFunction, -1)
	if err != nil {
		return nil, errPkg.Wrap(err, "failed to marshal to cbor")
	}

	return obj.Cid(), nil
}

func (msg *Message) Sign(from Address, s Signer) error {
	if msg.Signed() {
		// TODO(frrist): Replace this with a real error before merge
		panic("here")
	}

	bmsg, err := msg.Marshal()
	if err != nil {
		return err
	}

	msg.Signature, err = s.SignBytes(from, bmsg)
	if err != nil {
		return err
	}

	return nil
}

func (msg *Message) RecoverAddress(r Recoverer) (Address, error) {
	if !msg.Signed() {
		panic("here")
	}

	// Do this because the recovered pk will be different if we include the sig
	recSig := msg.Signature
	recMsg := &Message{
		To:     msg.To,
		From:   msg.From,
		Nonce:  msg.Nonce,
		Value:  msg.Value,
		Method: msg.Method,
		Params: msg.Params,
	}

	bRecMsg, err := recMsg.Marshal()
	if err != nil {
		return Address{}, err
	}

	maybePk, err := r.Ecrecover(bRecMsg, recSig)
	if err != nil {
		return Address{}, err
	}

	maybeAddrHash, err := AddressHash(maybePk)
	if err != nil {
		return Address{}, err
	}

	return NewMainnetAddress(maybeAddrHash), nil

}

func (msg *Message) Signed() bool {
	return len(msg.Signature) > 0
}

// NewMessage creates a new message.
func NewMessage(from, to Address, nonce uint64, value *AttoFIL, method string, params []byte) *Message {
	return &Message{
		From:   from,
		To:     to,
		Nonce:  Uint64(nonce),
		Value:  value,
		Method: method,
		Params: params,
	}
}
