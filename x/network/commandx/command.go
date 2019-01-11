package commandx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

// ErrMultiDocCommandResponse occurs when the server sent multiple documents in response to a command.
var ErrMultiDocCommandResponse = errors.New("command returned multiple documents")

// ErrNoDocCommandResponse occurs when the server indicated a response existed, but none was found.
var ErrNoDocCommandResponse = errors.New("command returned no documents")

// ErrUnknownCommandFailure occurs when a command fails for an unknown reason.
var ErrUnknownCommandFailure = errors.New("unknown command failure")

// ErrNoCommandResponse occurs when the server sent no response document to a command.
var ErrNoCommandResponse = errors.New("no command response document")

// ErrDocumentTooLarge occurs when a document that is larger than the maximum size accepted by a
// server is passed to an insert command.
var ErrDocumentTooLarge = errors.New("an inserted document is too large")

// ErrNonPrimaryRP occurs when a nonprimary read preference is used with a transaction.
var ErrNonPrimaryRP = errors.New("read preference in a transaction must be primary")

// UnknownTransactionCommitResult is an error label for unknown transaction commit results.
var UnknownTransactionCommitResult = "UnknownTransactionCommitResult"

// TransientTransactionError is an error label for transient errors with transactions.
var TransientTransactionError = "TransientTransactionError"

// NetworkError is an error label for network errors.
var NetworkError = "NetworkError"

// ReplyDocumentMismatch is an error label for OP_QUERY field mismatch errors.
var ReplyDocumentMismatch = "malformed OP_REPLY: NumberReturned does not match number of documents returned"

var retryableCodes = []int32{11600, 11602, 10107, 13435, 13436, 189, 91, 7, 6, 89, 9001}

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
	Labels  []string
	Name    string
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

// helper method to extract an error from a reader if there is one; first returned item is the
// error if it exists, the second holds parsing errors
func extractError(rdr bsoncore.Document) error {
	var errmsg, codeName string
	var code int32
	var labels []string
	elems, err := rdr.Elements()
	if err != nil {
		return err
	}

	for _, elem := range elems {
		switch elem.Key() {
		case "ok":
			switch elem.Value().Type {
			case bson.TypeInt32:
				if elem.Value().Int32() == 1 {
					return nil
				}
			case bson.TypeInt64:
				if elem.Value().Int64() == 1 {
					return nil
				}
			case bson.TypeDouble:
				if elem.Value().Double() == 1 {
					return nil
				}
			}
		case "errmsg":
			if str, okay := elem.Value().StringValueOK(); okay {
				errmsg = str
			}
		case "codeName":
			if str, okay := elem.Value().StringValueOK(); okay {
				codeName = str
			}
		case "code":
			if c, okay := elem.Value().Int32OK(); okay {
				code = c
			}
		case "errorLabels":
			if arr, okay := elem.Value().ArrayOK(); okay {
				elems, err := arr.Elements()
				if err != nil {
					continue
				}
				for _, elem := range elems {
					if str, ok := elem.Value().StringValueOK(); ok {
						labels = append(labels, str)
					}
				}

			}
		}
	}

	if errmsg == "" {
		errmsg = "command failed"
	}

	return Error{
		Code:    code,
		Message: errmsg,
		Name:    codeName,
		Labels:  labels,
	}
}
