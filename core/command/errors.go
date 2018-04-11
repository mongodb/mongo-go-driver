// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"errors"
	"fmt"

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

// QueryFailureError is an error representing a command failure as a document.
type QueryFailureError struct {
	Message  string
	Response bson.Reader
}

// Error implements the error interface.
func (e QueryFailureError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Response)
}

// ResponseError is an error parsing the response to a command.
type ResponseError struct {
	Message string
	Wrapped error
}

// NewCommandResponseError creates a CommandResponseError.
func NewCommandResponseError(msg string, err error) ResponseError {
	return ResponseError{Message: msg, Wrapped: err}
}

// Error implements the error interface.
func (e ResponseError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Wrapped)
	}
	return fmt.Sprintf("%s", e.Message)
}

// Error is a command execution error from the database.
type Error struct {
	Code    int32
	Message string
	Name    string
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("(%v) %v", e.Name, e.Message)
	}
	return e.Message
}

// IsNotFound indicates if the error is from a namespace not being found.
func IsNotFound(err error) bool {
	e, ok := err.(Error)
	return ok && (e.Code == 26)
}
