// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/builder"
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
			ReqID:  requestID,
			RespTo: responseTo,
		}
		replyMessage.ResponseFlags = ReplyFlags(readInt32(b, 16))
		replyMessage.CursorID = readInt64(b, 20)
		replyMessage.StartingFrom = readInt32(b, 28)
		replyMessage.NumberReturned = readInt32(b, 32)
		// We don't copy the bytes here since we'll copy the document is
		// retrieved.
		replyMessage.DocumentsBytes = b[36:]
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

	var dataBytes []byte
	var err error

	switch t := data.(type) {
	// NOTE: bson.Document is covered by bson.Marshaler
	case bson.Marshaler:
		dataBytes, err = t.MarshalBSON()
		if err != nil {
			return nil, err
		}
	case bson.Reader:
		_, err = t.Validate()
		if err != nil {
			return nil, err
		}
		dataBytes = t
	case builder.DocumentBuilder:
		dataBytes = make([]byte, t.RequiredBytes())
		_, err = t.WriteDocument(dataBytes)
		if err != nil {
			return nil, err
		}
	case []byte:
		_, err = bson.Reader(t).Validate()
		if err != nil {
			return nil, err
		}
		dataBytes = t
	default:
		var buf bytes.Buffer
		err = bson.NewEncoder(&buf).Encode(data)
		if err != nil {
			return nil, err
		}
		dataBytes = buf.Bytes()
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
