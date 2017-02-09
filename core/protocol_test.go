package core_test

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"

	. "github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"github.com/10gen/mongo-go-driver/internal/internaltest"
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

	conn := &internaltest.MockConnection{}
	conn.ResponseQ = append(conn.ResponseQ, internaltest.CreateCommandReply(bson.D{{"ok", 1}}))

	var result okResp
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	if err != nil {
		t.Fatalf("expected nil err but got \"%s\"", err)
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

	conn := &internaltest.MockConnection{}
	conn.WriteErr = fmt.Errorf("error writing")

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed sending commands", 1)
}

func TestExecuteCommand_Error_reading_from_connection(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed receiving command response", 1)
}

func TestExecuteCommand_ResponseTo_does_not_equal_request_id(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{
		SkipResponseToFixup: true,
	}
	reply := internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.RespTo = 1000
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "received out of order response", 1)
}

func TestExecuteCommand_NumberReturned_is_0(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}
	reply := internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command returned no documents", 1)
}

func TestExecuteCommand_NumberReturned_is_greater_than_1(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}
	reply := internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 10
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command returned multiple documents", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_no_document(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
		ResponseFlags:  msg.QueryFailure,
	}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: unknown command failure", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_malformed_document(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &internaltest.MockConnection{}
	reply := internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)

	// not enough bytes for document
	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)

	// corrupted document
	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command failure document", 1)
}

func TestExecuteCommand_No_command_response(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
	}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: no command response document", 1)
}

func TestExecuteCommand_Error_decoding_response(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &internaltest.MockConnection{}
	reply := internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)

	// not enough bytes for document
	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)

	// corrupted document
	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: failed to read command response document:", 1)
}

func TestExecuteCommand_OK_field_is_false(t *testing.T) {
	t.Parallel()

	conn := &internaltest.MockConnection{}
	reply := internaltest.CreateCommandReply(bson.D{{"funny", 0}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed", 1)

	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 0}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed", 1)

	conn = &internaltest.MockConnection{}
	reply = internaltest.CreateCommandReply(bson.D{{"ok", 0}, {"errmsg", "weird command was invalid"}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed reading command response for 0: command failed: weird command was invalid", 1)
}
