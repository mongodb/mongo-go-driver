package wiremessage

import "github.com/mongodb/mongo-go-driver/bson"

// Reply represents the OP_REPLY message of the MongoDB wire protocol.
type Reply struct {
	MsgHeader      Header
	ResponseFlags  ReplyFlag
	CursorID       int64
	StartingFrom   int32
	NumberReturned int32
	Documents      []bson.Reader
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (r Reply) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (r Reply) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (r Reply) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (r Reply) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (r Reply) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (r *Reply) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}

// ReplyFlag represents the flags of an OP_REPLY message.
type ReplyFlag int32

// These constants represent the individual flags of an OP_REPLY message.
const (
	CursorNotFound ReplyFlag = 1 << iota
	QueryFailure
	ShardConfigStale
	AwaitCapable
)
