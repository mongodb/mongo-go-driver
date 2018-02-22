package wiremessage

import "github.com/skriptble/wilson/bson"

// Command represents the OP_COMMAND message of the MongoDB wire protocol.
type Command struct {
	MsgHeader   Header
	Database    string
	CommandName string
	Metadata    string
	CommandArgs string
	InputDocs   []bson.Reader
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (c Command) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (c Command) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (c Command) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (c Command) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (c Command) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (c *Command) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}
