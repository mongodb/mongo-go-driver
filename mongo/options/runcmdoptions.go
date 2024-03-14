// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// RunCmdArgs represents arguments that can be used to configure a RunCommand
// operation.
type RunCmdArgs struct {
	// The read preference to use for the operation. The default value is nil, which means that the primary read
	// preference will be used.
	ReadPreference *readpref.ReadPref
}

// RunCmdOptions contains options to configure runCommand operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type RunCmdOptions struct {
	Opts []func(*RunCmdArgs) error
}

// RunCmd creates a new RunCmdOptions instance.
func RunCmd() *RunCmdOptions {
	return &RunCmdOptions{}
}

// ArgsSetters returns a list of CountArgs setter functions.
func (rc *RunCmdOptions) ArgsSetters() []func(*RunCmdArgs) error {
	return rc.Opts
}

// SetReadPreference sets value for the ReadPreference field.
func (rc *RunCmdOptions) SetReadPreference(rp *readpref.ReadPref) *RunCmdOptions {
	rc.Opts = append(rc.Opts, func(args *RunCmdArgs) error {
		args.ReadPreference = rp

		return nil
	})

	return rc
}
