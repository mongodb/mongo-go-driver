package logger

// TODO: add messages to everything

const (
	CommandMessageFailedDefault    = "Command failed"
	CommandMessageStartedDefault   = "Command started"
	CommandMessageSucceededDefault = "Command succeeded"
	CommandMessageDroppedDefault   = "Command dropped due to full log buffer"
)

type CommandMessage struct{}

func (*CommandMessage) Component() Component {
	return CommandComponent
}

type CommandStartedMessage struct {
	CommandMessage `bson:"-"`

	DriverConnectionID int32  `bson:"driverConnectionId"`
	Name               string `bson:"commandName"`
	OperationID        int32  `bson:"operationId"`
	RequestID          int64  `bson:"requestId"`
	ServerHost         string `bson:"serverHost"`
	ServerPort         int32  `bson:"serverPort"`
	ServerConnectionID int32  `bson:"serverConnectionId"`
	Message            string `bson:"message"`
	Command            string `bson:"command"`
	DatabaseName       string `bson:"databaseName"`
}

type CommandSucceededMessage struct {
	CommandMessage `bson:"-"`

	DriverConnectionID int32  `bson:"driverConnectionId"`
	Name               string `bson:"commandName"`
	RequestID          int64  `bson:"requestId"`
	OperationID        int32  `bson:"operationId"`
	ServerHost         string `bson:"serverHost"`
	ServerPort         int32  `bson:"serverPort"`
	ServerConnectionID int32  `bson:"serverConnectionId"`
	Message            string `bson:"message"`
	DurationMS         int64  `bson:"durationMS"`
	Reply              string `bson:"reply"`
}

type CommandFailedMessage struct {
	CommandMessage `bson:"-"`

	DriverConnectionID int32  `bson:"driverConnectionId"`
	Name               string `bson:"commandName"`
	RequestID          int64  `bson:"requestId"`
	OperationID        int32  `bson:"operationId"`
	ServerHost         string `bson:"serverHost"`
	ServerPort         int32  `bson:"serverPort"`
	ServerConnectionID int32  `bson:"serverConnectionId"`
	Message            string `bson:"message"`
	DurationMS         int64  `bson:"durationMS"`
	Failure            string `bson:"failure"`
}

type CommandMessageDropped struct {
	CommandMessage `bson:"-"`

	Message string `bson:"message"`
}
