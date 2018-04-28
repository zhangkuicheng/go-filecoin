package types

import (
	cbor "gx/ipfs/QmRVSCwQtW1rjHCay9NqKXDwbtKTgDcN4iY7PrpSqfKM5D/go-ipld-cbor"
)

func init() {
	cbor.RegisterCborType(MessageReceipt{})
}

// ReturnValueLength is the length limit of return values
const ReturnValueLength = 256

// ReturnValue is the fixed size type for return values in receipts.
type ReturnValue = [ReturnValueLength]byte

// MessageReceipt represents the result of sending a message.
type MessageReceipt struct {
	// `0` is success, anything else is an error code in unix style.
	ExitCode uint8 `json:"exitCode"`

	// TODO: switch to ptr + size once allocations are implemented
	Return ReturnValue `json:"return"`

	ReturnSize uint32 `json:"returnSize"`
}

func (r *MessageReceipt) ReturnBytes() []byte {
	return r.Return[0:r.ReturnSize]
}

// NewMessageReceipt creates a new MessageReceipt.
func NewMessageReceipt(exitCode uint8, ret ReturnValue, retSize uint32) *MessageReceipt {
	return &MessageReceipt{
		ExitCode:   exitCode,
		Return:     ret,
		ReturnSize: retSize,
	}
}
