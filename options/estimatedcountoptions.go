// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// EstimatedDocumentCountOptions represents all possible options to the estimatedDocumentCount() function
type EstimatedDocumentCountOptions struct {
	MaxTimeMS *int64 // The maximum amount of time to allow the operation to run
}

// EstimatedDocumentCount returns a pointer to a new EstimatedDocumentCountOptions
func EstimatedDocumentCount() *EstimatedDocumentCountOptions {
	return &EstimatedDocumentCountOptions{}
}

// SetMaxTimeMS specifies the maximum amount of time to allow the operation to run
func (eco *EstimatedDocumentCountOptions) SetMaxTimeMS(i int64) *EstimatedDocumentCountOptions {
	eco.MaxTimeMS = &i
	return eco
}
