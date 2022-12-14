package logger

const (
	CommandMessageStarted   = "Command started"
	CommandMessageSucceeded = "Command succeeded"
)

type CommandStartedMessage struct {
	Name       string `bson:"commandName"`
	RequestID  int64  `bson:"requestId"`
	ServerHost string `bson:"serverHost"`
	Msg        string `bson:"message"`
	Database   string `bson:"databaseName"`
}

func (*CommandStartedMessage) Component() Component {
	return CommandComponent
}

func (msg *CommandStartedMessage) KeysAndValues() []interface{} {
	return []interface{}{
		"message", msg.Msg,
		"databaseName", msg.Database,
		"commandName", msg.Name,
	}
}

func (msg *CommandStartedMessage) Message() string {
	return msg.Msg
}

type CommandSucceededMessage struct {
	Name       string `bson:"commandName"`
	RequestID  int64  `bson:"requestId"`
	ServerHost string `bson:"serverHost"`
	ServerPort int32  `bson:"serverPort"`
	Msg        string `bson:"message"`
	DurationMS int64  `bson:"durationMS"`
	Reply      string `bson:"reply0"`
	ReplyRaw   []byte `bson:"reply"`
}

func (*CommandSucceededMessage) Component() Component {
	return CommandComponent
}

func (msg *CommandSucceededMessage) KeysAndValues() []interface{} {
	return []interface{}{
		"commandName", msg.Name,
		"requestId", msg.RequestID,
		"message", msg.Msg,
		"durationMS", msg.DurationMS,
		"reply", msg.Reply,
		"serverHost", msg.ServerHost,
		"serverPort", msg.ServerPort,
	}
}

func (msg *CommandSucceededMessage) Message() string {
	return msg.Msg
}
