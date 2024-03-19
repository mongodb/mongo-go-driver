// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// EstimatedDocumentCountOptions represents options that can be used to configure an EstimatedDocumentCount operation.
type EstimatedDocumentCountOptions struct {
	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default is nil, which means that no comment will be included in the logs.
	Comment interface{}
}

// EstimatedDocumentCount creates a new EstimatedDocumentCountOptions instance.
func EstimatedDocumentCount() *EstimatedDocumentCountOptions {
	return &EstimatedDocumentCountOptions{}
}

// SetComment sets the value for the Comment field.
func (eco *EstimatedDocumentCountOptions) SetComment(comment interface{}) *EstimatedDocumentCountOptions {
	eco.Comment = comment
	return eco
}
