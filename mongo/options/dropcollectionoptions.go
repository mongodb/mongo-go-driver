// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DropCollectionOptions represents options that can be used to configure a Drop operation.
type DropCollectionOptions struct {
	// EncryptedFields configures encrypted fields for FLE 2.0.
	//
	// This option is only valid for MongoDB versions >= 6.0
	EncryptedFields interface{}
}

// DropCollection creates a new DropCollectionOptions instance.
func DropCollection() *DropCollectionOptions {
	return &DropCollectionOptions{}
}

// SetEncryptedFields sets the encrypted fields for FLE 2.0 collections.
func (c *DropCollectionOptions) SetEncryptedFields(encryptedFields interface{}) *DropCollectionOptions {
	c.EncryptedFields = encryptedFields
	return c
}

// MergeDropCollectionOptions combines the given DropCollectionOptions instances into a single
// DropCollectionOptions in a last-one-wins fashion.
func MergeDropCollectionOptions(opts ...*DropCollectionOptions) *DropCollectionOptions {
	cc := DropCollection()

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.EncryptedFields != nil {
			cc.EncryptedFields = opt.EncryptedFields
		}
	}

	return cc
}
