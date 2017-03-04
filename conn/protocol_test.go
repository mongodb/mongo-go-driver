package conn_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"

	. "github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/conntest"
	"github.com/10gen/mongo-go-driver/internal/msgtest"
	"github.com/10gen/mongo-go-driver/msg"
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
	conn.ResponseQ = append(conn.ResponseQ, msgtest.CreateCommandReply(bson.D{{"ok", 1}}))

	var result okResp
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

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

	conn := &conntest.MockConnection{}
	conn.WriteErr = fmt.Errorf("error writing")

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed sending commands", 1)
}

func TestExecuteCommand_Error_reading_from_connection(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed receiving command response", 1)
}

func TestExecuteCommand_Error_from_multiple_requests(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{
		SkipResponseToFixup: true,
	}
	reply := msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommands(context.Background(), conn, []msg.Request{&msg.Query{}, &msg.Query{}}, []interface{}{&result, &result})

	validateExecuteCommandError(t, err, "multiple errors encountered", 1)
}

func TestExecuteCommand_NumberReturned_is_0(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 0
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "command returned no documents", 1)
}

func TestExecuteCommand_NumberReturned_is_greater_than_1(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.NumberReturned = 10
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

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

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "unknown command failure", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_malformed_document(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)

	// not enough bytes for document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)

	// corrupted document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command failure document", 1)
}

func TestExecuteCommand_QueryFailure_flag_with_document(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"error", true}})
	reply.ResponseFlags = msg.QueryFailure
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "command failure: [{error true}]", 1)
}

func TestExecuteCommand_No_command_response(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := &msg.Reply{
		NumberReturned: 1,
	}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "no command response document", 1)
}

func TestExecuteCommand_Error_decoding_response(t *testing.T) {
	t.Parallel()

	// can't read length
	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command response document", 1)

	// not enough bytes for document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{0, 1, 5, 62, 23}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command response document", 1)

	// corrupted document
	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 1}})
	reply.DocumentsBytes = []byte{1, 0, 0, 0, 4, 6}
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "failed to read command response document", 1)
}

func TestExecuteCommand_OK_field_is_false(t *testing.T) {
	t.Parallel()

	conn := &conntest.MockConnection{}
	reply := msgtest.CreateCommandReply(bson.D{{"funny", 0}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	var result bson.D
	err := ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "command failed", 1)

	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 0}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "command failed", 1)

	conn = &conntest.MockConnection{}
	reply = msgtest.CreateCommandReply(bson.D{{"ok", 0}, {"errmsg", "weird command was invalid"}})
	conn.ResponseQ = append(conn.ResponseQ, reply)

	err = ExecuteCommand(context.Background(), conn, &msg.Query{}, &result)

	validateExecuteCommandError(t, err, "weird command was invalid", 1)
}
