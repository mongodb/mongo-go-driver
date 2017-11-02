// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

// NewCommand creates a new RequestMessage to be set as a command.
func NewCommand(requestID int32, dbName string, slaveOK bool, cmd interface{}) Request {
	flags := QueryFlags(0)
	if slaveOK {
		flags |= SlaveOK
	}

	return &Query{
		ReqID:              requestID,
		Flags:              flags,
		FullCollectionName: dbName + ".$cmd",
		NumberToReturn:     -1,
		Query:              cmd,
	}
}
