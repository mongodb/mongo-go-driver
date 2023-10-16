// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// RunCmdOptions represents options that can be used to configure a RunCommand operation.
type RunCmdOptions struct {
	// The read preference to use for the operation. The default value is nil, which means that the primary read
	// preference will be used.
	ReadPreference *readpref.ReadPref
}

// RunCmd creates a new RunCmdOptions instance.
func RunCmd() *RunCmdOptions {
	return &RunCmdOptions{}
}

// SetReadPreference sets value for the ReadPreference field.
func (rc *RunCmdOptions) SetReadPreference(rp *readpref.ReadPref) *RunCmdOptions {
	rc.ReadPreference = rp
	return rc
}
