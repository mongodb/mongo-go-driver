// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DropCollectionOptions represents arguments that can be used to configure a
// Drop operation.
//
// See corresponding setter methods for documentation.
type DropCollectionOptions struct {
	EncryptedFields any
}

// DropCollectionOptionsBuilder contains options to configure collection drop
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type DropCollectionOptionsBuilder struct {
	Opts []func(*DropCollectionOptions) error
}

// DropCollection creates a new DropCollectionOptions instance.
func DropCollection() *DropCollectionOptionsBuilder {
	return &DropCollectionOptionsBuilder{}
}

// List returns a list of DropCollectionOptions setter functions.
func (d *DropCollectionOptionsBuilder) List() []func(*DropCollectionOptions) error {
	return d.Opts
}

// SetEncryptedFields sets the encrypted fields for encrypted collections.
//
// This option is only valid for MongoDB versions >= 6.0
func (d *DropCollectionOptionsBuilder) SetEncryptedFields(encryptedFields any) *DropCollectionOptionsBuilder {
	d.Opts = append(d.Opts, func(opts *DropCollectionOptions) error {
		opts.EncryptedFields = encryptedFields

		return nil
	})

	return d
}
