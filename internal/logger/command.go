package logger

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const DefaultCommandSucceededMessageMessage = "Command succeeded"

type Command struct {
	// Name is the name of the command.
	Name string `bson:"name"`

	// RequestID is the driver-generated request ID for the command.
	RequestID int64 `bson:"requestID"`
}

func (cmd *Command) KeysAndValues() []interface{} {
	return []interface{}{
		"name", cmd.Name,
		"requestID", cmd.RequestID,
	}
}

type CommandSucceededMessage struct {
	*Command

	Message    string `bson:"message"`
	DurationMS int64  `bson:"durationMS"`
	Reply      string `bson:"reply"`
}

func NewCommandSuccessMessage(duration time.Duration, reply bson.Raw, cmd *Command) *CommandSucceededMessage {
	return &CommandSucceededMessage{
		Command:    cmd,
		Message:    DefaultCommandSucceededMessageMessage,
		DurationMS: duration.Milliseconds(),
		Reply:      reply.String(),
	}
}

func (*CommandSucceededMessage) Component() LogComponent {
	return CommandLogComponent
}

func (msg *CommandSucceededMessage) ExtJSONBytes() ([]byte, error) {
	return bson.MarshalExtJSON(msg, false, false)
}

func (msg *CommandSucceededMessage) KeysAndValues() []interface{} {
	return []interface{}{
		"command", msg.Command.KeysAndValues(),
		"message", msg.Message,
		"durationMS", msg.DurationMS,
		"reply", msg.Reply,
	}
}
