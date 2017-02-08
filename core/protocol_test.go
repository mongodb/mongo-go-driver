package core_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"

	. "github.com/craiggwilson/mongo-go-driver/core"
	"github.com/craiggwilson/mongo-go-driver/core/msg"
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

	conn := &mockConnection{}
	conn.pushResponse(createCommandReply(bson.D{{"ok", 1}}))

	var result okResp
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	if err != nil {
		t.Fatalf("expected nil err but got \"%s\"", err)
	}
	if conn.writeCount != 1 {
		t.Fatalf("expected 1 write, but had %d", conn.writeCount)
	}

	if result.OK != 1 {
		t.Fatalf("expected response ok to be 1 but was %d", result.OK)
	}
}

func TestExecuteCommand_Error_writing_to_connection(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	conn.writeErr = fmt.Errorf("error writing")

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed sending commands", 1)
}

func TestExecuteCommand_Error_reading_from_connection(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed receiving command response", 1)
}

func TestExecuteCommand_ResponseTo_does_not_equal_request_id(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"ok", 1}})
	reply.RespTo = 1000
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "received out of order response", 1)
}

func TestExecuteCommand_NumberReturned_is_0(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 0
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command returned no documents", 1)
}

func TestExecuteCommand_NumberReturned_is_greater_than_1(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 10
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command returned multiple documents", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_no_document(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
		ResponseFlags:  msg.QueryFailure,
	}
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: unknown command failure", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_malformed_document(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	reply.ResponseFlags = msg.QueryFailure
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)

	// not enough bytes for document
	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	reply.ResponseFlags = msg.QueryFailure
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)

	// corrupted document
	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	reply.ResponseFlags = msg.QueryFailure
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)
}

func TestExecuteCommand_No_command_response(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
	}
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: no command response document", 1)
}

func TestExecuteCommand_Error_decoding_response(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)

	// not enough bytes for document
	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)

	// corrupted document
	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)
}

func TestExecuteCommand_OK_field_is_false(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	reply := createCommandReply(bson.D{{"funny", 0}})
	conn.pushResponse(reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed", 1)

	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 0}})
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed", 1)

	conn = &mockConnection{}
	reply = createCommandReply(bson.D{{"ok", 0}, {"errmsg", "weird command was invalid"}})
	conn.pushResponse(reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed: weird command was invalid", 1)
}

type mockConnection struct {
	responseQ  []msg.Response
	writeCount uint8
	writeErr   error
}

func (c *mockConnection) Desc() *ConnectionDesc {
	return &ConnectionDesc{}
}

func (c *mockConnection) Read() (msg.Response, error) {
	if len(c.responseQ) == 0 {
		return nil, fmt.Errorf("no response queued")
	}
	resp := c.responseQ[0]
	c.responseQ = c.responseQ[1:]
	return resp, nil
}

func (c *mockConnection) Write(...msg.Request) error {
	c.writeCount++
	return c.writeErr
}

func (c *mockConnection) pushResponse(resp msg.Response) {
	c.responseQ = append(c.responseQ, resp)
}

func createCommandReply(in interface{}) *msg.Reply {
	doc, _ := bson.Marshal(in)
	reply := &msg.Reply{
		NumberReturned: 1,
		DocumentsBytes: doc,
	}

	// encode it, then decode it to handle the internal workings of msg.Reply
	codec := msg.NewWireProtocolCodec()
	var b bytes.Buffer
	err := codec.Encode(&b, reply)
	if err != nil {
		panic(err)
	}
	resp, err := codec.Decode(&b)
	if err != nil {
		panic(err)
	}

	return resp.(*msg.Reply)
}
