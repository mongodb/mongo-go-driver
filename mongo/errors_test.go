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
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
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
			name:   "other error",
			err:    errors.New("foo"),
			result: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := IsTimeout(tc.err)
			assert.Equal(t, res, tc.result, "expected IsTimeout %v, got %v", tc.result, res)
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

func TestErrorCodesFrom(t *testing.T) {
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
			input: CommandError{Code: 123},
			want:  []int{123},
		},
		{
			name:  "WriteError single code",
			input: WriteError{Code: 45},
			want:  []int{45},
		},
		{
			name:  "WriteException write errors only",
			input: WriteException{WriteErrors: WriteErrors{{Code: 1}, {Code: 2}}},
			want:  []int{1, 2},
		},
		{
			name:  "WriteException with write concern error",
			input: WriteException{WriteErrors: WriteErrors{{Code: 1}}, WriteConcernError: &WriteConcernError{Code: 64}},
			want:  []int{1, 64},
		},
		{
			name: "BulkWriteException write errors only",
			input: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{WriteError: WriteError{Code: 10}},
					{WriteError: WriteError{Code: 11}},
				},
			},
			want: []int{10, 11},
		},
		{
			name: "BulkWriteException with write concern error",
			input: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{WriteError: WriteError{Code: 10}},
					{WriteError: WriteError{Code: 11}},
				},
				WriteConcernError: &WriteConcernError{Code: 79},
			},
			want: []int{10, 11, 79},
		},
		{
			name:  "driver.Error wraps to CommandError",
			input: driver.Error{Code: 91, Message: "shutdown in progress"},
			want:  []int{91},
		},
		{
			name:  "wrapped driver.Error",
			input: fmt.Errorf("context: %w", driver.Error{Code: 262, Message: "ExceededTimeLimit"}),
			want:  []int{262},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ErrorCodesFrom(tt.input))
		})
	}
}
