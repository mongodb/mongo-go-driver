package wiremessage

import (
	"errors"
	"fmt"
)

// Header represents the header of a MongoDB wire protocol message.
type Header struct {
	MessageLength int32
	RequestID     int32
	ResponseTo    int32
	OpCode        OpCode
}

func ReadHeader(b []byte, pos int32) (Header, error) {
	if len(b) < 16 {
		return Header{}, errors.New("invalid header: []byte too small")
	}
	return Header{
		MessageLength: readInt32(b, 0),
		RequestID:     readInt32(b, 4),
		ResponseTo:    readInt32(b, 8),
		OpCode:        OpCode(readInt32(b, 12)),
	}, nil
}

func (h Header) String() string {
	return fmt.Sprintf(
		`Header{MessageLength: %d, RequestID: %d, ResponseTo: %d, OpCode: %v}`,
		h.MessageLength, h.RequestID, h.ResponseTo, h.OpCode,
	)
}

func (h Header) AppendHeader(b []byte) []byte {
	b = appendInt32(b, h.MessageLength)
	b = appendInt32(b, h.RequestID)
	b = appendInt32(b, h.ResponseTo)
	b = appendInt32(b, int32(h.OpCode))
	return b
}
