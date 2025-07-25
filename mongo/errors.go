// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/codecutil"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mongocrypt"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

// ErrClientDisconnected is returned when disconnected Client is used to run an operation.
var ErrClientDisconnected = errors.New("client is disconnected")

// InvalidArgumentError wraps an invalid argument error.
type InvalidArgumentError struct {
	wrapped error
}

// Error implements the error interface.
func (e InvalidArgumentError) Error() string {
	return e.wrapped.Error()
}

// Unwrap returns the underlying error.
func (e InvalidArgumentError) Unwrap() error {
	return e.wrapped
}

// ErrMultipleIndexDrop is returned if multiple indexes would be dropped from a call to IndexView.DropOne.
var ErrMultipleIndexDrop error = InvalidArgumentError{errors.New("multiple indexes would be dropped")}

// ErrNilDocument is returned when a nil document is passed to a CRUD method.
var ErrNilDocument error = InvalidArgumentError{errors.New("document is nil")}

// ErrNilValue is returned when a nil value is passed to a CRUD method.
var ErrNilValue error = InvalidArgumentError{errors.New("value is nil")}

// ErrEmptySlice is returned when an empty slice is passed to a CRUD method that requires a non-empty slice.
var ErrEmptySlice error = InvalidArgumentError{errors.New("must provide at least one element in input slice")}

// ErrNotSlice is returned when a type other than slice is passed to InsertMany.
var ErrNotSlice error = InvalidArgumentError{errors.New("must provide a non-empty slice")}

// ErrMapForOrderedArgument is returned when a map with multiple keys is passed to a CRUD method for an ordered parameter
type ErrMapForOrderedArgument struct {
	ParamName string
}

// Error implements the error interface.
func (e ErrMapForOrderedArgument) Error() string {
	return fmt.Sprintf("multi-key map passed in for ordered parameter %v", e.ParamName)
}

// wrapErrors wraps error types and values that are defined in "internal" and
// "x" packages with error types and values that are defined in this package.
// That allows users to inspect the errors using errors.Is/errors.As without
// relying on "internal" or "x" packages.
func wrapErrors(err error) error {
	// Return nil when err is nil to avoid costly reflection logic below.
	if err == nil {
		return nil
	}

	// Do not propagate the acknowledgement sentinel error. For DDL commands,
	// (creating indexes, dropping collections, etc) acknowledgement should be
	// ignored. For non-DDL write commands (insert, update, etc), acknowledgement
	// should be be propagated at the result-level: e.g.,
	// SingleResult.Acknowledged.
	if errors.Is(err, driver.ErrUnacknowledgedWrite) {
		return nil
	}
	if errors.Is(err, topology.ErrTopologyClosed) {
		return ErrClientDisconnected
	}

	var de driver.Error
	if errors.As(err, &de) {
		return CommandError{
			Code:    de.Code,
			Message: de.Message,
			Labels:  de.Labels,
			Name:    de.Name,
			Wrapped: err,
			Raw:     bson.Raw(de.Raw),

			// Set wrappedMsgOnly=true here so that the Code and Message are not
			// repeated multiple times in the error string. We expect that the
			// wrapped driver.Error already contains that info in the error
			// string.
			wrappedMsgOnly: true,
		}
	}

	var qe driver.QueryFailureError
	if errors.As(err, &qe) {
		// qe.Message is "command failure"
		ce := CommandError{
			Name:    qe.Message,
			Wrapped: err,
			Raw:     bson.Raw(qe.Response),

			// Don't set wrappedMsgOnly=true here because the code below adds
			// additional error context that is not provided by the
			// driver.QueryFailureError. Additionally, driver.QueryFailureError
			// is only returned when parsing OP_QUERY replies (OP_REPLY), so
			// it's unlikely this block will ever be run now that MongoDB 3.6 is
			// no longer supported.
		}

		dollarErr, err := qe.Response.LookupErr("$err")
		if err == nil {
			ce.Message, _ = dollarErr.StringValueOK()
		}
		code, err := qe.Response.LookupErr("code")
		if err == nil {
			ce.Code, _ = code.Int32OK()
		}

		return ce
	}

	var me mongocrypt.Error
	if errors.As(err, &me) {
		return MongocryptError{
			Code:    me.Code,
			Message: me.Message,
			wrapped: err,

			// Set wrappedMsgOnly=true here so that the Code and Message are not
			// repeated multiple times in the error string. We expect that the
			// wrapped mongocrypt.Error already contains that info in the error
			// string.
			wrappedMsgOnly: true,
		}
	}

	if errors.Is(err, codecutil.ErrNilValue) {
		return ErrNilValue
	}

	var marshalErr codecutil.MarshalError
	if errors.As(err, &marshalErr) {
		return MarshalError{
			Value: marshalErr.Value,
			Err:   err,

			// Set wrappedMsgOnly=true here so that the Value is not repeated
			// multiple times in the error string. We expect that the wrapped
			// codecutil.MarshalError already contains that info in the error
			// string.
			wrappedMsgOnly: true,
		}
	}

	return err
}

// IsDuplicateKeyError returns true if err is a duplicate key error. For BulkWriteExceptions,
// IsDuplicateKeyError returns true if at least one of the errors is a duplicate key error.
func IsDuplicateKeyError(err error) bool {
	if se := ServerError(nil); errors.As(err, &se) {
		return se.HasErrorCode(11000) || // Duplicate key error.
			se.HasErrorCode(11001) || // Duplicate key error on update.
			// Duplicate key error in a capped collection. See SERVER-7164.
			se.HasErrorCode(12582) ||
			// Mongos insert error caused by a duplicate key error. See
			// SERVER-11493.
			se.HasErrorCodeWithMessage(16460, " E11000 ")
	}
	return false
}

// timeoutErrs is a list of error values that indicate a timeout happened.
var timeoutErrs = [...]error{
	context.DeadlineExceeded,
	driver.ErrDeadlineWouldBeExceeded,
}

// IsTimeout returns true if err was caused by a timeout. For error chains,
// IsTimeout returns true if any error in the chain was caused by a timeout.
func IsTimeout(err error) bool {
	// Check if the error chain contains any of the timeout error values.
	for _, target := range timeoutErrs {
		if errors.Is(err, target) {
			return true
		}
	}

	// Check if the error chain contains any error types that can indicate
	// timeout.
	if errors.As(err, &topology.WaitQueueTimeoutError{}) {
		return true
	}
	if ce := (CommandError{}); errors.As(err, &ce) && ce.IsMaxTimeMSExpiredError() {
		return true
	}
	if we := (WriteException{}); errors.As(err, &we) && we.WriteConcernError != nil && we.WriteConcernError.IsMaxTimeMSExpiredError() {
		return true
	}
	if ne := net.Error(nil); errors.As(err, &ne) {
		return ne.Timeout()
	}
	// Check timeout error labels.
	if le := LabeledError(nil); errors.As(err, &le) {
		if le.HasErrorLabel("NetworkTimeoutError") || le.HasErrorLabel("ExceededTimeLimitError") {
			return true
		}
	}

	return false
}

// errorHasLabel returns true if err contains the specified label
func errorHasLabel(err error, label string) bool {
	var le LabeledError
	return errors.As(err, &le) && le.HasErrorLabel(label)
}

// IsNetworkError returns true if err is a network error
func IsNetworkError(err error) bool {
	return errorHasLabel(err, "NetworkError")
}

// MarshalError is returned when attempting to marshal a value into a document
// results in an error.
type MarshalError struct {
	Value interface{}
	Err   error

	// If wrappedMsgOnly is true, Error() only returns the error message from
	// the "Err" error.
	//
	// This is typically only set by the wrapErrors function, which uses
	// MarshalError to wrap codecutil.MarshalError, allowing users to access the
	// "Value" from the underlying error but preventing duplication in the error
	// string.
	wrappedMsgOnly bool
}

// Error implements the error interface.
func (me MarshalError) Error() string {
	// If the MarshalError was created with wrappedMsgOnly=true, only return the
	// error from the wrapped error. See the MarshalError.wrappedMsgOnly docs
	// for more info.
	if me.wrappedMsgOnly {
		return me.Err.Error()
	}

	return fmt.Sprintf("cannot marshal type %s to a BSON Document: %v", reflect.TypeOf(me.Value), me.Err)
}

func (me MarshalError) Unwrap() error { return me.Err }

// MongocryptError represents an libmongocrypt error during in-use encryption.
type MongocryptError struct {
	Code    int32
	Message string
	wrapped error

	// If wrappedMsgOnly is true, Error() only returns the error message from
	// the "wrapped" error.
	//
	// This is typically only set by the wrapErrors function, which uses
	// MarshalError to wrap mongocrypt.Error, allowing users to access the
	// "Code" and "Message" from the underlying error but preventing duplication
	// in the error string.
	wrappedMsgOnly bool
}

// Error implements the error interface.
func (m MongocryptError) Error() string {
	// If the MongocryptError was created with wrappedMsgOnly=true, only return
	// the error from the wrapped error. See the MongocryptError.wrappedMsgOnly
	// docs for more info.
	if m.wrappedMsgOnly {
		return m.wrapped.Error()
	}

	return fmt.Sprintf("mongocrypt error %d: %v", m.Code, m.Message)
}

// Unwrap returns the underlying error.
func (m MongocryptError) Unwrap() error { return m.wrapped }

// EncryptionKeyVaultError represents an error while communicating with the key vault collection during in-use
// encryption.
type EncryptionKeyVaultError struct {
	Wrapped error
}

// Error implements the error interface.
func (ekve EncryptionKeyVaultError) Error() string {
	return fmt.Sprintf("key vault communication error: %v", ekve.Wrapped)
}

// Unwrap returns the underlying error.
func (ekve EncryptionKeyVaultError) Unwrap() error {
	return ekve.Wrapped
}

// MongocryptdError represents an error while communicating with mongocryptd during in-use encryption.
type MongocryptdError struct {
	Wrapped error
}

// Error implements the error interface.
func (e MongocryptdError) Error() string {
	return fmt.Sprintf("mongocryptd communication error: %v", e.Wrapped)
}

// Unwrap returns the underlying error.
func (e MongocryptdError) Unwrap() error {
	return e.Wrapped
}

// LabeledError is an interface for errors with labels.
type LabeledError interface {
	error
	// HasErrorLabel returns true if the error contains the specified label.
	HasErrorLabel(string) bool
}

// ServerError is the interface implemented by errors returned from the server. Custom implementations of this
// interface should not be used in production.
type ServerError interface {
	LabeledError
	// HasErrorCode returns true if the error has the specified code.
	HasErrorCode(int) bool
	// HasErrorMessage returns true if the error contains the specified message.
	HasErrorMessage(string) bool
	// HasErrorCodeWithMessage returns true if any of the contained errors have the specified code and message.
	HasErrorCodeWithMessage(int, string) bool

	// ErrorCodes returns all error codes (unsorted) in the server’s response.
	// This would include nested errors (e.g., write concern errors) for
	// supporting implementations (e.g., BulkWriteException) as well as the
	// top-level error code.
	ErrorCodes() []int

	serverError()
}

func hasErrorCode(srvErr ServerError, code int) bool {
	for _, srvErrCode := range srvErr.ErrorCodes() {
		if code == srvErrCode {
			return true
		}
	}

	return false
}

var _ ServerError = CommandError{}
var _ ServerError = WriteError{}
var _ ServerError = WriteException{}
var _ ServerError = BulkWriteException{}

var _ error = ClientBulkWriteException{}

// CommandError represents a server error during execution of a command. This can be returned by any operation.
type CommandError struct {
	Code    int32
	Message string
	Labels  []string // Categories to which the error belongs
	Name    string   // A human-readable name corresponding to the error code
	Wrapped error    // The underlying error, if one exists.
	Raw     bson.Raw // The original server response containing the error.

	// If wrappedMsgOnly is true, Error() only returns the error message from
	// the "Wrapped" error.
	//
	// This is typically only set by the wrapErrors function, which uses
	// CommandError to wrap driver.Error, allowing users to access the "Code",
	// "Message", "Labels", "Name", and "Raw" from the underlying error but
	// preventing duplication in the error string.
	wrappedMsgOnly bool
}

// Error implements the error interface.
func (e CommandError) Error() string {
	// If the CommandError was created with wrappedMsgOnly=true, only return the
	// error from the wrapped error. See the CommandError.wrappedMsgOnly docs
	// for more info.
	if e.wrappedMsgOnly {
		return e.Wrapped.Error()
	}

	var msg string
	if e.Name != "" {
		msg += fmt.Sprintf("(%v)", e.Name)
	}
	if e.Message != "" {
		msg += " " + e.Message
	}
	if e.Wrapped != nil {
		msg += ": " + e.Wrapped.Error()
	}

	return msg
}

// Unwrap returns the underlying error.
func (e CommandError) Unwrap() error {
	return e.Wrapped
}

// HasErrorCode returns true if the error has the specified code.
func (e CommandError) HasErrorCode(code int) bool {
	return int(e.Code) == code
}

// ErrorCodes returns a list of error codes returned by the server.
func (e CommandError) ErrorCodes() []int {
	return []int{int(e.Code)}
}

// HasErrorLabel returns true if the error contains the specified label.
func (e CommandError) HasErrorLabel(label string) bool {
	for _, l := range e.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// HasErrorMessage returns true if the error contains the specified message.
func (e CommandError) HasErrorMessage(message string) bool {
	return strings.Contains(e.Message, message)
}

// HasErrorCodeWithMessage returns true if the error has the specified code and Message contains the specified message.
func (e CommandError) HasErrorCodeWithMessage(code int, message string) bool {
	return int(e.Code) == code && strings.Contains(e.Message, message)
}

// IsMaxTimeMSExpiredError returns true if the error is a MaxTimeMSExpired error.
func (e CommandError) IsMaxTimeMSExpiredError() bool {
	return e.Code == 50 || e.Name == "MaxTimeMSExpired"
}

// serverError implements the ServerError interface.
func (e CommandError) serverError() {}

// WriteError is an error that occurred during execution of a write operation. This error type is only returned as part
// of a WriteException or BulkWriteException.
type WriteError struct {
	// The index of the write in the slice passed to an InsertMany or BulkWrite operation that caused this error.
	Index int

	Code    int
	Message string
	Details bson.Raw

	// The original write error from the server response.
	Raw bson.Raw
}

func (we WriteError) Error() string {
	msg := we.Message
	if len(we.Details) > 0 {
		msg = fmt.Sprintf("%s: %s", msg, we.Details.String())
	}
	return msg
}

// HasErrorCode returns true if the error has the specified code.
func (we WriteError) HasErrorCode(code int) bool {
	return we.Code == code
}

// ErrorCodes returns a list of error codes returned by the server.
func (we WriteError) ErrorCodes() []int {
	return []int{we.Code}
}

// HasErrorLabel returns true if the error contains the specified label. WriteErrors do not contain labels,
// so we always return false.
func (we WriteError) HasErrorLabel(string) bool {
	return false
}

// HasErrorMessage returns true if the error contains the specified message.
func (we WriteError) HasErrorMessage(message string) bool {
	return strings.Contains(we.Message, message)
}

// HasErrorCodeWithMessage returns true if the error has the specified code and Message contains the specified message.
func (we WriteError) HasErrorCodeWithMessage(code int, message string) bool {
	return we.Code == code && strings.Contains(we.Message, message)
}

// serverError implements the ServerError interface.
func (we WriteError) serverError() {}

// WriteErrors is a group of write errors that occurred during execution of a write operation.
type WriteErrors []WriteError

// Error implements the error interface.
func (we WriteErrors) Error() string {
	errs := make([]error, len(we))
	for i := 0; i < len(we); i++ {
		errs[i] = we[i]
	}
	// WriteErrors isn't returned from batch operations, but we can still use the same formatter.
	return "write errors: " + joinBatchErrors(errs)
}

func writeErrorsFromDriverWriteErrors(errs driver.WriteErrors) WriteErrors {
	wes := make(WriteErrors, 0, len(errs))
	for _, err := range errs {
		wes = append(wes, WriteError{
			Index:   int(err.Index),
			Code:    int(err.Code),
			Message: err.Message,
			Details: bson.Raw(err.Details),
			Raw:     bson.Raw(err.Raw),
		})
	}
	return wes
}

// WriteConcernError represents a write concern failure during execution of a write operation. This error type is only
// returned as part of a WriteException or a BulkWriteException.
type WriteConcernError struct {
	Name    string
	Code    int
	Message string
	Details bson.Raw
	Raw     bson.Raw // The original write concern error from the server response.
}

// Error implements the error interface.
func (wce WriteConcernError) Error() string {
	if wce.Name != "" {
		return fmt.Sprintf("(%v) %v", wce.Name, wce.Message)
	}
	return wce.Message
}

// IsMaxTimeMSExpiredError returns true if the error is a MaxTimeMSExpired error.
func (wce WriteConcernError) IsMaxTimeMSExpiredError() bool {
	return wce.Code == 50
}

// WriteException is the error type returned by the InsertOne, DeleteOne, DeleteMany, UpdateOne, UpdateMany, and
// ReplaceOne operations.
type WriteException struct {
	// The write concern error that occurred, or nil if there was none.
	WriteConcernError *WriteConcernError

	// The write errors that occurred during operation execution.
	WriteErrors WriteErrors

	// The categories to which the exception belongs.
	Labels []string

	// The original server response containing the error.
	Raw bson.Raw
}

// Error implements the error interface.
func (mwe WriteException) Error() string {
	causes := make([]string, 0, 2)
	if mwe.WriteConcernError != nil {
		causes = append(causes, "write concern error: "+mwe.WriteConcernError.Error())
	}
	if len(mwe.WriteErrors) > 0 {
		// The WriteErrors error message already starts with "write errors:", so don't add it to the
		// error message again.
		causes = append(causes, mwe.WriteErrors.Error())
	}

	message := "write exception: "
	if len(causes) == 0 {
		return message + "no causes"
	}
	return message + strings.Join(causes, ", ")
}

// HasErrorCode returns true if the error has the specified code.
func (mwe WriteException) HasErrorCode(code int) bool {
	return hasErrorCode(mwe, code)
}

// ErrorCodes returns a list of error codes returned by the server.
func (mwe WriteException) ErrorCodes() []int {
	errorCodes := []int{}
	for _, writeError := range mwe.WriteErrors {
		errorCodes = append(errorCodes, writeError.Code)
	}

	if mwe.WriteConcernError != nil {
		errorCodes = append(errorCodes, mwe.WriteConcernError.Code)
	}

	return errorCodes
}

// HasErrorLabel returns true if the error contains the specified label.
func (mwe WriteException) HasErrorLabel(label string) bool {
	for _, l := range mwe.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// HasErrorMessage returns true if the error contains the specified message.
func (mwe WriteException) HasErrorMessage(message string) bool {
	if mwe.WriteConcernError != nil && strings.Contains(mwe.WriteConcernError.Message, message) {
		return true
	}
	for _, we := range mwe.WriteErrors {
		if strings.Contains(we.Message, message) {
			return true
		}
	}
	return false
}

// HasErrorCodeWithMessage returns true if any of the contained errors have the specified code and message.
func (mwe WriteException) HasErrorCodeWithMessage(code int, message string) bool {
	if mwe.WriteConcernError != nil &&
		mwe.WriteConcernError.Code == code && strings.Contains(mwe.WriteConcernError.Message, message) {
		return true
	}
	for _, we := range mwe.WriteErrors {
		if we.Code == code && strings.Contains(we.Message, message) {
			return true
		}
	}
	return false
}

// serverError implements the ServerError interface.
func (mwe WriteException) serverError() {}

func convertDriverWriteConcernError(wce *driver.WriteConcernError) *WriteConcernError {
	if wce == nil {
		return nil
	}

	return &WriteConcernError{
		Name:    wce.Name,
		Code:    int(wce.Code),
		Message: wce.Message,
		Details: bson.Raw(wce.Details),
		Raw:     bson.Raw(wce.Raw),
	}
}

// BulkWriteError is an error that occurred during execution of one operation in a BulkWrite. This error type is only
// returned as part of a BulkWriteException.
type BulkWriteError struct {
	WriteError            // The WriteError that occurred.
	Request    WriteModel // The WriteModel that caused this error.
}

// Error implements the error interface.
func (bwe BulkWriteError) Error() string {
	return bwe.WriteError.Error()
}

// BulkWriteException is the error type returned by BulkWrite and InsertMany operations.
type BulkWriteException struct {
	// The write concern error that occurred, or nil if there was none.
	WriteConcernError *WriteConcernError

	// The write errors that occurred during operation execution.
	WriteErrors []BulkWriteError

	// The categories to which the exception belongs.
	Labels []string
}

// Error implements the error interface.
func (bwe BulkWriteException) Error() string {
	causes := make([]string, 0, 2)
	if bwe.WriteConcernError != nil {
		causes = append(causes, "write concern error: "+bwe.WriteConcernError.Error())
	}
	if len(bwe.WriteErrors) > 0 {
		errs := make([]error, len(bwe.WriteErrors))
		for i := 0; i < len(bwe.WriteErrors); i++ {
			errs[i] = &bwe.WriteErrors[i]
		}
		causes = append(causes, "write errors: "+joinBatchErrors(errs))
	}

	message := "bulk write exception: "
	if len(causes) == 0 {
		return message + "no causes"
	}
	return "bulk write exception: " + strings.Join(causes, ", ")
}

// HasErrorCode returns true if any of the errors have the specified code.
func (bwe BulkWriteException) HasErrorCode(code int) bool {
	return hasErrorCode(bwe, code)
}

// ErrorCodes returns a list of error codes returned by the server.
func (bwe BulkWriteException) ErrorCodes() []int {
	errorCodes := []int{}
	for _, writeError := range bwe.WriteErrors {
		errorCodes = append(errorCodes, writeError.Code)
	}

	if bwe.WriteConcernError != nil {
		errorCodes = append(errorCodes, bwe.WriteConcernError.Code)
	}

	return errorCodes
}

// HasErrorLabel returns true if the error contains the specified label.
func (bwe BulkWriteException) HasErrorLabel(label string) bool {
	for _, l := range bwe.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// HasErrorMessage returns true if the error contains the specified message.
func (bwe BulkWriteException) HasErrorMessage(message string) bool {
	if bwe.WriteConcernError != nil && strings.Contains(bwe.WriteConcernError.Message, message) {
		return true
	}
	for _, we := range bwe.WriteErrors {
		if strings.Contains(we.Message, message) {
			return true
		}
	}
	return false
}

// HasErrorCodeWithMessage returns true if any of the contained errors have the specified code and message.
func (bwe BulkWriteException) HasErrorCodeWithMessage(code int, message string) bool {
	if bwe.WriteConcernError != nil &&
		bwe.WriteConcernError.Code == code && strings.Contains(bwe.WriteConcernError.Message, message) {
		return true
	}
	for _, we := range bwe.WriteErrors {
		if we.Code == code && strings.Contains(we.Message, message) {
			return true
		}
	}
	return false
}

// serverError implements the ServerError interface.
func (bwe BulkWriteException) serverError() {}

// ClientBulkWriteException is the error type returned by ClientBulkWrite operations.
type ClientBulkWriteException struct {
	// A top-level error that occurred when attempting to communicate with the server
	// or execute the bulk write. This value may not be populated if the exception was
	// thrown due to errors occurring on individual writes.
	WriteError *WriteError

	// The write concern errors that occurred.
	WriteConcernErrors []WriteConcernError

	// The write errors that occurred during individual operation execution.
	// This map will contain at most one entry if the bulk write was ordered.
	WriteErrors map[int]WriteError

	// The results of any successful operations that were performed before the error
	// was encountered.
	PartialResult *ClientBulkWriteResult
}

// Error implements the error interface.
func (bwe ClientBulkWriteException) Error() string {
	causes := make([]string, 0, 4)
	if bwe.WriteError != nil {
		causes = append(causes, "top level error: "+bwe.WriteError.Error())
	}
	if len(bwe.WriteConcernErrors) > 0 {
		errs := make([]error, len(bwe.WriteConcernErrors))
		for i := 0; i < len(bwe.WriteConcernErrors); i++ {
			errs[i] = bwe.WriteConcernErrors[i]
		}
		causes = append(causes, "write concern errors: "+joinBatchErrors(errs))
	}
	if len(bwe.WriteErrors) > 0 {
		errs := make([]error, 0, len(bwe.WriteErrors))
		for _, v := range bwe.WriteErrors {
			errs = append(errs, v)
		}
		causes = append(causes, "write errors: "+joinBatchErrors(errs))
	}
	if bwe.PartialResult != nil {
		causes = append(causes, fmt.Sprintf("result: %v", *bwe.PartialResult))
	}

	message := "bulk write exception: "
	if len(causes) == 0 {
		return message + "no causes"
	}
	return "bulk write exception: " + strings.Join(causes, ", ")
}

// returnResult is used to determine if a function calling processWriteError should return
// the result or return nil. Since the processWriteError function is used by many different
// methods, both *One and *Many, we need a way to differentiate if the method should return
// the result and the error.
type returnResult int

const (
	rrNone returnResult = 1 << iota // None means do not return the result ever.
	rrOne                           // One means return the result if this was called by a *One method.
	rrMany                          // Many means return the result is this was called by a *Many method.
	rrUnacknowledged

	rrAll               returnResult = rrOne | rrMany           // All means always return the result.
	rrAllUnacknowledged returnResult = rrAll | rrUnacknowledged // All + unacknowledged write
)

func (rr returnResult) isAcknowledged() bool {
	return rr&rrUnacknowledged == 0
}

// processWriteError handles processing the result of a write operation. If the retrunResult matches
// the calling method's type, it should return the result object in addition to the error.
// This function will wrap the errors from other packages and return them as errors from this package.
//
// WriteConcernError will be returned over WriteErrors if both are present.
func processWriteError(err error) (returnResult, error) {
	if err == nil {
		return rrAll, nil
	}
	// Do not propagate the acknowledgement sentinel error. For DDL commands,
	// (creating indexes, dropping collections, etc) acknowledgement should be
	// ignored. For non-DDL write commands (insert, update, etc), acknowledgement
	// should be be propagated at the result-level: e.g.,
	// SingleResult.Acknowledged.
	if errors.Is(err, driver.ErrUnacknowledgedWrite) {
		return rrAllUnacknowledged, nil
	}

	var wce driver.WriteCommandError
	if !errors.As(err, &wce) {
		return rrNone, wrapErrors(err)
	}

	return rrMany, WriteException{
		WriteConcernError: convertDriverWriteConcernError(wce.WriteConcernError),
		WriteErrors:       writeErrorsFromDriverWriteErrors(wce.WriteErrors),
		Labels:            wce.Labels,
		Raw:               bson.Raw(wce.Raw),
	}
}

// batchErrorsTargetLength is the target length of error messages returned by batch operation
// error types. Try to limit batch error messages to 2kb to prevent problems when printing error
// messages from large batch operations.
const batchErrorsTargetLength = 2000

// joinBatchErrors appends messages from the given errors to a comma-separated string. If the
// string exceeds 2kb, it stops appending error messages and appends the message "+N more errors..."
// to the end.
//
// Example format:
//
//	"[message 1, message 2, +8 more errors...]"
func joinBatchErrors(errs []error) string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "[")
	for idx, err := range errs {
		if idx != 0 {
			fmt.Fprint(&buf, ", ")
		}
		// If the error message has exceeded the target error message length, stop appending errors
		// to the message and append the number of remaining errors instead.
		if buf.Len() > batchErrorsTargetLength {
			fmt.Fprintf(&buf, "+%d more errors...", len(errs)-idx)
			break
		}
		fmt.Fprint(&buf, err.Error())
	}
	fmt.Fprint(&buf, "]")

	return buf.String()
}
