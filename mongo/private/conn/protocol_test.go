// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"

	"github.com/mongodb/mongo-go-driver/mongo/internal/conntest"
	"github.com/mongodb/mongo-go-driver/mongo/internal/msgtest"
	. "github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

func validateExecuteCommandError(t *testing.T, err error, errPrefix string, writeCount int) {
	if err == nil {
		t.Fatalf("expected an err but did not get one")
	}
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
	if writeCount != 1 {
		t.Fatalf("expected 1 write, but had %d", writeCount)
	}
}

func TestExecuteCommand_Valid(t *testing.T) {
	t.Parallel()

	type okResp struct {
		OK int32 `bson:"ok"`
	}

	conn := &conntest.MockConnection{}
	conn.ResponseQ = append(conn.ResponseQ, msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1))))

	var result okResp
	rdr, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	if err != nil {
		t.Fatalf("expected nil err but got \"%s\"", err)
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&result)
	if err != nil {
		t.Fatalf("unexpected error while trying to decode result: %s", err)
	}

	if len(conn.Sent) != 1 {
		t.Fatalf("expected 1 write, but had %d", len(conn.Sent))
	}

	if result.OK != 1 {
		t.Fatalf("expected response ok to be 1 but was %d", result.OK)
	}
}

func TestExecuteCommand_Error_writing_to_connection(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	conn.WriteErr = fmt.Errorf("error writing")

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed sending commands", 1)
}

func TestExecuteCommand_Error_reading_from_connection(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed receiving command response", 1)
}

func TestExecuteCommand_Error_from_multiple_requests(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{
		SkipResponseToFixup: true,
	}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommands(context.Background(), conn, []msg.Request{&msg.Query{}, &msg.Query{}})

	validateExecuteCommandError(t, err, "multiple errors encountered", 1)
}

func TestExecuteCommand_NumberReturned_is_0(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "command returned no documents", 1)
}

func TestExecuteCommand_NumberReturned_is_greater_than_1(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.NumberReturned = 10
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "command returned multiple documents", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_no_document(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
		ResponseFlags:  msg.QueryFailure,
	}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "unknown command failure", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_malformed_document(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{0, 1, 5}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)

	// not enough bytes for document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)

	// corrupted document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_document(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Boolean("error", true)))
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, `command failure: bson.Reader{bson.Element{[boolean]"error": true}}`, 1)
}

func TestExecuteCommand_No_command_response(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
	}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "no command response document", 1)
}

func TestExecuteCommand_Error_decoding_response(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{0, 1, 5}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command response document", 1)

	// not enough bytes for document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command response document", 1)

	// corrupted document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "failed to read command response document", 1)
}

func TestExecuteCommand_OK_field_is_false(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("funny", 0)))
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err := ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "command failed", 1)

	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 0)))
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "command failed", 1)

	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 0), bson.EC.String("errmsg", "weird command was invalid")))
	conn.ResponseQ = append(conn.ResponseQ, reply)

	_, err = ExecuteCommand(context.Background(), conn, &msg.Query{})

	validateExecuteCommandError(t, err, "weird command was invalid", 1)
}
