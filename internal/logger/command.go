package logger

import (
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DefaultMaxDocumentLength is the default maximum number of bytes that can be
// logged for a stringified BSON document.
const DefaultMaxDocumentLength = 1000

// TruncationSuffix are trailling ellipsis "..." appended to a message to
// indicate to the user that truncation occurred. This constant does not count
// toward the max document length.
const TruncationSuffix = "..."

const (
	CommandMessageFailedDefault    = "Command failed"
	CommandMessageStartedDefault   = "Command started"
	CommandMessageSucceededDefault = "Command succeeded"

	// CommandMessageDroppedDefault indicates that a the message was dropped
	// likely due to a full buffer. It is not an indication that the command
	// failed.
	CommandMessageDroppedDefault = "Command message dropped"
)

type CommandMessage struct {
	DriverConnectionID int32
	MessageLiteral     string
	Name               string
	OperationID        int32
	RequestID          int64
	ServerConnectionID *int32
	ServerHost         string
	ServerPort         string
	ServiceID          *primitive.ObjectID
}

func (*CommandMessage) Component() Component {
	return ComponentCommand
}

func (msg *CommandMessage) Message() string {
	return msg.MessageLiteral
}

func serializeKeysAndValues(msg CommandMessage) ([]interface{}, error) {
	keysAndValues := []interface{}{
		"commandName", msg.Name,
		"driverConnectionId", msg.DriverConnectionID,
		"message", msg.MessageLiteral,
		"operationId", msg.OperationID,
		"requestId", msg.RequestID,
		"serverHost", msg.ServerHost,
	}

	// Convert the ServerPort into an integer.
	port, err := strconv.ParseInt(msg.ServerPort, 0, 32)
	if err != nil {
		return nil, err
	}

	keysAndValues = append(keysAndValues, "serverPort", port)

	// Add the "serverConnectionId" if it is not nil.
	if msg.ServerConnectionID != nil {
		keysAndValues = append(keysAndValues,
			"serverConnectionId", *msg.ServerConnectionID)
	}

	// Add the "serviceId" if it is not nil.
	if msg.ServiceID != nil {
		keysAndValues = append(keysAndValues,
			"serviceId", msg.ServiceID.Hex())
	}

	return keysAndValues, nil
}

type CommandStartedMessage struct {
	CommandMessage

	Command      string
	DatabaseName string
}

func (msg *CommandStartedMessage) Serialize(maxDocLen uint) ([]interface{}, error) {
	kv, err := serializeKeysAndValues(msg.CommandMessage)
	if err != nil {
		return nil, err
	}

	return append(kv,
		"message", msg.MessageLiteral,
		"command", formatMessage(msg.Command, maxDocLen),
		"databaseName", msg.DatabaseName), nil
}

type CommandSucceededMessage struct {
	CommandMessage

	Duration time.Duration
	Reply    string
}

func (msg *CommandSucceededMessage) Serialize(maxDocLen uint) ([]interface{}, error) {
	kv, err := serializeKeysAndValues(msg.CommandMessage)
	if err != nil {
		return nil, err
	}

	return append(kv,
		"message", msg.MessageLiteral,
		"durationMS", msg.Duration/time.Millisecond,
		"reply", formatMessage(msg.Reply, maxDocLen)), nil
}

type CommandFailedMessage struct {
	CommandMessage

	Duration time.Duration
	Failure  string
}

func (msg *CommandFailedMessage) Serialize(maxDocLen uint) ([]interface{}, error) {
	kv, err := serializeKeysAndValues(msg.CommandMessage)
	if err != nil {
		return nil, err
	}

	return append(kv,
		"message", msg.MessageLiteral,
		"durationMS", msg.Duration/time.Millisecond,
		"failure", formatMessage(msg.Failure, maxDocLen)), nil
}

func truncate(str string, width uint) string {
	if width == 0 {
		return ""
	}

	if len(str) <= int(width) {
		return str
	}

	// Truncate the byte slice of the string to the given width.
	newStr := str[:width]

	// Check if the last byte is at the beginning of a multi-byte character.
	// If it is, then remove the last byte.
	if newStr[len(newStr)-1]&0xC0 == 0xC0 {
		return newStr[:len(newStr)-1] + TruncationSuffix
	}

	// Check if the last byte is in the middle of a multi-byte character. If
	// it is, then step back until we find the beginning of the character.
	if newStr[len(newStr)-1]&0xC0 == 0x80 {
		for i := len(newStr) - 1; i >= 0; i-- {
			if newStr[i]&0xC0 == 0xC0 {
				return newStr[:i] + TruncationSuffix
			}
		}
	}

	return newStr + TruncationSuffix
}

// formatMessage formats a BSON document for logging. The document is truncated
// to the given "commandWidth".
func formatMessage(msg string, commandWidth uint) string {
	if len(msg) == 0 {
		return "{}"
	}

	return truncate(msg, commandWidth)
}
