package logger

import (
	"io"
	"log"
)

type osSink struct {
	log *log.Logger
}

// Compiile-time check to ensure osSink implements the LogSink interface.
var _ LogSink = &osSink{}

func newOSSink(out io.Writer) *osSink {
	return &osSink{
		log: log.New(out, "", log.LstdFlags),
	}
}

func logCommandMessageStarted(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q started on database %q using a connection with " +
		"server-generated ID %d to %s:%d. The requestID is %d and " +
		"the operation ID is %d. Command: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["databaseName"],
		kvMap["serverConnectionId"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["command"])

}

func logCommandMessageSucceeded(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q succeeded in %d ms using server-generated ID " +
		"%d to %s:%d. The requestID is %d and the operation ID is " +
		"%d. Command reply: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["duration"],
		kvMap["serverConnectionId"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["reply"])
}

func logCommandMessageFailed(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q failed in %d ms using a connection with " +
		"server-generated ID %d to %s:%d. The requestID is %d and " +
		"the operation ID is %d. Error: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["duration"],
		kvMap["serverConnectionID"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["failure"])
}

func logCommandDropped(log *log.Logger) {
	log.Println(CommandMessageDroppedDefault)
}

func (osSink *osSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	kvMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		kvMap[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	switch msg {
	case CommandMessageStartedDefault:
		logCommandMessageStarted(osSink.log, kvMap)
	case CommandMessageSucceededDefault:
		logCommandMessageSucceeded(osSink.log, kvMap)
	case CommandMessageFailedDefault:
		logCommandMessageFailed(osSink.log, kvMap)
	case CommandMessageDroppedDefault:
		logCommandDropped(osSink.log)
	}
}

func (osSink *osSink) Error(err error, msg string, kv ...interface{}) {
	osSink.Info(0, msg, kv...)
}
