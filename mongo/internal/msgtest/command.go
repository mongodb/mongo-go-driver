// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msgtest

import (
	"bytes"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// CreateCommandReply creates a msg.Reply from the BSON response from the server.
func CreateCommandReply(cmd interface{}) *msg.Reply {
	var buf bytes.Buffer
	err := bson.NewEncoder(&buf).Encode(cmd)
	if err != nil {
		panic(err)
	}
	reply := &msg.Reply{
		NumberReturned: 1,
		DocumentsBytes: buf.Bytes(),
	}

	// encode it, then decode it to handle the internal workings of msg.Reply
	codec := msg.NewWireProtocolCodec()
	var b bytes.Buffer
	err = codec.Encode(&b, reply)
	if err != nil {
		panic(err)
	}
	resp, err := codec.Decode(&b)
	if err != nil {
		panic(err)
	}

	return resp.(*msg.Reply)
}
