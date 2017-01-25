package msg

// Reply is a message received from the server.
type Reply struct {
	ReqID          int32
	RespTo         int32
	ResponseFlags  ReplyFlags
	CursorID       int64
	StartingFrom   int32
	NumberReturned int32
	DocumentsBytes []byte

	unmarshaller documentUnmarshaller
	partitioner  documentPartitioner
}

// ResponseTo gets the request id the message was in response to.
func (m *Reply) ResponseTo() int32 { return m.RespTo }

type documentUnmarshaller func([]byte, interface{}) error
type documentPartitioner func([]byte) (int, error)

// ReplyFlags are the flags in a Reply.
type ReplyFlags int32

// ReplayMessageFlags constants.
const (
	CursorNotFound ReplyFlags = 1 << iota
	QueryFailure
	_
	AwaitCapable
)

// Iter returns a ReplyIter to iterate over each document
// returned by the server.
func (m *Reply) Iter() *ReplyIter {
	return &ReplyIter{
		unmarshaller:   m.unmarshaller,
		partitioner:    m.partitioner,
		documentsBytes: m.DocumentsBytes,
	}
}

// ReplyIter iterates over the documents returned
// in a Reply.
type ReplyIter struct {
	unmarshaller   documentUnmarshaller
	partitioner    documentPartitioner
	documentsBytes []byte
	pos            int

	err error
}

// One reads a single document from the iterator.
func (i *ReplyIter) One(result interface{}) (bool, error) {
	if !i.Next(result) {
		return false, i.err
	}

	return true, nil
}

// Next marshals the next document into the provided result and returns
// a value indicating whether or not it was successful.
func (i *ReplyIter) Next(result interface{}) bool {
	if i.pos >= len(i.documentsBytes) {
		return false
	}
	n, err := i.partitioner(i.documentsBytes[i.pos:])
	if err != nil {
		i.err = err
		return false
	}

	err = i.unmarshaller(i.documentsBytes[i.pos:i.pos+n], result)
	if err != nil {
		i.err = err
		return false
	}

	i.pos += n
	return true
}

// Err indicates if there was an error unmarshalling the last document
// attempted.
func (i *ReplyIter) Err() error {
	return i.err
}
