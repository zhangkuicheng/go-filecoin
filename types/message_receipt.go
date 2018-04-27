package types

import (
	cbor "gx/ipfs/QmRVSCwQtW1rjHCay9NqKXDwbtKTgDcN4iY7PrpSqfKM5D/go-ipld-cbor"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

func init() {
	cbor.RegisterCborType(MessageReceipt{})
}

// ReturnValueLength is the length limit of return values
const ReturnValueLength = 2048

// ReturnValue is the fixed size type for return values in receipts.
type ReturnValue = [ReturnValueLength]byte

// MessageReceipt represents the result of sending a message.
type MessageReceipt struct {
	// `0` is success, anything else is an error code in unix style.
	ExitCode uint8 `json:"exitCode"`

	// TODO: switch to ptr + size once allocations are implemented
	Return ReturnValue `json:"return"`
}

// NewMessageReceipt creates a new MessageReceipt.
func NewMessageReceipt(msg *cid.Cid, exitCode uint8, ret ReturnValue) *MessageReceipt {
	return &MessageReceipt{
		ExitCode: exitCode,
		Return:   ret,
	}
}
