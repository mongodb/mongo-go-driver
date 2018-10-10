// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DistinctOptions represents all possible options to the distinct() function
type DistinctOptions struct {
	Collation *Collation // Specifies a collation
	MaxTimeMS *int64     // The maximum amount of time to allow the operation to run
}

// Distinct returns a pointer to a new DistinctOptions
func Distinct() *DistinctOptions {
	return &DistinctOptions{}
}

// SetCollation specifies a collation
func (do *DistinctOptions) SetCollation(c Collation) *DistinctOptions {
	do.Collation = &c
	return do
}

// SetMaxTimeMS specifies the maximum amount of time to allow the operation to run
func (do *DistinctOptions) SetMaxTimeMS(i int64) *DistinctOptions {
	do.MaxTimeMS = &i
	return do
}

// ToDistinctOptions combines the argued DistinctOptions into a single DistinctOptions in a last-one-wins fashion
func ToDistinctOptions(opts ...*DistinctOptions) *DistinctOptions {
	distinctOpts := Distinct()
	for _, do := range opts {
		if do.Collation != nil {
			distinctOpts.Collation = do.Collation
		}
		if do.MaxTimeMS != nil {
			distinctOpts.MaxTimeMS = do.MaxTimeMS
		}
	}

	return distinctOpts
}
