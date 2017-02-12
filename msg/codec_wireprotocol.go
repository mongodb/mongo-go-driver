package msg

import (
	"fmt"
	"io"

	"gopkg.in/mgo.v2/bson"
)

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
		return nil, fmt.Errorf("unable to decode message length: %v", err)
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
		return nil, fmt.Errorf("unable to decode message: %v", err)
	}

	return c.decode(b)
}

func (c *wireProtocolCodec) Encode(writer io.Writer, msgs ...Message) error {

	b := make([]byte, 0, 256)

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
				return fmt.Errorf("unable to marshal query: %v", err)
			}
			if typedM.ReturnFieldsSelector != nil {
				b, err = addMarshalled(b, typedM.ReturnFieldsSelector)
				if err != nil {
					return fmt.Errorf("unable to marshal return fields selector: %v", err)
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
		return fmt.Errorf("unable to encode messages: %v", err)
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

func addCString(b []byte, s string) []byte {
	b = append(b, []byte(s)...)
	return append(b, 0)
}

func addInt32(b []byte, i int32) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
}

func addInt64(b []byte, i int64) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24), byte(i>>32), byte(i>>40), byte(i>>48), byte(i>>56))
}

func addMarshalled(b []byte, data interface{}) ([]byte, error) {
	if data == nil {
		return append(b, 5, 0, 0, 0, 0), nil
	}

	dataBytes, err := bson.Marshal(data)
	if err != nil {
		return nil, err
	}

	return append(b, dataBytes...), nil
}

func setInt32(b []byte, pos int32, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

func addHeader(b []byte, length, requestID, responseTo, opCode int32) []byte {
	b = addInt32(b, length)
	b = addInt32(b, requestID)
	b = addInt32(b, responseTo)
	return addInt32(b, opCode)
}

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) |
		(int32(b[pos+1]) << 8) |
		(int32(b[pos+2]) << 16) |
		(int32(b[pos+3]) << 24)
}

func readInt64(b []byte, pos int32) int64 {
	return (int64(b[pos+0])) |
		(int64(b[pos+1]) << 8) |
		(int64(b[pos+2]) << 16) |
		(int64(b[pos+3]) << 24) |
		(int64(b[pos+4]) << 32) |
		(int64(b[pos+5]) << 40) |
		(int64(b[pos+6]) << 48) |
		(int64(b[pos+7]) << 56)
}

func bsonDocumentPartitioner(bytes []byte) (int, error) {
	if len(bytes) < 4 {
		return 0, fmt.Errorf("int32 requires 4 bytes but only %d available", len(bytes))
	}

	n := readInt32(bytes, 0)
	return int(n), nil
}
