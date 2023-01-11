package logger

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	CommandMessageFailedDefault    = "Command failed"
	CommandMessageStartedDefault   = "Command started"
	CommandMessageSucceededDefault = "Command succeeded"
	CommandMessageDroppedDefault   = "Command dropped due to full log buffer"
)

type CommandMessage struct {
	DriverConnectionID int32
	MessageLiteral     string
	Name               string
	OperationID        int32
	RequestID          int64
	ServerConnectionID int32
	ServerHost         string
	ServerPort         int32
}

func (*CommandMessage) Component() Component {
	return ComponentCommand
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
	CommandMessage

	Command      bson.Raw
	DatabaseName string
}

func (msg *CommandStartedMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"command", msg.Command,
		"databaseName", msg.DatabaseName,
	}...)
}

type CommandSucceededMessage struct {
	CommandMessage

	Duration time.Duration
	Reply    bson.Raw
}

func (msg *CommandSucceededMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"durationMS", msg.Duration / time.Millisecond,
		"reply", msg.Reply,
	}...)
}

type CommandFailedMessage struct {
	CommandMessage

	Duration time.Duration
	Failure  string
}

func (msg *CommandFailedMessage) Serialize() []interface{} {
	return append(serializeKeysAndValues(msg.CommandMessage), []interface{}{
		"message", msg.MessageLiteral,
		"durationMS", msg.Duration / time.Millisecond,
		"failure", msg.Failure,
	}...)
}

type CommandMessageDropped struct {
	CommandMessage
}

func (msg *CommandMessageDropped) Serialize() []interface{} {
	msg.MessageLiteral = CommandMessageDroppedDefault

	return serializeKeysAndValues(msg.CommandMessage)
}
