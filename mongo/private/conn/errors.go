// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
)

var (
	// ErrUnknownCommandFailure occurs when a command fails for an unknown reason.
	ErrUnknownCommandFailure = errors.New("unknown command failure")
	// ErrNoCommandResponse occurs when the server sent no response document to a command.
	ErrNoCommandResponse = errors.New("no command response document")
	// ErrMultiDocCommandResponse occurs when the server sent multiple documents in response to a command.
	ErrMultiDocCommandResponse = errors.New("command returned multiple documents")
	// ErrNoDocCommandResponse occurs when the server indicated a response existed, but none was found.
	ErrNoDocCommandResponse = errors.New("command returned no documents")
)

// CommandFailureError is an error with a failure response as a document.
type CommandFailureError struct {
	Msg      string
	Response bson.Reader
}

func (e *CommandFailureError) Error() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Response)
}

// Message retrieves the message of the error.
func (e *CommandFailureError) Message() string {
	return e.Msg
}

// CommandResponseError is an error in the response to a command.
type CommandResponseError struct {
	Message string
}

// NewCommandResponseError creates a new CommandResponseError.
func NewCommandResponseError(msg string) *CommandResponseError {
	return &CommandResponseError{msg}
}

func (e *CommandResponseError) Error() string {
	return e.Message
}

// CommandError is an error in the execution of a command.
type CommandError struct {
	Code    int32
	Message string
	Name    string
}

func (e *CommandError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("(%v) %v", e.Name, e.Message)
	}
	return e.Message
}

// IsNsNotFound indicates if the error is about a namespace not being found.
func IsNsNotFound(err error) bool {
	e, ok := err.(*CommandError)
	return ok && (e.Code == 26)
}

// IsCommandNotFound indicates if the error is about a command not being found.
func IsCommandNotFound(err error) bool {
	e, ok := err.(*CommandError)
	return ok && (e.Code == 59 || e.Code == 13390 || strings.HasPrefix(e.Message, "no such cmd:"))
}
