// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "time"

// EstimatedDocumentCountArgs represents arguments that can be used to configure
// an EstimatedDocumentCount operation.
type EstimatedDocumentCountArgs struct {
	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime
	// is ignored if Timeout is set on the client.
	MaxTime *time.Duration
}

// EstimatedDocumentCountOptions contains options to estimate document count.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type EstimatedDocumentCountOptions struct {
	Opts []func(*EstimatedDocumentCountArgs) error
}

// EstimatedDocumentCount creates a new EstimatedDocumentCountOptions instance.
func EstimatedDocumentCount() *EstimatedDocumentCountOptions {
	return &EstimatedDocumentCountOptions{}
}

// ArgsSetters returns a list of CountArgs setter functions.
func (eco *EstimatedDocumentCountOptions) ArgsSetters() []func(*EstimatedDocumentCountArgs) error {
	return eco.Opts
}

// SetComment sets the value for the Comment field.
func (eco *EstimatedDocumentCountOptions) SetComment(comment interface{}) *EstimatedDocumentCountOptions {
	eco.Opts = append(eco.Opts, func(args *EstimatedDocumentCountArgs) error {
		args.Comment = comment

		return nil
	})

	return eco
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option
// may be used in its place to control the amount of time that a single operation can run before
// returning an error. MaxTime is ignored if Timeout is set on the client.
func (eco *EstimatedDocumentCountOptions) SetMaxTime(d time.Duration) *EstimatedDocumentCountOptions {
	eco.Opts = append(eco.Opts, func(args *EstimatedDocumentCountArgs) error {
		args.MaxTime = &d

		return nil
	})

	return eco
}
