// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog // import "go.mongodb.org/mongo-driver/mongo/mongolog"

// Logger is an interface for loggers to implement
type Logger interface {
	Trace(message string, args ...Field)
	Debug(message string, args ...Field)
	Info(message string, args ...Field)
	Notice(message string, args ...Field)
	Warning(message string, args ...Field)
	Error(message string, args ...Field)
}
