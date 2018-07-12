package event

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// CommandMetadata contains metadata about a command sent to the server.
type CommandMetadata struct {
	Name string
	Time time.Time
}

// CreateMetadata creates metadata for a command.
func CreateMetadata(name string) *CommandMetadata {
	return &CommandMetadata{
		Name: name,
		Time: time.Now(),
	}
}

// TimeDifference returns the difference between now and the time a command was sent in nanoseconds.
func (cm *CommandMetadata) TimeDifference() int64 {
	t := time.Now()
	duration := t.Sub(cm.Time)
	return duration.Nanoseconds()
}

// CommandStartedEvent represents an event generated when a command is sent to a server.
type CommandStartedEvent struct {
	Command      *bson.Document
	DatabaseName string
	CommandName  string
	RequestID    int64
	ConnectionID string
}

// CommandFinishedEvent represents a generic command finishing.
type CommandFinishedEvent struct {
	DurationNanos int64
	CommandName   string
	RequestID     int64
	ConnectionID  string
}

// CommandSucceededEvent represents an event generated when a command's execution succeeds.
type CommandSucceededEvent struct {
	CommandFinishedEvent
	Reply *bson.Document
}

// CommandFailedEvent represents an event generated when a command's execution fails.
type CommandFailedEvent struct {
	CommandFinishedEvent
	Failure string
}

// CommandMonitor represents a monitor that is triggered for different events.
type CommandMonitor struct {
	Started   func(*CommandStartedEvent)
	Succeeded func(*CommandSucceededEvent)
	Failed    func(*CommandFailedEvent)
}
