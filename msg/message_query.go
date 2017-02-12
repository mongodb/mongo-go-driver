package msg

// Query is a message sent to the server.
type Query struct {
	ReqID                int32
	Flags                QueryFlags
	FullCollectionName   string
	NumberToSkip         int32
	NumberToReturn       int32
	Query                interface{}
	ReturnFieldsSelector interface{}
}

// RequestID gets the request id of the message.
func (m *Query) RequestID() int32 { return m.ReqID }

// QueryFlags are the flags in a Query.
type QueryFlags int32

// QueryFlags constants.
const (
	_ QueryFlags = 1 << iota
	TailableCursor
	SlaveOK
	OplogReplay
	NoCursorTimeout
	AwaitData
	Exhaust
	Partial
)
