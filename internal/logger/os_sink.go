package logger

import (
	"io"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

type osSink struct {
	log *log.Logger
}

func newOSSink(out io.Writer) *osSink {
	return &osSink{
		log: log.New(out, "", log.LstdFlags),
	}
}

// TODO: (GODRIVERS-2570) Figure out how to handle errors from unmarshalMessage.
func unmarshalMessage(msg interface{}, args ...interface{}) {
	actualD := bson.D{}
	for i := 0; i < len(args); i += 2 {
		actualD = append(actualD, bson.E{Key: args[i].(string), Value: args[i+1]})
	}

	bytes, _ := bson.Marshal(actualD)
	bson.Unmarshal(bytes, msg)
}

func logCommandMessageStarted(log *log.Logger, args ...interface{}) {
	var csm CommandStartedMessage
	unmarshalMessage(&csm, args...)

	format := "Command %q started on database %q using a connection with server-generated ID %d to %s:%d. " +
		"The requestID is %d and the operation ID is %d. Command: %s"

	log.Printf(format,
		csm.Name,
		csm.DatabaseName,
		csm.ServerConnectionID,
		csm.ServerHost,
		csm.ServerPort,
		csm.RequestID,
		csm.OperationID,
		csm.Command)

}

func logCommandMessageSucceeded(log *log.Logger, args ...interface{}) {
	var csm CommandSucceededMessage
	unmarshalMessage(&csm, args...)

	format := "Command %q succeeded in %d ms using server-generated ID %d to %s:%d. " +
		"The requestID is %d and the operation ID is %d. Command reply: %s"

	log.Printf(format,
		csm.Name,
		csm.DurationMS,
		csm.ServerConnectionID,
		csm.ServerHost,
		csm.ServerPort,
		csm.RequestID,
		csm.OperationID,
		csm.Reply)
}

func logCommandMessageFailed(log *log.Logger, args ...interface{}) {
	var cfm CommandFailedMessage
	unmarshalMessage(&cfm, args...)

	format := "Command %q failed in %d ms using a connection with server-generated ID %d to %s:%d. " +
		" The requestID is %d and the operation ID is %d. Error: %s"

	log.Printf(format,
		cfm.Name,
		cfm.DurationMS,
		cfm.ServerConnectionID,
		cfm.ServerHost,
		cfm.ServerPort,
		cfm.RequestID,
		cfm.OperationID,
		cfm.Failure)
}

func logCommandDropped(log *log.Logger) {
	log.Println(CommandMessageDroppedDefault)
}

func (osSink *osSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	// TODO: (GODRIVERS-2570) This is how the specification says we SHOULD handle errors. It might be much
	// TODO: better to just pass the message and then the keys and values ala
	// TODO: "msg: %s, key1: %v, key2: %v, key3: %v, ...".
	switch msg {
	case CommandMessageStartedDefault:
		logCommandMessageStarted(osSink.log, keysAndValues...)
	case CommandMessageSucceededDefault:
		logCommandMessageSucceeded(osSink.log, keysAndValues...)
	case CommandMessageFailedDefault:
		logCommandMessageFailed(osSink.log, keysAndValues...)
	case CommandMessageDroppedDefault:
		logCommandDropped(osSink.log)
	}
}
