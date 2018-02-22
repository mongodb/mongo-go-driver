// Package result contains the results from various operations.
package result

// Delete is a result from a Delete command.
type Delete struct{}

// Document is a result from a command that returns a single Document.
type Document struct{}

// Update is a result of an Update command.
type Update struct{}

// IsMaster is a result of an IsMaster command.
type IsMaster struct{}

// BuildInfo is a result of a BuildInfo command.
type BuildInfo struct{}
