package topologyx

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

var emptyDoc bson.Raw

type legacyConnection struct {
	conn       *Connection
	writeBuf   []byte
	readBuf    []byte
	cmdMonitor *event.CommandMonitor
	commandMap map[int64]*commandMetadata // map for monitoring commands sent to server
}

func (c *legacyConnection) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	// Truncate the write buffer
	c.writeBuf = c.writeBuf[:0]

	// We don't handle compression here as the *Connection will handle that for us.

	var err error
	c.writeBuf, err = wm.AppendWireMessage(c.writeBuf)
	if err != nil {
		return ConnectionError{
			ConnectionID: c.conn.pc.id,
			Wrapped:      err,
			message:      "unable to encode wire message",
		}
	}

	err = c.conn.WriteWireMessage(ctx, c.writeBuf)
	if err != nil {
		// The error we got back was probably a ConnectionError already, so we don't really need to
		// wrap it here.
		return ConnectionError{
			ConnectionID: c.conn.pc.id,
			Wrapped:      err,
			message:      "unable to write wire message to network",
		}
	}

	return c.commandStartedEvent(ctx, wm)
}

func (c *legacyConnection) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	// Truncate the write buffer
	c.readBuf = c.readBuf[:0]

	var err error
	c.readBuf, err = c.conn.ReadWireMessage(ctx, c.readBuf)
	if err != nil {
		// The error we got back was probably a ConnectionError already, so we don't really need to
		// wrap it here.
		return nil, ConnectionError{
			ConnectionID: c.conn.pc.id,
			Wrapped:      err,
			message:      "unable to read wire message from network",
		}
	}
	hdr, err := wiremessage.ReadHeader(c.readBuf, 0)
	if err != nil {
		return nil, ConnectionError{
			ConnectionID: c.conn.pc.id,
			Wrapped:      err,
			message:      "unable to decode header",
		}
	}

	messageToDecode := c.readBuf
	opcodeToCheck := hdr.OpCode

	var wm wiremessage.WireMessage
	switch opcodeToCheck {
	case wiremessage.OpReply:
		var r wiremessage.Reply
		err := r.UnmarshalWireMessage(messageToDecode)
		if err != nil {
			return nil, ConnectionError{
				ConnectionID: c.conn.pc.id,
				Wrapped:      err,
				message:      "unable to decode OP_REPLY",
			}
		}
		wm = r
	case wiremessage.OpMsg:
		var reply wiremessage.Msg
		err := reply.UnmarshalWireMessage(messageToDecode)
		if err != nil {
			return nil, ConnectionError{
				ConnectionID: c.conn.pc.id,
				Wrapped:      err,
				message:      "unable to decode OP_MSG",
			}
		}
		wm = reply
	default:
		return nil, ConnectionError{
			ConnectionID: c.conn.pc.id,
			message:      fmt.Sprintf("opcode %s not implemented", hdr.OpCode),
		}
	}

	// TODO: do we care if monitoring fails?
	return wm, c.commandFinishedEvent(ctx, wm)
}

func (lc *legacyConnection) Close() error {
	panic("not implemented")
}

func (lc *legacyConnection) Expired() bool {
	panic("not implemented")
}

func (lc *legacyConnection) Alive() bool {
	panic("not implemented")
}

func (lc *legacyConnection) ID() string {
	panic("not implemented")
}

func (c *legacyConnection) commandStartedEvent(ctx context.Context, wm wiremessage.WireMessage) error {
	if c.cmdMonitor == nil || c.cmdMonitor.Started == nil {
		return nil
	}

	startedEvent := &event.CommandStartedEvent{
		ConnectionID: c.conn.pc.id,
	}

	var cmd bsonx.Doc
	var err error
	var legacy bool
	var fullCollName string

	var acknowledged bool
	switch converted := wm.(type) {
	case wiremessage.Query:
		cmd, err = converted.CommandDocument()
		if err != nil {
			return err
		}

		acknowledged = converted.AcknowledgedWrite()
		startedEvent.DatabaseName = converted.DatabaseName()
		startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
		legacy = converted.Legacy()
		fullCollName = converted.FullCollectionName
	case wiremessage.Msg:
		cmd, err = converted.GetMainDocument()
		if err != nil {
			return err
		}

		acknowledged = converted.AcknowledgedWrite()
		arr, identifier, err := converted.GetSequenceArray()
		if err != nil {
			return err
		}
		if arr != nil {
			cmd = cmd.Copy() // make copy to avoid changing original command
			cmd = append(cmd, bsonx.Elem{identifier, bsonx.Array(arr)})
		}

		dbVal, err := cmd.LookupErr("$db")
		if err != nil {
			return err
		}

		startedEvent.DatabaseName = dbVal.StringValue()
		startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
	case wiremessage.GetMore:
		cmd = converted.CommandDocument()
		startedEvent.DatabaseName = converted.DatabaseName()
		startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
		acknowledged = true
		legacy = true
		fullCollName = converted.FullCollectionName
	case wiremessage.KillCursors:
		cmd = converted.CommandDocument()
		startedEvent.DatabaseName = converted.DatabaseName
		startedEvent.RequestID = int64(converted.MsgHeader.RequestID)
		legacy = true
	}

	rawcmd, _ := cmd.MarshalBSON()
	startedEvent.Command = rawcmd
	startedEvent.CommandName = cmd[0].Key
	if !canMonitor(startedEvent.CommandName) {
		startedEvent.Command = emptyDoc
	}

	c.cmdMonitor.Started(ctx, startedEvent)

	if !acknowledged {
		if c.cmdMonitor.Succeeded == nil {
			return nil
		}

		// unack writes must provide a CommandSucceededEvent with an { ok: 1 } reply
		finishedEvent := event.CommandFinishedEvent{
			DurationNanos: 0,
			CommandName:   startedEvent.CommandName,
			RequestID:     startedEvent.RequestID,
			ConnectionID:  c.conn.pc.id,
		}

		c.cmdMonitor.Succeeded(ctx, &event.CommandSucceededEvent{
			CommandFinishedEvent: finishedEvent,
			Reply:                bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "ok", 1)),
		})

		return nil
	}

	c.commandMap[startedEvent.RequestID] = createMetadata(startedEvent.CommandName, legacy, fullCollName)
	return nil
}

func (c *legacyConnection) commandFinishedEvent(ctx context.Context, wm wiremessage.WireMessage) error {
	if c.cmdMonitor == nil {
		return nil
	}

	var reply bsonx.Doc
	var requestID int64
	var err error

	switch converted := wm.(type) {
	case wiremessage.Reply:
		requestID = int64(converted.MsgHeader.ResponseTo)
	case wiremessage.Msg:
		requestID = int64(converted.MsgHeader.ResponseTo)
	}
	cmdMetadata := c.commandMap[requestID]
	delete(c.commandMap, requestID)

	switch converted := wm.(type) {
	case wiremessage.Reply:
		if cmdMetadata.Legacy {
			reply, err = converted.GetMainLegacyDocument(cmdMetadata.FullCollectionName)
		} else {
			reply, err = converted.GetMainDocument()
		}
	case wiremessage.Msg:
		reply, err = converted.GetMainDocument()
	}
	if err != nil {
		return err
	}

	success, errmsg := processReply(reply)

	if (success && c.cmdMonitor.Succeeded == nil) || (!success && c.cmdMonitor.Failed == nil) {
		return nil
	}

	finishedEvent := event.CommandFinishedEvent{
		DurationNanos: cmdMetadata.TimeDifference(),
		CommandName:   cmdMetadata.Name,
		RequestID:     requestID,
		ConnectionID:  c.conn.pc.id,
	}

	if success {
		if !canMonitor(finishedEvent.CommandName) {
			successEvent := &event.CommandSucceededEvent{
				Reply:                emptyDoc,
				CommandFinishedEvent: finishedEvent,
			}
			c.cmdMonitor.Succeeded(ctx, successEvent)
			return nil
		}

		// if response has type 1 document sequence, the sequence must be included as a BSON array in the event's reply.
		if opmsg, ok := wm.(wiremessage.Msg); ok {
			arr, identifier, err := opmsg.GetSequenceArray()
			if err != nil {
				return err
			}
			if arr != nil {
				reply = reply.Copy() // make copy to avoid changing original command
				reply = append(reply, bsonx.Elem{identifier, bsonx.Array(arr)})
			}
		}

		replyraw, _ := reply.MarshalBSON()
		successEvent := &event.CommandSucceededEvent{
			Reply:                replyraw,
			CommandFinishedEvent: finishedEvent,
		}

		c.cmdMonitor.Succeeded(ctx, successEvent)
		return nil
	}

	failureEvent := &event.CommandFailedEvent{
		Failure:              errmsg,
		CommandFinishedEvent: finishedEvent,
	}

	c.cmdMonitor.Failed(ctx, failureEvent)
	return nil
}

func canMonitor(cmd string) bool {
	if cmd == "authenticate" || cmd == "saslStart" || cmd == "saslContinue" || cmd == "getnonce" || cmd == "createUser" ||
		cmd == "updateUser" || cmd == "copydbgetnonce" || cmd == "copydbsaslstart" || cmd == "copydb" {
		return false
	}

	return true
}

func processReply(reply bsonx.Doc) (bool, string) {
	var success bool
	var errmsg string
	var errCode int32

	for _, elem := range reply {
		switch elem.Key {
		case "ok":
			switch elem.Value.Type() {
			case bsontype.Int32:
				if elem.Value.Int32() == 1 {
					success = true
				}
			case bsontype.Int64:
				if elem.Value.Int64() == 1 {
					success = true
				}
			case bsontype.Double:
				if elem.Value.Double() == 1 {
					success = true
				}
			}
		case "errmsg":
			if str, ok := elem.Value.StringValueOK(); ok {
				errmsg = str
			}
		case "code":
			if c, ok := elem.Value.Int32OK(); ok {
				errCode = c
			}
		}
	}

	if success {
		return true, ""
	}

	fullErrMsg := fmt.Sprintf("Error code %d: %s", errCode, errmsg)
	return false, fullErrMsg
}
