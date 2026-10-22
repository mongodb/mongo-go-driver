// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

func TestErrorMessages(t *testing.T) {
	details, err := bson.Marshal(bson.D{{"details", bson.D{{"operatorName", "$jsonSchema"}}}})
	require.Nil(t, err, "unexpected error marshaling BSON")

	cases := []struct {
		desc     string
		err      error
		expected string
	}{
		{
			desc: "WriteException error message should contain the WriteError Message and Details",
			err: WriteException{
				WriteErrors: WriteErrors{
					{
						Message: "test message 1",
						Details: details,
					},
					{
						Message: "test message 2",
						Details: details,
					},
				},
			},
			expected: `write exception: write errors: [test message 1: {"details": {"operatorName": "$jsonSchema"}}, test message 2: {"details": {"operatorName": "$jsonSchema"}}]`,
		},
		{
			desc: "BulkWriteException error message should contain the WriteError Message and Details",
			err: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Message: "test message 1",
							Details: details,
						},
					},
					{
						WriteError: WriteError{
							Message: "test message 2",
							Details: details,
						},
					},
				},
			},
			expected: `bulk write exception: write errors: [test message 1: {"details": {"operatorName": "$jsonSchema"}}, test message 2: {"details": {"operatorName": "$jsonSchema"}}]`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestServerError(t *testing.T) {
	matchWrapped := errors.New("wrapped err")
	otherWrapped := errors.New("other err")
	const matchCode = 100
	const otherCode = 120
	const label = "testError"

	testCases := []struct {
		name               string
		err                ServerError
		hasCode            bool
		hasLabel           bool
		hasMessage         bool
		hasCodeWithMessage bool
		isResult           bool
	}{
		{
			name: "CommandError all true",
			err: CommandError{
				Code:    matchCode,
				Message: "foo",
				Labels:  []string{label},
				Name:    "name",
				Wrapped: matchWrapped,
			},
			hasCode:            true,
			hasLabel:           true,
			hasMessage:         true,
			hasCodeWithMessage: true,
			isResult:           true,
		},
		{
			name: "CommandError all false",
			err: CommandError{
				Code:    otherCode,
				Message: "bar",
				Labels:  []string{"otherError"},
				Name:    "name",
				Wrapped: otherWrapped,
			},
			hasCode:            false,
			hasLabel:           false,
			hasMessage:         false,
			hasCodeWithMessage: false,
			isResult:           false,
		},
		{
			name: "CommandError has code not message",
			err: CommandError{
				Code:    matchCode,
				Message: "bar",
				Labels:  []string{},
				Name:    "name",
			},
			hasCode:            true,
			hasLabel:           false,
			hasMessage:         false,
			hasCodeWithMessage: false,
			isResult:           false,
		},
		{
			name: "WriteException all in writeConcernError",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    matchCode,
					Message: "foo",
				},
				Labels: []string{label},
			},
			hasCode:            true,
			hasLabel:           true,
			hasMessage:         true,
			hasCodeWithMessage: true,
			isResult:           false,
		},
		{
			name: "WriteException all in writeError",
			err: WriteException{
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    otherCode,
						Message: "bar",
					},
					WriteError{
						Index:   0,
						Code:    matchCode,
						Message: "foo",
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            true,
			hasLabel:           false,
			hasMessage:         true,
			hasCodeWithMessage: true,
			isResult:           false,
		},
		{
			name: "WriteException all false",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    otherCode,
					Message: "bar",
				},
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    otherCode,
						Message: "baz",
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            false,
			hasLabel:           false,
			hasMessage:         false,
			hasCodeWithMessage: false,
			isResult:           false,
		},
		{
			name: "WriteException HasErrorCodeAndMessage false",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    matchCode,
					Message: "bar",
				},
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    otherCode,
						Message: "foo",
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            true,
			hasLabel:           false,
			hasMessage:         true,
			hasCodeWithMessage: false,
			isResult:           false,
		},
		{
			name: "BulkWriteException all in writeConcernError",
			err: BulkWriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    matchCode,
					Message: "foo",
				},
				Labels: []string{label},
			},
			hasCode:            true,
			hasLabel:           true,
			hasMessage:         true,
			hasCodeWithMessage: true,
			isResult:           false,
		},
		{
			name: "BulkWriteException all in writeError",
			err: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Index:   0,
							Code:    matchCode,
							Message: "foo",
						},
						Request: &InsertOneModel{},
					},
					{
						WriteError: WriteError{
							Index:   0,
							Code:    otherCode,
							Message: "bar",
						},
						Request: &InsertOneModel{},
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            true,
			hasLabel:           false,
			hasMessage:         true,
			hasCodeWithMessage: true,
			isResult:           false,
		},
		{
			name: "BulkWriteException all false",
			err: BulkWriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    otherCode,
					Message: "bar",
				},
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Index:   0,
							Code:    otherCode,
							Message: "baz",
						},
						Request: &InsertOneModel{},
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            false,
			hasLabel:           false,
			hasMessage:         false,
			hasCodeWithMessage: false,
			isResult:           false,
		},
		{
			name: "BulkWriteException HasErrorCodeAndMessage false",
			err: BulkWriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    matchCode,
					Message: "bar",
				},
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Index:   0,
							Code:    otherCode,
							Message: "foo",
						},
						Request: &InsertOneModel{},
					},
				},
				Labels: []string{"otherError"},
			},
			hasCode:            true,
			hasLabel:           false,
			hasMessage:         true,
			hasCodeWithMessage: false,
			isResult:           false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			has := tc.err.HasErrorCode(matchCode)
			assert.Equal(t, has, tc.hasCode, "expected HasErrorCode to return %v, got %v", tc.hasCode, has)
			has = tc.err.HasErrorLabel(label)
			assert.Equal(t, has, tc.hasLabel, "expected HasErrorLabel to return %v, got %v", tc.hasLabel, has)

			// Check for full message and substring
			has = tc.err.HasErrorMessage("foo")
			assert.Equal(t, has, tc.hasMessage, "expected HasErrorMessage to return %v, got %v", tc.hasMessage, has)
			has = tc.err.HasErrorMessage("fo")
			assert.Equal(t, has, tc.hasMessage, "expected HasErrorMessage to return %v, got %v", tc.hasMessage, has)
			has = tc.err.HasErrorCodeWithMessage(matchCode, "foo")
			assert.Equal(t, has, tc.hasCodeWithMessage, "expected HasErrorCodeWithMessage to return %v, got %v", tc.hasCodeWithMessage, has)
			has = tc.err.HasErrorCodeWithMessage(matchCode, "fo")
			assert.Equal(t, has, tc.hasCodeWithMessage, "expected HasErrorCodeWithMessage to return %v, got %v", tc.hasCodeWithMessage, has)

			assert.Equal(t, errors.Is(tc.err, matchWrapped), tc.isResult, "expected errors.Is result to be %v", tc.isResult)
		})
	}
}

func TestIsDuplicateKeyError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		result bool
	}{
		{
			name: "CommandError true",
			err: CommandError{
				Code: 11000,
				Name: "blah",
			},
			result: true,
		},
		{
			name: "CommandError false",
			err: CommandError{
				Code: 100,
				Name: "blah",
			},
			result: false,
		},
		{
			name: "WriteError true",
			err: WriteError{
				Index: 0,
				Code:  11000,
			},
			result: true,
		},
		{
			name: "WriteError false",
			err: WriteError{
				Index: 0,
				Code:  100,
			},
			result: false,
		},
		{
			name: "WriteException true in writeConcernError",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    11001,
					Message: "bar",
				},
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    100,
						Message: "baz",
					},
				},
			},
			result: true,
		},
		{
			name: "WriteException true in writeErrors",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    100,
					Message: "bar",
				},
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    12582,
						Message: "baz",
					},
				},
			},
			result: true,
		},
		{
			name: "WriteException false",
			err: WriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    16460,
					Message: "bar",
				},
				WriteErrors: WriteErrors{
					WriteError{
						Index:   0,
						Code:    100,
						Message: "blah E11000 blah",
					},
				},
			},
			result: false,
		},
		{
			name: "BulkWriteException true",
			err: BulkWriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    100,
					Message: "bar",
				},
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Index:   0,
							Code:    16460,
							Message: "blah E11000 blah",
						},
						Request: &InsertOneModel{},
					},
				},
				Labels: []string{"otherError"},
			},
			result: true,
		},
		{
			name: "BulkWriteException false",
			err: BulkWriteException{
				WriteConcernError: &WriteConcernError{
					Name:    "name",
					Code:    100,
					Message: "bar",
				},
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Index:   0,
							Code:    110,
							Message: "blah",
						},
						Request: &InsertOneModel{},
					},
				},
				Labels: []string{"otherError"},
			},
			result: false,
		},
		{
			name:   "wrapped error",
			err:    fmt.Errorf("%w", CommandError{Code: 11000, Name: "blah"}),
			result: true,
		},
		{
			name:   "other error type",
			err:    errors.New("foo"),
			result: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := IsDuplicateKeyError(tc.err)
			assert.Equal(t, res, tc.result, "expected IsDuplicateKeyError %v, got %v", tc.result, res)
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	const networkLabel = "NetworkError"
	const otherLabel = "other"
	testCases := []struct {
		name   string
		err    error
		result bool
	}{
		{
			name: "ServerError true",
			err: CommandError{
				Code:   100,
				Labels: []string{networkLabel},
				Name:   "blah",
			},
			result: true,
		},
		{
			name: "ServerError false",
			err: CommandError{
				Code:   100,
				Labels: []string{otherLabel},
				Name:   "blah",
			},
			result: false,
		},
		{
			name: "wrapped error",
			err: fmt.Errorf("%w", CommandError{
				Code:   100,
				Labels: []string{networkLabel},
				Name:   "blah",
			}),
			result: true,
		},
		{
			name:   "other error type",
			err:    errors.New("foo"),
			result: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := IsNetworkError(tc.err)
			assert.Equal(t, res, tc.result, "expected IsNetworkError %v, got %v", tc.result, res)
		})
	}
}

func TestIsTimeout(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		result bool
	}{
		{
			name: "context timeout",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: context.DeadlineExceeded,
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "deadline would be exceeded",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: driver.ErrDeadlineWouldBeExceeded,
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "server selection timeout",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: context.DeadlineExceeded,
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "wait queue timeout",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: topology.WaitQueueTimeoutError{},
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "ServerError NetworkTimeoutError",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"NetworkTimeoutError"},
				Name:    "blah",
				Wrapped: nil,
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "ServerError ExceededTimeLimitError",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"ExceededTimeLimitError"},
				Name:    "blah",
				Wrapped: nil,
				Raw:     nil,
			},
			result: true,
		},
		{
			name: "ServerError false",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: nil,
				Raw:     nil,
			},
			result: false,
		},
		{
			name: "net error true",
			err: CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: netErr{true},
				Raw:     nil,
			},
			result: true,
		},
		{
			name:   "net error false",
			err:    netErr{false},
			result: false,
		},
		{
			name: "wrapped error",
			err: fmt.Errorf("%w", CommandError{
				Code:    100,
				Message: "",
				Labels:  []string{"other"},
				Name:    "blah",
				Wrapped: context.DeadlineExceeded,
				Raw:     nil,
			}),
			result: true,
		},
		{
			name:   "timeout error",
			err:    timeoutError{},
			result: true,
		},
		{
			name:   "other error",
			err:    errors.New("foo"),
			result: false,
		},
		// A WriteException keeps the error chain it was built from, so a timeout
		// from an earlier attempt of a retried write is reachable through it. See
		// GODRIVER-4172.
		{
			name:   "WriteException wrapping context.DeadlineExceeded",
			err:    writeExceptionWrapping(context.DeadlineExceeded),
			result: true,
		},
		{
			name:   "WriteException wrapping ErrDeadlineWouldBeExceeded",
			err:    writeExceptionWrapping(driver.ErrDeadlineWouldBeExceeded),
			result: true,
		},
		{
			name:   "WriteException wrapping WaitQueueTimeoutError",
			err:    writeExceptionWrapping(topology.WaitQueueTimeoutError{}),
			result: true,
		},
		{
			name:   "WriteException wrapping a timeout net error",
			err:    writeExceptionWrapping(netErr{true}),
			result: true,
		},
		{
			name:   "WriteException without a timeout in the chain",
			err:    writeExceptionWrapping(errors.New("foo")),
			result: false,
		},
		{
			// A non-timeout net.Error in the chain must not suppress the error
			// label check.
			name: "ExceededTimeLimitError label with a non-timeout net error",
			err: writeExceptionWrapping(netErr{false},
				driver.WriteCommandError{
					WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key"}},
					Labels:      []string{"ExceededTimeLimitError"},
				}),
			result: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := IsTimeout(tc.err)
			assert.Equal(t, res, tc.result, "expected IsTimeout %v, got %v", tc.result, res)
		})
	}
}

func TestTimeoutError(t *testing.T) {
	tests := []struct {
		name   string
		err    timeoutError
		errMsg string
		labels []string
	}{
		{
			name: "TimeoutError without wrapped error",
			err: timeoutError{
				Wrapped: nil,
			},
			errMsg: "operation timed out",
			labels: []string{},
		},
		{
			name: "TimeoutError with wrapped LabeledError",
			err: timeoutError{
				Wrapped: CommandError{
					Code:    100,
					Message: "",
					Labels:  []string{"other"},
					Name:    "blah",
					Wrapped: context.DeadlineExceeded,
					Raw:     nil,
				},
			},
			errMsg: "operation timed out: (blah): context deadline exceeded",
			labels: []string{"other"},
		},
		{
			name: "TimeoutError with wrapped non-LabeledError",
			err: timeoutError{
				Wrapped: context.DeadlineExceeded,
			},
			errMsg: "operation timed out: context deadline exceeded",
			labels: []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, IsTimeout(tc.err), "expected a timeout error")
			assert.Equal(t, tc.err.Error(), tc.errMsg, "expected error message %q, got %q", tc.errMsg, tc.err.Error())
			var lerr LabeledError
			require.True(t, errors.As(tc.err, &lerr), "expected a LabeledError")
			for _, label := range tc.labels {
				assert.True(t, lerr.HasErrorLabel(label), "expected label %q", label)
			}
		})
	}
}

func TestServerError_ErrorCodes(t *testing.T) {
	tests := []struct {
		name  string
		error ServerError
		want  []int
	}{
		{
			name:  "CommandError",
			error: CommandError{Code: 1},
			want:  []int{1},
		},
		{
			name:  "WriteError",
			error: WriteError{Code: 1},
			want:  []int{1},
		},
		{
			name:  "WriteException single",
			error: WriteException{WriteErrors: []WriteError{{Code: 1}}},
			want:  []int{1},
		},
		{
			name:  "WriteException multiple",
			error: WriteException{WriteErrors: []WriteError{{Code: 1}, {Code: 2}}},
			want:  []int{1, 2},
		},
		{
			name:  "WriteException duplicates",
			error: WriteException{WriteErrors: []WriteError{{Code: 1}, {Code: 2}, {Code: 2}}},
			want:  []int{1, 2, 2},
		},
		{
			name:  "BulkWriteException single",
			error: BulkWriteException{WriteErrors: []BulkWriteError{{WriteError: WriteError{Code: 1}}}},
			want:  []int{1},
		},
		{
			name: "BulkWriteException multiple",
			error: BulkWriteException{WriteErrors: []BulkWriteError{
				{WriteError: WriteError{Code: 1}},
				{WriteError: WriteError{Code: 2}},
			}},
			want: []int{1, 2},
		},
		{
			name: "BulkWriteException duplicates",
			error: BulkWriteException{WriteErrors: []BulkWriteError{
				{WriteError: WriteError{Code: 1}},
				{WriteError: WriteError{Code: 2}},
				{WriteError: WriteError{Code: 2}},
			}},
			want: []int{1, 2, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.error.ErrorCodes()

			assert.ElementsMatch(t, got, test.want)
		})
	}
}

type netErr struct {
	timeout bool
}

func (n netErr) Error() string {
	return "error"
}

func (n netErr) Timeout() bool {
	return n.timeout
}

func (n netErr) Temporary() bool {
	return false
}

var _ net.Error = (*netErr)(nil)

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  []int
	}{
		{
			name:  "nil error",
			input: nil,
			want:  nil,
		},
		{
			name:  "non-server error",
			input: errors.New("boom"),
			want:  []int{},
		},
		{
			name:  "CommandError single code",
			input: CommandError{Code: 1},
			want:  []int{1},
		},
		{
			name:  "WriteError single code",
			input: WriteError{Code: 1},
			want:  []int{1},
		},
		{
			name:  "WriteException write errors only",
			input: WriteException{WriteErrors: WriteErrors{{Code: 1}, {Code: 2}}},
			want:  []int{1, 2},
		},
		{
			name:  "WriteException with write concern error",
			input: WriteException{WriteErrors: WriteErrors{{Code: 1}}, WriteConcernError: &WriteConcernError{Code: 2}},
			want:  []int{1, 2},
		},
		{
			name: "BulkWriteException write errors only",
			input: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{WriteError: WriteError{Code: 1}},
					{WriteError: WriteError{Code: 2}},
				},
			},
			want: []int{1, 2},
		},
		{
			name: "BulkWriteException with write concern error",
			input: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{WriteError: WriteError{Code: 1}},
					{WriteError: WriteError{Code: 2}},
				},
				WriteConcernError: &WriteConcernError{Code: 3},
			},
			want: []int{1, 2, 3},
		},
		{
			name:  "driver.Error wraps to CommandError",
			input: driver.Error{Code: 1, Message: "shutdown in progress"},
			want:  []int{1},
		},
		{
			name:  "wrapped driver.Error",
			input: fmt.Errorf("context: %w", driver.Error{Code: 1, Message: "ExceededTimeLimit"}),
			want:  []int{1},
		},
		{
			input: wrapErrors(driver.Error{Code: 1, Message: "Custom error"}),
			name:  "double wrapped driver.Error",
			want:  []int{1},
		},
		{
			name:  "already wrapped CommandError",
			input: CommandError{Code: 1},
			want:  []int{1},
		},
		{
			name:  "CommandError wrapped in fmt.Errorf",
			input: fmt.Errorf("operation failed: %w", CommandError{Code: 1}),
			want:  []int{1},
		},
		{
			name: "WriteException wrapped in fmt.Errorf",
			input: fmt.Errorf("batch failed: %w", WriteException{
				WriteErrors: WriteErrors{{Code: 1}, {Code: 2}},
			}),
			want: []int{1, 2},
		},
		{
			name: "BulkWriteException with all error types",
			input: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{WriteError: WriteError{Code: 1}},
					{WriteError: WriteError{Code: 2}},
					{WriteError: WriteError{Code: 1}},
				},
				WriteConcernError: &WriteConcernError{Code: 2},
			},
			want: []int{1, 2, 1, 2},
		},
		{
			name:  "driver.Error with multiple fields",
			input: driver.Error{Code: 1, Message: "test", Name: "TestError", Labels: []string{"label1"}},
			want:  []int{1},
		},
		{
			name:  "topology.ErrTopologyClosed converts to ErrClientDisconnected",
			input: topology.ErrTopologyClosed,
			want:  []int{},
		},
		{
			name:  "ClientBulkWriteException single write error",
			input: ClientBulkWriteException{WriteErrors: map[int]WriteError{0: {Code: 7}}},
			want:  []int{7},
		},
		{
			name:  "ClientBulkWriteException single write concern error",
			input: ClientBulkWriteException{WriteConcernErrors: []WriteConcernError{{Code: 8}}},
			want:  []int{8},
		},
		{
			name:  "ClientBulkWriteException top-level write error only",
			input: ClientBulkWriteException{WriteError: &WriteError{Code: 9}},
			want:  []int{9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ErrorCodes(tt.input))
		})
	}
}

func TestClientBulkWriteException_ErrorCodes(t *testing.T) {
	tests := []struct {
		name  string
		input ClientBulkWriteException
		want  []int
	}{
		{
			name:  "empty",
			input: ClientBulkWriteException{},
			want:  []int{},
		},
		{
			name:  "top-level write error",
			input: ClientBulkWriteException{WriteError: &WriteError{Code: 1}},
			want:  []int{1},
		},
		{
			name:  "write concern errors",
			input: ClientBulkWriteException{WriteConcernErrors: []WriteConcernError{{Code: 2}, {Code: 3}}},
			want:  []int{2, 3},
		},
		{
			name:  "write errors (map)",
			input: ClientBulkWriteException{WriteErrors: map[int]WriteError{0: {Code: 4}, 1: {Code: 5}}},
			want:  []int{4, 5},
		},
		{
			name: "all sources combined",
			input: ClientBulkWriteException{
				WriteError:         &WriteError{Code: 10},
				WriteConcernErrors: []WriteConcernError{{Code: 11}},
				WriteErrors:        map[int]WriteError{0: {Code: 12}},
			},
			want: []int{10, 11, 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ElementsMatch(
				t,
				tt.want,
				tt.input.ErrorCodes(),
				"%T.ErrorCodes method",
				tt.input,
			)

			require.ElementsMatch(
				t,
				tt.want,
				ErrorCodes(tt.input),
				"mongo.ErrorCodes(%T)",
				tt.input,
			)
		})
	}
}

func TestErrorCodesNoDoubleWrapping(t *testing.T) {
	driverErr := driver.Error{Code: 1, Message: "test error"}

	// Wrap it once
	wrapped := wrapErrors(driverErr)
	cmdErr, ok := wrapped.(CommandError)
	require.True(t, ok, "wrapErrors should return CommandError")
	require.Equal(t, int32(1), cmdErr.Code)

	// Call ErrorCodes on the wrapped error
	codes := ErrorCodes(wrapped)
	require.Equal(t, []int{1}, codes)

	// The wrapped error's structure should not have changed
	cmdErrAfter, ok := wrapped.(CommandError)
	require.True(t, ok, "error should still be CommandError")
	require.Equal(t, cmdErr.Code, cmdErrAfter.Code)

	// Verify that calling ErrorCodes again gives same result
	codes2 := ErrorCodes(wrapped)
	require.Equal(t, codes, codes2)
}

// TestProcessWriteErrorChain asserts that processWriteError wraps the error
// chain it is passed instead of replacing it, so that errors wrapped or joined
// beneath the driver.WriteCommandError (e.g. the errors from the previous
// attempts of a retried operation) stay reachable with errors.Is and
// errors.As. See GODRIVER-4172.
func TestProcessWriteErrorChain(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	wce := driver.WriteCommandError{
		WriteErrors: driver.WriteErrors{{Index: 0, Code: 11000, Message: "duplicate key"}},
		Labels:      []string{"RetryableWriteError"},
	}

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "wrapped",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("%w: %w", sentinel, wce)),
		},
		{
			name: "joined",
			err:  errors.Join(sentinel, wce),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rr, got := processWriteError(test.err)
			require.Equal(t, rrMany, rr, "expected a *Many return result")

			var we WriteException
			require.True(t, errors.As(got, &we), "expected a WriteException, got %T", got)
			assert.Equal(t, 11000, we.WriteErrors[0].Code, "expected the write error code")
			assert.Equal(t, []string{"RetryableWriteError"}, we.Labels, "expected the labels")

			// The entire error chain must stay reachable.
			assert.ErrorIs(t, got, sentinel, "expected the sentinel error in the chain")
			assert.True(t, errors.As(got, &driver.WriteCommandError{}),
				"expected the driver.WriteCommandError in the chain")
			assert.Equal(t, test.err, errors.Unwrap(got),
				"expected Unwrap to return the original error chain")
		})
	}
}

// writeExceptionWrapping returns the WriteException that processWriteError
// builds from a driver.WriteCommandError joined with the given error. It is
// used to build the error chains that a retried write operation returns. The
// driver.WriteCommandError defaults to a duplicate key error.
func writeExceptionWrapping(err error, wce ...driver.WriteCommandError) error {
	cmdErr := driver.WriteCommandError{
		WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key"}},
	}
	if len(wce) > 0 {
		cmdErr = wce[0]
	}

	_, writeErr := processWriteError(errors.Join(err, cmdErr))

	return writeErr
}

// TestProcessWriteError asserts the full returnResult and error contract of
// processWriteError.
func TestProcessWriteError(t *testing.T) {
	t.Parallel()

	wce := driver.WriteCommandError{
		WriteConcernError: &driver.WriteConcernError{Code: 100, Name: "wce", Message: "write concern error"},
		WriteErrors:       driver.WriteErrors{{Index: 1, Code: 11000, Message: "duplicate key"}},
		Labels:            []string{"RetryableWriteError"},
		Raw:               bsoncore.Document{5, 0, 0, 0, 0},
	}

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		rr, err := processWriteError(nil)
		assert.Equal(t, rrAll, rr, "expected rrAll")
		assert.NoError(t, err, "expected no error")
	})

	t.Run("unacknowledged write", func(t *testing.T) {
		t.Parallel()

		for _, err := range []error{
			driver.ErrUnacknowledgedWrite,
			fmt.Errorf("outer: %w", driver.ErrUnacknowledgedWrite),
		} {
			rr, got := processWriteError(err)
			assert.Equal(t, rrAllUnacknowledged, rr, "expected rrAllUnacknowledged")
			assert.NoError(t, got, "expected the acknowledgement sentinel error to not be propagated")
		}
	})

	t.Run("command error", func(t *testing.T) {
		t.Parallel()

		rr, got := processWriteError(driver.Error{Code: 1, Message: "command error"})
		assert.Equal(t, rrNone, rr, "expected rrNone")

		var cmdErr CommandError
		require.True(t, errors.As(got, &cmdErr), "expected a CommandError, got %T", got)
		assert.Equal(t, int32(1), cmdErr.Code, "expected the driver error code")
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()

		err := errors.New("foo")
		rr, got := processWriteError(err)
		assert.Equal(t, rrNone, rr, "expected rrNone")
		assert.Equal(t, err, got, "expected the error to be returned unchanged")
	})

	t.Run("write command error", func(t *testing.T) {
		t.Parallel()

		rr, got := processWriteError(wce)
		assert.Equal(t, rrMany, rr, "expected rrMany")

		var we WriteException
		require.True(t, errors.As(got, &we), "expected a WriteException, got %T", got)
		require.NotNil(t, we.WriteConcernError, "expected a write concern error")
		assert.Equal(t, 100, we.WriteConcernError.Code, "expected the write concern error code")
		assert.Equal(t, "wce", we.WriteConcernError.Name, "expected the write concern error name")
		require.Len(t, we.WriteErrors, 1, "expected 1 write error")
		assert.Equal(t, 1, we.WriteErrors[0].Index, "expected the write error index")
		assert.Equal(t, 11000, we.WriteErrors[0].Code, "expected the write error code")
		assert.Equal(t, []string{"RetryableWriteError"}, we.Labels, "expected the labels")
		assert.Equal(t, bson.Raw(wce.Raw), we.Raw, "expected the raw response")
	})
}

// TestProcessWriteErrorNoCommandErrorInChain asserts that processWriteError
// wraps the error it is passed rather than wrapErrors(err). Converting the
// chain would turn a driver.Error from an earlier attempt into a CommandError,
// which would make errors.As report an error from an attempt other than the one
// the operation actually failed with.
func TestProcessWriteErrorNoCommandErrorInChain(t *testing.T) {
	t.Parallel()

	driverErr := driver.Error{Code: 10107, Name: "NotWritablePrimary", Message: "not primary"}
	wce := driver.WriteCommandError{
		WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key"}},
	}

	_, got := processWriteError(errors.Join(driverErr, wce))

	var cmdErr CommandError
	assert.False(t, errors.As(got, &cmdErr), "expected no mongo.CommandError in the chain")

	var de driver.Error
	require.True(t, errors.As(got, &de), "expected the driver.Error to stay reachable")
	assert.Equal(t, int32(10107), de.Code, "expected the driver error code")
}

// TestWriteExceptionRootMatchesFirst asserts that the helpers that walk the
// error chain still resolve against the WriteException at the root of the
// chain, not against an error from an earlier attempt.
func TestWriteExceptionRootMatchesFirst(t *testing.T) {
	t.Parallel()

	err := writeExceptionWrapping(driver.Error{Code: 1, Name: "other", Message: "other"})

	assert.Equal(t, []int{11000}, ErrorCodes(err), "expected only the write error code")
	assert.True(t, IsDuplicateKeyError(err), "expected a duplicate key error")
	assert.False(t, IsNetworkError(err), "expected no network error")
}

// TestWriteExceptionUnwrapNil asserts that a WriteException that was not
// returned by a write operation has no underlying error and that inspecting it
// does not panic.
func TestWriteExceptionUnwrapNil(t *testing.T) {
	t.Parallel()

	we := WriteException{WriteErrors: WriteErrors{{Code: 11000, Message: "duplicate key"}}}

	assert.NoError(t, errors.Unwrap(we), "expected no underlying error")
	assert.False(t, errors.Is(we, context.DeadlineExceeded), "expected no match in the chain")
	assert.False(t, errors.As(error(we), &driver.WriteCommandError{}), "expected no match in the chain")
	assert.Equal(t, "write exception: write errors: [duplicate key]", we.Error(),
		"expected the write exception error message")
}

// TestWriteExceptionErrorMessage asserts that the error message of a
// WriteException returned by a write operation defers to the wrapped error
// instead of duplicating it, and that it still reports everything the
// WriteException used to report on its own.
func TestWriteExceptionErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("no duplication", func(t *testing.T) {
		t.Parallel()

		wce := driver.WriteCommandError{
			WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key"}},
		}
		in := fmt.Errorf("attempt 1: %w", wce)

		_, got := processWriteError(in)

		assert.Equal(t, in.Error(), got.Error(), "expected the wrapped error message")
		assert.Equal(t, 1, strings.Count(got.Error(), "duplicate key"),
			"expected the write error message to appear exactly once")
	})

	t.Run("includes write error details", func(t *testing.T) {
		t.Parallel()

		details, err := bson.Marshal(bson.D{{"details", bson.D{{"operatorName", "$jsonSchema"}}}})
		require.NoError(t, err, "unexpected error marshaling BSON")

		wce := driver.WriteCommandError{
			WriteErrors: driver.WriteErrors{{
				Code:    121,
				Message: "Document failed validation",
				Details: bsoncore.Document(details),
			}},
		}

		_, got := processWriteError(wce)

		assert.Equal(t,
			`write command error: write errors: [Document failed validation: {"details": {"operatorName": "$jsonSchema"}}]`,
			got.Error(),
			"expected the write error details in the error message")
	})

	t.Run("truncates large batches", func(t *testing.T) {
		t.Parallel()

		writeErrors := make(driver.WriteErrors, 0, 500)
		for i := 0; i < 500; i++ {
			writeErrors = append(writeErrors, driver.WriteError{
				Index:   int64(i),
				Code:    11000,
				Message: "duplicate key error collection: db.coll index: _id_",
			})
		}

		_, got := processWriteError(driver.WriteCommandError{WriteErrors: writeErrors})

		assert.True(t, len(got.Error()) < 3000, "expected a truncated error message, got %d bytes", len(got.Error()))
		assert.Contains(t, got.Error(), "more errors...", "expected the truncation message")
	})
}
