// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

type EstimatedDocumentCountOptions struct {
	MaxTimeMS *int32
}

func EstimatedCount() *EstimatedDocumentCountOptions {
	return &EstimatedDocumentCountOptions{}
}

func (eco *EstimatedDocumentCountOptions) SetMaxTimeMS(i int32) *EstimatedDocumentCountOptions {
	eco.MaxTimeMS = &i
	return eco
}
