package logger

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type CommandSucceededMessage struct {
	Message    string
	DurationMS time.Duration
	Reply      string
}

func (*CommandSucceededMessage) Component() LogComponent {
	return CommandLogComponent
}

func (msg *CommandSucceededMessage) ExtJSONBytes() ([]byte, error) {
	return bson.MarshalExtJSON(msg, false, false)
}

func (msg *CommandSucceededMessage) KeysAndValues() []interface{} {
	return []interface{}{
		"message", msg.Message,
		"durationMS", msg.DurationMS,
		"reply", msg.Reply,
	}
}
