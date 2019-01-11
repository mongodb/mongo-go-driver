package driverx

import (
	"bytes"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var retryableCodes = []int32{11600, 11602, 10107, 13435, 13436, 189, 91, 7, 6, 89, 9001}

// QueryFailureError is an error representing a command failure as a document.
type QueryFailureError struct {
	Message  string
	Response bsoncore.Document
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

// WriteCommandError is an error for a write command.
type WriteCommandError struct {
	WriteConcernError *WriteConcernError
	WriteErrors       WriteErrors
}

func (wce WriteCommandError) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "write command error: [")
	fmt.Fprintf(&buf, "{%s}, ", wce.WriteErrors)
	fmt.Fprintf(&buf, "{%s}]", wce.WriteConcernError)
	return buf.String()
}

// Retryable returns true if the error is retryable
func (wce WriteCommandError) Retryable() bool {
	if wce.WriteConcernError == nil {
		return false
	}
	return (*wce.WriteConcernError).Retryable()
}

// WriteConcernError is a write concern failure that occurred as a result of a
// write operation.
type WriteConcernError struct {
	Code    int64
	Message string
	Details bsoncore.Document
}

func (wce WriteConcernError) Error() string { return wce.Message }

// Retryable returns true if the error is retryable
func (wce WriteConcernError) Retryable() bool {
	for _, code := range retryableCodes {
		if wce.Code == int64(code) {
			return true
		}
	}
	if strings.Contains(wce.Message, "not master") || strings.Contains(wce.Message, "node is recovering") {
		return true
	}

	return false
}

// WriteError is a non-write concern failure that occurred as a result of a write
// operation.
type WriteError struct {
	Index   int64
	Code    int64
	Message string
}

func (we WriteError) Error() string { return we.Message }

// WriteErrors is a group of non-write concern failures that occurred as a result
// of a write operation.
type WriteErrors []WriteError

func (we WriteErrors) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "write errors: [")
	for idx, err := range we {
		if idx != 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprintf(&buf, "{%s}", err)
	}
	fmt.Fprint(&buf, "]")
	return buf.String()
}

// ErrorKind defines the kind of error that an Error represents.
type ErrorKind uint

const (
	EKGeneric ErrorKind = iota
	EKWriteConcern
	EKWriteError
)

// Error is a command execution error from the database.
type Error struct {
	Kind    ErrorKind
	Code    int32
	Message string
	Labels  []string
	Name    string
	Err     error
	Info    bsoncore.Document
	Index   int
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("(%v) %v", e.Name, e.Message)
	}
	return e.Message
}

// HasErrorLabel returns true if the error contains the specified label.
func (e Error) HasErrorLabel(label string) bool {
	if e.Labels != nil {
		for _, l := range e.Labels {
			if l == label {
				return true
			}
		}
	}
	return false
}

// Retryable returns true if the error is retryable
func (e Error) Retryable() bool {
	for _, label := range e.Labels {
		if label == NetworkError {
			return true
		}
	}
	for _, code := range retryableCodes {
		if e.Code == code {
			return true
		}
	}
	if strings.Contains(e.Message, "not master") || strings.Contains(e.Message, "node is recovering") {
		return true
	}

	return false
}

// Errors combines multiple errors into a single error. Mostly used for write errors.
type Errors []Error

func (errs Errors) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "multiple errors: [")
	for idx, err := range errs {
		if idx != 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprintf(&buf, "{%s}", err)
	}
	fmt.Fprint(&buf, "]")
	return buf.String()
}

// writeConcernErrorRetryable returns true if the provided write concern error is retryable.
func writeConcernErrorRetryable(val bsoncore.Value) bool {
	wce, ok := val.DocumentOK()
	if !ok {
		return false
	}
	if wceCode := wce.Lookup("code"); wceCode.Type == bsontype.Int32 || wceCode.Type == bsontype.Int64 {
		var c int32
		if i32, ok := wceCode.Int32OK(); ok {
			c = i32
		}
		if i64, ok := wceCode.Int64OK(); ok {
			c = int32(i64)
		}
		for _, code := range retryableCodes {
			if c == code {
				return true
			}
		}
	}
	errMsg, ok := wce.Lookup("errmsg").StringValueOK()
	if !ok {
		return false
	}
	return strings.Contains(errMsg, "not master") || strings.Contains(errMsg, "node is recovering")
}
