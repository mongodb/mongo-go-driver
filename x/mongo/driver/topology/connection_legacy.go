package topology

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/compressor"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

var emptyDoc bson.Raw

type connectionLegacy struct {
	*connection
	writeBuf []byte
	readBuf  []byte
	monitor  *event.CommandMonitor
	cmdMap   map[int64]*commandMetadata // map for monitoring commands sent to server

	wireMessageBuf []byte                // buffer to store uncompressed wire message before compressing
	compressBuf    []byte                // buffer to compress messages
	uncompressBuf  []byte                // buffer to uncompress messages
	compressor     compressor.Compressor // use for compressing messages
	// server can compress response with any compressor supported by driver
	compressorMap map[wiremessage.CompressorID]compressor.Compressor

	s *Server

	sync.RWMutex
}

func newConnectionLegacy(c *connection, s *Server, opts ...ConnectionOption) (*connectionLegacy, error) {
	cfg, err := newConnectionConfig(opts...)
	if err != nil {
		return nil, err
	}

	compressorMap := make(map[wiremessage.CompressorID]compressor.Compressor)

	for _, comp := range cfg.compressors {
		switch comp {
		case "snappy":
			snappyComp := compressor.CreateSnappy()
			compressorMap[snappyComp.CompressorID()] = snappyComp
		case "zlib":
			zlibComp, err := compressor.CreateZlib(cfg.zlibLevel)
			if err != nil {
				return nil, err
			}

			compressorMap[zlibComp.CompressorID()] = zlibComp
		}
	}

	cl := &connectionLegacy{
		connection:     c,
		compressorMap:  compressorMap,
		cmdMap:         make(map[int64]*commandMetadata),
		compressBuf:    make([]byte, 256),
		readBuf:        make([]byte, 256),
		uncompressBuf:  make([]byte, 256),
		writeBuf:       make([]byte, 0, 256),
		wireMessageBuf: make([]byte, 256),

		s: s,
	}

	d := c.desc
	if len(d.Compression) > 0 {
	clientMethodLoop:
		for _, comp := range cl.compressorMap {
			method := comp.Name()

			for _, serverMethod := range d.Compression {
				if method != serverMethod {
					continue
				}

				cl.compressor = comp // found matching compressor
				break clientMethodLoop
			}
		}
	}

	cl.monitor = cfg.cmdMonitor // Attach the command monitor later to avoid monitoring auth.
	return cl, nil
}

func (c *connectionLegacy) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	// Truncate the write buffer
	c.writeBuf = c.writeBuf[:0]

	messageToWrite := wm
	// Compress if possible
	if c.compressor != nil {
		compressed, err := c.compressMessage(wm)
		if err != nil {
			return ConnectionError{
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to compress wire message",
			}
		}
		messageToWrite = compressed
	}

	var err error
	c.writeBuf, err = messageToWrite.AppendWireMessage(c.writeBuf)
	if err != nil {
		return ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to encode wire message",
		}
	}

	err = c.writeWireMessage(ctx, c.writeBuf)
	if c.s != nil {
		c.s.ProcessError(err)
	}
	if err != nil {
		// The error we got back was probably a ConnectionError already, so we don't really need to
		// wrap it here.
		return ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to write wire message to network",
		}
	}

	return c.commandStartedEvent(ctx, wm)
}

func (c *connectionLegacy) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	// Truncate the write buffer
	c.readBuf = c.readBuf[:0]

	var err error
	c.readBuf, err = c.readWireMessage(ctx, c.readBuf)
	if c.s != nil {
		c.s.ProcessError(err)
	}
	if err != nil {
		// The error we got back was probably a ConnectionError already, so we don't really need to
		// wrap it here.
		return nil, ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to read wire message from network",
		}
	}
	hdr, err := wiremessage.ReadHeader(c.readBuf, 0)
	if err != nil {
		return nil, ConnectionError{
			ConnectionID: c.id,
			Wrapped:      err,
			message:      "unable to decode header",
		}
	}

	messageToDecode := c.readBuf
	opcodeToCheck := hdr.OpCode

	if hdr.OpCode == wiremessage.OpCompressed {
		var compressed wiremessage.Compressed
		err := compressed.UnmarshalWireMessage(c.readBuf)
		if err != nil {
			return nil, ConnectionError{
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to decode OP_COMPRESSED",
			}
		}

		uncompressed, origOpcode, err := c.uncompressMessage(compressed)
		if err != nil {
			return nil, ConnectionError{
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to uncompress message",
			}
		}
		messageToDecode = uncompressed
		opcodeToCheck = origOpcode
	}

	var wm wiremessage.WireMessage
	switch opcodeToCheck {
	case wiremessage.OpReply:
		var r wiremessage.Reply
		err := r.UnmarshalWireMessage(messageToDecode)
		if err != nil {
			return nil, ConnectionError{
				ConnectionID: c.id,
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
				ConnectionID: c.id,
				Wrapped:      err,
				message:      "unable to decode OP_MSG",
			}
		}
		wm = reply
	default:
		return nil, ConnectionError{
			ConnectionID: c.id,
			message:      fmt.Sprintf("opcode %s not implemented", hdr.OpCode),
		}
	}

	if c.s != nil {
		c.s.ProcessError(command.DecodeError(wm))
	}

	// TODO: do we care if monitoring fails?
	return wm, c.commandFinishedEvent(ctx, wm)
}

func (c *connectionLegacy) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.connection == nil {
		return nil
	}
	if c.s != nil {
		c.s.sem.Release(1)
	}
	err := c.pool.put(c.connection)
	if err != nil {
		return err
	}
	c.connection = nil
	return nil
}

func (c *connectionLegacy) Expired() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connection == nil || c.expired()
}

func (c *connectionLegacy) Alive() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connection != nil
}

func (c *connectionLegacy) ID() string {
	c.RLock()
	defer c.RUnlock()
	if c.connection == nil {
		return "<closed>"
	}
	return c.id
}

func (c *connectionLegacy) commandStartedEvent(ctx context.Context, wm wiremessage.WireMessage) error {
	if c.monitor == nil || c.monitor.Started == nil {
		return nil
	}

	startedEvent := &event.CommandStartedEvent{
		ConnectionID: c.id,
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

	c.monitor.Started(ctx, startedEvent)

	if !acknowledged {
		if c.monitor.Succeeded == nil {
			return nil
		}

		// unack writes must provide a CommandSucceededEvent with an { ok: 1 } reply
		finishedEvent := event.CommandFinishedEvent{
			DurationNanos: 0,
			CommandName:   startedEvent.CommandName,
			RequestID:     startedEvent.RequestID,
			ConnectionID:  c.id,
		}

		c.monitor.Succeeded(ctx, &event.CommandSucceededEvent{
			CommandFinishedEvent: finishedEvent,
			Reply:                bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "ok", 1)),
		})

		return nil
	}

	c.cmdMap[startedEvent.RequestID] = createMetadata(startedEvent.CommandName, legacy, fullCollName)
	return nil
}

func (c *connectionLegacy) commandFinishedEvent(ctx context.Context, wm wiremessage.WireMessage) error {
	if c.monitor == nil {
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
	cmdMetadata := c.cmdMap[requestID]
	delete(c.cmdMap, requestID)

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

	if (success && c.monitor.Succeeded == nil) || (!success && c.monitor.Failed == nil) {
		return nil
	}

	finishedEvent := event.CommandFinishedEvent{
		DurationNanos: cmdMetadata.TimeDifference(),
		CommandName:   cmdMetadata.Name,
		RequestID:     requestID,
		ConnectionID:  c.id,
	}

	if success {
		if !canMonitor(finishedEvent.CommandName) {
			successEvent := &event.CommandSucceededEvent{
				Reply:                emptyDoc,
				CommandFinishedEvent: finishedEvent,
			}
			c.monitor.Succeeded(ctx, successEvent)
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

		c.monitor.Succeeded(ctx, successEvent)
		return nil
	}

	failureEvent := &event.CommandFailedEvent{
		Failure:              errmsg,
		CommandFinishedEvent: finishedEvent,
	}

	c.monitor.Failed(ctx, failureEvent)
	return nil
}

func canCompress(cmd string) bool {
	if cmd == "isMaster" || cmd == "saslStart" || cmd == "saslContinue" || cmd == "getnonce" || cmd == "authenticate" ||
		cmd == "createUser" || cmd == "updateUser" || cmd == "copydbSaslStart" || cmd == "copydbgetnonce" || cmd == "copydb" {
		return false
	}
	return true
}

func (c *connectionLegacy) compressMessage(wm wiremessage.WireMessage) (wiremessage.WireMessage, error) {
	var requestID int32
	var responseTo int32
	var origOpcode wiremessage.OpCode

	switch converted := wm.(type) {
	case wiremessage.Query:
		firstElem, err := converted.Query.IndexErr(0)
		if err != nil {
			return wiremessage.Compressed{}, err
		}

		key := firstElem.Key()
		if !canCompress(key) {
			return wm, nil // return original message because this command can't be compressed
		}
		requestID = converted.MsgHeader.RequestID
		origOpcode = wiremessage.OpQuery
		responseTo = converted.MsgHeader.ResponseTo
	case wiremessage.Msg:
		firstElem, err := converted.Sections[0].(wiremessage.SectionBody).Document.IndexErr(0)
		if err != nil {
			return wiremessage.Compressed{}, err
		}

		key := firstElem.Key()
		if !canCompress(key) {
			return wm, nil
		}

		requestID = converted.MsgHeader.RequestID
		origOpcode = wiremessage.OpMsg
		responseTo = converted.MsgHeader.ResponseTo
	}

	// can compress
	c.wireMessageBuf = c.wireMessageBuf[:0] // truncate
	var err error
	c.wireMessageBuf, err = wm.AppendWireMessage(c.wireMessageBuf)
	if err != nil {
		return wiremessage.Compressed{}, err
	}

	c.wireMessageBuf = c.wireMessageBuf[16:] // strip header
	c.compressBuf = c.compressBuf[:0]
	compressedBytes, err := c.compressor.CompressBytes(c.wireMessageBuf, c.compressBuf)
	if err != nil {
		return wiremessage.Compressed{}, err
	}

	compressedMessage := wiremessage.Compressed{
		MsgHeader: wiremessage.Header{
			// MessageLength and OpCode will be set when marshalling wire message by SetDefaults()
			RequestID:  requestID,
			ResponseTo: responseTo,
		},
		OriginalOpCode:    origOpcode,
		UncompressedSize:  int32(len(c.wireMessageBuf)), // length of uncompressed message excluding MsgHeader
		CompressorID:      wiremessage.CompressorID(c.compressor.CompressorID()),
		CompressedMessage: compressedBytes,
	}

	return compressedMessage, nil
}

// returns []byte of uncompressed message with reconstructed header, original opcode, error
func (c *connectionLegacy) uncompressMessage(compressed wiremessage.Compressed) ([]byte, wiremessage.OpCode, error) {
	// server doesn't guarantee the same compression method will be used each time so the CompressorID field must be
	// used to find the correct method for uncompressing data
	uncompressor := c.compressorMap[compressed.CompressorID]

	// reset uncompressBuf
	c.uncompressBuf = c.uncompressBuf[:0]
	if int(compressed.UncompressedSize) > cap(c.uncompressBuf) {
		c.uncompressBuf = make([]byte, 0, compressed.UncompressedSize)
	}

	uncompressedMessage, err := uncompressor.UncompressBytes(compressed.CompressedMessage, c.uncompressBuf[:compressed.UncompressedSize])

	if err != nil {
		return nil, 0, err
	}

	origHeader := wiremessage.Header{
		MessageLength: int32(len(uncompressedMessage)) + 16, // add 16 for original header
		RequestID:     compressed.MsgHeader.RequestID,
		ResponseTo:    compressed.MsgHeader.ResponseTo,
	}

	switch compressed.OriginalOpCode {
	case wiremessage.OpReply:
		origHeader.OpCode = wiremessage.OpReply
	case wiremessage.OpMsg:
		origHeader.OpCode = wiremessage.OpMsg
	default:
		return nil, 0, fmt.Errorf("opcode %s not implemented", compressed.OriginalOpCode)
	}

	var fullMessage []byte
	fullMessage = origHeader.AppendHeader(fullMessage)
	fullMessage = append(fullMessage, uncompressedMessage...)
	return fullMessage, origHeader.OpCode, nil
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
