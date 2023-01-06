package logger

const (
	CommandMessageFailedDefault    = "Command failed"
	CommandMessageStartedDefault   = "Command started"
	CommandMessageSucceededDefault = "Command succeeded"
	CommandMessageDroppedDefault   = "Command dropped due to full log buffer"
)

type CommandMessage struct {
	DriverConnectionID int32  `bson:"driverConnectionId"`
	MessageLiteral     string `bson:"message"`
	Name               string `bson:"commandName"`
	OperationID        int32  `bson:"operationId"`
	RequestID          int64  `bson:"requestId"`
	ServerConnectionID int32  `bson:"serverConnectionId"`
	ServerHost         string `bson:"serverHost"`
	ServerPort         int32  `bson:"serverPort"`
}

func (*CommandMessage) Component() Component {
	return CommandComponent
}

func (msg *CommandMessage) Message() string {
	return msg.MessageLiteral
}

func serializeKeysAndValues(msg CommandMessage) []interface{} {
	return []interface{}{
		"commandName", msg.Name,
		"driverConnectionId", msg.DriverConnectionID,
		"message", msg.MessageLiteral,
		"operationId", msg.OperationID,
		"requestId", msg.RequestID,
		"serverConnectionId", msg.ServerConnectionID,
		"serverHost", msg.ServerHost,
		"serverPort", msg.ServerPort,
	}
}

type CommandStartedMessage struct {
	CommandMessage `bson:"-"`

	Command      string `bson:"command"`
	DatabaseName string `bson:"databaseName"`
}

func (msg *CommandStartedMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"command", msg.Command,
		"databaseName", msg.DatabaseName,
	}...)
}

type CommandSucceededMessage struct {
	CommandMessage `bson:"-"`

	DurationMS int64  `bson:"durationMS"`
	Reply      string `bson:"reply"`
}

func (msg *CommandSucceededMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"durationMS", msg.DurationMS,
		"reply", msg.Reply,
	}...)
}

type CommandFailedMessage struct {
	CommandMessage `bson:"-"`

	DurationMS int64  `bson:"durationMS"`
	Failure    string `bson:"failure"`
}

func (msg *CommandFailedMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"durationMS", msg.DurationMS,
		"failure", msg.Failure,
	}...)
}

type CommandMessageDropped struct {
	CommandMessage `bson:"-"`
}

func (msg *CommandMessageDropped) Serialize() []interface{} {
	msg.MessageLiteral = CommandMessageDroppedDefault

	return serializeKeysAndValues(msg.CommandMessage)
}
