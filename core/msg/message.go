package msg

import "sync/atomic"

var globalRequestID int32

// CurrentRequestID gets the current request id.
func CurrentRequestID() int32 {
	return atomic.AddInt32(&globalRequestID, 0)
}

// NextRequestID gets the next request id.
func NextRequestID() int32 {
	return atomic.AddInt32(&globalRequestID, 1)
}

type opcode uint32

const (
	replyOpcode opcode = 1
	queryOpcode opcode = 2004
)

// Message represents a MongoDB message.
type Message interface {
	msg()
}

// Request is a message sent to the server.
type Request interface {
	Message
	RequestID() int32
}

// Response is a message received from the server.
type Response interface {
	Message
	ResponseTo() int32
}

func (m *Query) msg() {}
func (m *Reply) msg() {}
