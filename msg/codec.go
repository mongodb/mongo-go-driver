package msg

import (
	"fmt"
	"io"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/internal"
)

// Encoder encodes messages.
type Encoder interface {
	// Encode encodes a number of messages to the writer.
	Encode(io.Writer, ...Message) error
}

// Decoder decodes messages.
type Decoder interface {
	// Decode decodes one message from the reader.
	Decode(io.Reader) (Message, error)
}

// Codec encodes and decodes messages.
type Codec interface {
	Encoder
	Decoder
}

// NewWireProtocolCodec creates a MessageReadWriter for the binary message format.
func NewWireProtocolCodec() Codec {
	return &wireProtocolCodec{
		lengthBytes: make([]byte, 4),
	}
}

type wireProtocolCodec struct {
	lengthBytes []byte
}

func (c *wireProtocolCodec) Decode(reader io.Reader) (Message, error) {
	_, err := io.ReadFull(reader, c.lengthBytes)
	if err != nil {
		return nil, internal.WrapError(err, "unable to decode message length")
	}

	length := readInt32(c.lengthBytes, 0)

	// TODO: use a buffer pool
	b := make([]byte, length)

	b[0] = c.lengthBytes[0]
	b[1] = c.lengthBytes[1]
	b[2] = c.lengthBytes[2]
	b[3] = c.lengthBytes[3]

	_, err = io.ReadFull(reader, b[4:])
	if err != nil {
		return nil, internal.WrapError(err, "unable to decode message")
	}

	return c.decode(b)
}

func (c *wireProtocolCodec) Encode(writer io.Writer, msgs ...Message) error {

	b := make([]byte, 0, defaultEncodeBufferSize)

	var err error
	for _, msg := range msgs {
		start := len(b)
		switch typedM := msg.(type) {
		case *Query:
			b = addHeader(b, 0, typedM.ReqID, 0, int32(queryOpcode))
			b = addInt32(b, int32(typedM.Flags))
			b = addCString(b, typedM.FullCollectionName)
			b = addInt32(b, typedM.NumberToSkip)
			b = addInt32(b, typedM.NumberToReturn)
			b, err = addMarshalled(b, typedM.Query)
			if err != nil {
				return internal.WrapError(err, "unable to marshal query")
			}
			if typedM.ReturnFieldsSelector != nil {
				b, err = addMarshalled(b, typedM.ReturnFieldsSelector)
				if err != nil {
					return internal.WrapError(err, "unable to marshal return fields selector")
				}
			}
		case *Reply:
			b = addHeader(b, 0, typedM.ReqID, typedM.RespTo, int32(replyOpcode))
			b = addInt32(b, int32(typedM.ResponseFlags))
			b = addInt64(b, typedM.CursorID)
			b = addInt32(b, typedM.StartingFrom)
			b = addInt32(b, typedM.NumberReturned)
			b = append(b, typedM.DocumentsBytes...)
		}

		setInt32(b, int32(start), int32(len(b)-start))
	}

	_, err = writer.Write(b)
	if err != nil {
		return internal.WrapError(err, "unable to encode messages")
	}
	return nil
}

func (c *wireProtocolCodec) decode(b []byte) (Message, error) {
	// size := readInt32(b, 0) // no reason to do this
	requestID := readInt32(b, 4)
	responseTo := readInt32(b, 8)
	op := readInt32(b, 12)

	switch opcode(op) {
	case replyOpcode:
		replyMessage := &Reply{
			ReqID:        requestID,
			RespTo:       responseTo,
			partitioner:  bsonDocumentPartitioner,
			unmarshaller: bson.Unmarshal,
		}
		replyMessage.ResponseFlags = ReplyFlags(readInt32(b, 16))
		replyMessage.CursorID = readInt64(b, 20)
		replyMessage.StartingFrom = readInt32(b, 28)
		replyMessage.NumberReturned = readInt32(b, 32)
		replyMessage.DocumentsBytes = b[36:] // TODO: need to copy out the bytes?
		return replyMessage, nil
	}

	return nil, fmt.Errorf("opcode %d not implemented", op)
}
