// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// expectedError represents an error that is expected to occur during a test. This type ignores the "isError" field in
// test files because it is always true if it is specified, so the runner can simply assert that an error occurred.
type expectedError struct {
	IsClientError  *bool          `bson:"isClientError"`
	ErrorSubstring *string        `bson:"errorContains"`
	Code           *int32         `bson:"errorCode"`
	CodeName       *string        `bson:"errorCodeName"`
	IncludedLabels []string       `bson:"errorLabelsContain"`
	OmittedLabels  []string       `bson:"errorLabelsOmit"`
	ExpectedResult *bson.RawValue `bson:"expectResult"`
}

// verifyOperationError compares the expected error to the actual operation result. If the expected parameter is nil,
// this function will only check that result.Err is also nil. Otherwise, it will check that result.Err is non-nil and
// will perform any other assertions required by the expectedError object. An error is returned if any checks fail.
func verifyOperationError(ctx context.Context, expected *expectedError, result *operationResult) error {
	// The unified spec test format doesn't treat ErrUnacknowledgedWrite as an error, so set result.Err to nil
	// to indicate that no error occurred.
	if result.Err == mongo.ErrUnacknowledgedWrite {
		result.Err = nil
	}

	if expected == nil {
		if result.Err != nil {
			return fmt.Errorf("expected no error, but got %v", result.Err)
		}
		return nil
	}

	if result.Err == nil {
		return fmt.Errorf("expected error, got nil")
	}

	// Check ErrorSubstring for both client and server-side errors.
	if expected.ErrorSubstring != nil {
		// Lowercase the error messages because Go error messages always start with lowercase letters, so they may
		// not match the casing used in specs.
		expectedErrMsg := strings.ToLower(*expected.ErrorSubstring)
		actualErrMsg := strings.ToLower(result.Err.Error())
		if !strings.Contains(actualErrMsg, expectedErrMsg) {
			return fmt.Errorf("expected error %v to contain substring %s", result.Err, *expected.ErrorSubstring)
		}
	}

	// extractErrorDetails will only succeed for server errors, so it's "ok" return value can be used to determine
	// if we got a server or client-side error.
	details, serverError := extractErrorDetails(result.Err)
	if expected.IsClientError != nil {
		// The unified test format spec considers network errors to be client-side errors.
		isClientError := !serverError || mongo.IsNetworkError(result.Err)
		if *expected.IsClientError != isClientError {
			return fmt.Errorf("expected error %v to be a client error: %v, is client error: %v", result.Err,
				*expected.IsClientError, isClientError)
		}
	}
	if !serverError {
		// Error if extractErrorDetails failed but the test requires assertions about server error details.
		if expected.Code != nil || expected.CodeName != nil || expected.IncludedLabels != nil || expected.OmittedLabels != nil {
			return fmt.Errorf("failed to extract details from error %v of type %T", result.Err, result.Err)
		}
	}

	if expected.Code != nil {
		var found bool
		for _, code := range details.codes {
			if code == *expected.Code {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected error %v to have code %d", result.Err, *expected.Code)
		}
	}
	if expected.CodeName != nil {
		var found bool
		for _, codeName := range details.codeNames {
			if codeName == *expected.CodeName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected error %v to have code name %q", result.Err, *expected.CodeName)
		}
	}
	for _, label := range expected.IncludedLabels {
		if !stringSliceContains(details.labels, label) {
			return fmt.Errorf("expected error %v to contain label %q", result.Err, label)
		}
	}
	for _, label := range expected.OmittedLabels {
		if stringSliceContains(details.labels, label) {
			return fmt.Errorf("expected error %v to not contain label %q", result.Err, label)
		}
	}

	if expected.ExpectedResult != nil {
		if err := verifyOperationResult(ctx, *expected.ExpectedResult, result); err != nil {
			return fmt.Errorf("result comparison error: %v", err)
		}
	}
	return nil
}

// errorDetails consolidates information from different server error types.
type errorDetails struct {
	codes     []int32
	codeNames []string
	labels    []string
}

// extractErrorDetails creates an errorDetails instance based on the provided error. It returns the details and an "ok"
// value which is true if the provided error is of a known type that can be processed.
func extractErrorDetails(err error) (errorDetails, bool) {
	var details errorDetails

	switch converted := err.(type) {
	case mongo.CommandError:
		details.codes = []int32{converted.Code}
		details.codeNames = []string{converted.Name}
		details.labels = converted.Labels
	case mongo.WriteException:
		if converted.WriteConcernError != nil {
			details.codes = append(details.codes, int32(converted.WriteConcernError.Code))
			details.codeNames = append(details.codeNames, converted.WriteConcernError.Name)
		}
		for _, we := range converted.WriteErrors {
			details.codes = append(details.codes, int32(we.Code))
		}
		details.labels = converted.Labels
	case mongo.BulkWriteException:
		if converted.WriteConcernError != nil {
			details.codes = append(details.codes, int32(converted.WriteConcernError.Code))
			details.codeNames = append(details.codeNames, converted.WriteConcernError.Name)
		}
		for _, we := range converted.WriteErrors {
			details.codes = append(details.codes, int32(we.Code))
		}
		details.labels = converted.Labels
	default:
		return errorDetails{}, false
	}

	return details, true
}

func stringSliceContains(arr []string, target string) bool {
	for _, val := range arr {
		if val == target {
			return true
		}
	}
	return false
}
