// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// RunCmdOptions represents arguments that can be used to configure a RunCommand
// operation.
//
// See corresponding setter methods for documentation.
type RunCmdOptions struct {
	ReadPreference *readpref.ReadPref
}

// RunCmdOptionsBuilder contains options to configure runCommand operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type RunCmdOptionsBuilder struct {
	Opts []func(*RunCmdOptions) error
}

// RunCmd creates a new RunCmdOptions instance.
func RunCmd() *RunCmdOptionsBuilder {
	return &RunCmdOptionsBuilder{}
}

// List returns a list of CountOptions setter functions.
func (rc *RunCmdOptionsBuilder) List() []func(*RunCmdOptions) error {
	return rc.Opts
}

// SetReadPreference sets value for the ReadPreference field. Specifies the read preference
// to use for the operation. The default value is nil, which means that the primary read
// preference will be used.
func (rc *RunCmdOptionsBuilder) SetReadPreference(rp *readpref.ReadPref) *RunCmdOptionsBuilder {
	rc.Opts = append(rc.Opts, func(opts *RunCmdOptions) error {
		opts.ReadPreference = rp

		return nil
	})

	return rc
}
