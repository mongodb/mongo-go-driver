// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// EstimatedDocumentCountOptions represents arguments that can be used to configure
// an EstimatedDocumentCount operation.
//
// See corresponding setter methods for documentation.
type EstimatedDocumentCountOptions struct {
	Comment interface{}
}

// EstimatedDocumentCountOptionsBuilder contains options to estimate document
// count. Each option can be set through setter functions. See documentation for
// each setter function for an explanation of the option.
type EstimatedDocumentCountOptionsBuilder struct {
	Opts []func(*EstimatedDocumentCountOptions) error
}

// EstimatedDocumentCount creates a new EstimatedDocumentCountOptions instance.
func EstimatedDocumentCount() *EstimatedDocumentCountOptionsBuilder {
	return &EstimatedDocumentCountOptionsBuilder{}
}

// List returns a list of CountOptions setter functions.
func (eco *EstimatedDocumentCountOptionsBuilder) List() []func(*EstimatedDocumentCountOptions) error {
	return eco.Opts
}

// SetComment sets the value for the Comment field. Specifies a string or document
// that will be included in server logs, profiling logs, and currentOp queries to help
// trace the operation.  The default is nil, which means that no comment will be
// included in the logs.
func (eco *EstimatedDocumentCountOptionsBuilder) SetComment(comment interface{}) *EstimatedDocumentCountOptionsBuilder {
	eco.Opts = append(eco.Opts, func(opts *EstimatedDocumentCountOptions) error {
		opts.Comment = comment

		return nil
	})

	return eco
}
