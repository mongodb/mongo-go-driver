package connection

import "time"

// commandMetadata contains metadata about a command sent to the server.
type commandMetadata struct {
	Name string
	Time time.Time
}

// createMetadata creates metadata for a command.
func createMetadata(name string) *commandMetadata {
	return &commandMetadata{
		Name: name,
		Time: time.Now(),
	}
}

// TimeDifference returns the difference between now and the time a command was sent in nanoseconds.
func (cm *commandMetadata) TimeDifference() int64 {
	t := time.Now()
	duration := t.Sub(cm.Time)
	return duration.Nanoseconds()
}
