package wiremessage

import "fmt"

// Header represents the header of a MongoDB wire protocol message.
type Header struct {
	MessageLength int32
	RequestID     int32
	ResponseTo    int32
	OpCode        OpCode
}

func (h Header) String() string {
	return fmt.Sprintf(
		`Header{MessageLength: %d, RequestID: %d, ResponseTo: %d, OpCode: %v}`,
		h.MessageLength, h.RequestID, h.ResponseTo, h.OpCode,
	)
}
