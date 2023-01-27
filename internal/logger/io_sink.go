package logger

import (
	"io"
	"log"
)

// IOSink writes to an io.Writer using the standard library logging solution and
// is the default sink for the logger, with the default IO being os.Stderr.
type IOSink struct {
	log *log.Logger
}

// Compiile-time check to ensure osSink implements the LogSink interface.
var _ LogSink = &IOSink{}

// NewIOSink will create a new IOSink that writes to the provided io.Writer.
func NewIOSink(out io.Writer) *IOSink {
	return &IOSink{
		log: log.New(out, "", log.LstdFlags),
	}
}

func logCommandStartedMessage(log *log.Logger, kvMap map[string]interface{}) {
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

func logCommandSucceededMessage(log *log.Logger, kvMap map[string]interface{}) {
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

func logCommandFailedMessage(log *log.Logger, kvMap map[string]interface{}) {
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

func logPoolCreatedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection pool created for %s:%d using options " +
		"maxIdleTimeMS=%d, minPoolSize=%d, maxPoolSize=%d, " +
		"maxConnecting=%d"

	log.Printf(format,
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["maxIdleTimeMS"],
		kvMap["minPoolSize"],
		kvMap["maxPoolSize"],
		kvMap["maxConnecting"])
}

func (osSink *IOSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	kvMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		kvMap[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	switch msg {
	case CommandStarted:
		logCommandStartedMessage(osSink.log, kvMap)
	case CommandSucceeded:
		logCommandSucceededMessage(osSink.log, kvMap)
	case CommandFailed:
		logCommandFailedMessage(osSink.log, kvMap)
	case ConnectionPoolCreated:
		logPoolCreatedMessage(osSink.log, kvMap)
	}
}

func (osSink *IOSink) Error(err error, msg string, kv ...interface{}) {
	osSink.Info(0, msg, kv...)
}
