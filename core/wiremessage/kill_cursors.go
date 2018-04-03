package wiremessage

// KillCursors represents the OP_KILL_CURSORS message of the MongoDB wire protocol.
type KillCursors struct {
	MsgHeader         Header
	NumberOfCursorIDs int32
	CursorIDs         []int64
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (kc KillCursors) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (kc KillCursors) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (kc KillCursors) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (kc KillCursors) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (kc KillCursors) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (kc *KillCursors) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}
