// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package failpoint

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	// ModeAlwaysOn is the fail point mode that enables the fail point for an
	// indefinite number of matching commands.
	ModeAlwaysOn = "alwaysOn"

	// ModeOff is the fail point mode that disables the fail point.
	ModeOff = "off"
)

// FailPoint is used to configure a server fail point. It is intended to be
// passed as the command argument to RunCommand.
//
// For more information about fail points, see
// https://github.com/mongodb/specifications/tree/HEAD/source/transactions/tests#server-fail-point
type FailPoint struct {
	ConfigureFailPoint string `bson:"configureFailPoint"`
	// Mode should be a string, FailPointMode, or map[string]interface{}
	Mode interface{} `bson:"mode"`
	Data Data        `bson:"data"`
}

// Mode configures when a fail point will be enabled. It is used to set the
// FailPoint.Mode field.
type Mode struct {
	Times int32 `bson:"times"`
	Skip  int32 `bson:"skip"`
}

// Data configures how a fail point will behave. It is used to set the
// FailPoint.Data field.
type Data struct {
	FailCommands                  []string           `bson:"failCommands,omitempty"`
	CloseConnection               bool               `bson:"closeConnection,omitempty"`
	ErrorCode                     int32              `bson:"errorCode,omitempty"`
	FailBeforeCommitExceptionCode int32              `bson:"failBeforeCommitExceptionCode,omitempty"`
	ErrorLabels                   *[]string          `bson:"errorLabels,omitempty"`
	WriteConcernError             *WriteConcernError `bson:"writeConcernError,omitempty"`
	BlockConnection               bool               `bson:"blockConnection,omitempty"`
	BlockTimeMS                   int32              `bson:"blockTimeMS,omitempty"`
	AppName                       string             `bson:"appName,omitempty"`
}

// WriteConcernError is the write concern error to return when the fail point is
// triggered. It is used to set the FailPoint.Data.WriteConcernError field.
type WriteConcernError struct {
	Code        int32     `bson:"code"`
	Name        string    `bson:"codeName"`
	Errmsg      string    `bson:"errmsg"`
	ErrorLabels *[]string `bson:"errorLabels,omitempty"`
	ErrInfo     bson.Raw  `bson:"errInfo,omitempty"`
}
