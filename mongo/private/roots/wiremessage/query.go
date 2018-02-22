package wiremessage

import "github.com/skriptble/wilson/bson"

// Query represents the OP_QUERY message of the MongoDB wire protocol.
type Query struct {
	MsgHeader            Header
	Flags                QueryFlag
	FullCollectionName   string
	NumberToSkip         int32
	NumberToReturn       int32
	Query                bson.Reader
	ReturnFieldsSelector bson.Reader
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (q Query) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (q Query) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (q Query) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (q Query) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (q Query) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (q *Query) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}

// QueryFlag represents the flags on an OP_QUERY message.
type QueryFlag int32

// These constants represent the individual flags on an OP_QUERY message.
const (
	_ QueryFlag = 1 << iota
	TailableCursor
	SlaveOK
	OplogReplay
	NoCursorTimeout
	AwaitData
	Exhaust
	Partial
)
