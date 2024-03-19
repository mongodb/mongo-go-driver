// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DropCollectionArgs represents arguments that can be used to configure a Drop
// operation.
type DropCollectionArgs struct {
	// EncryptedFields configures encrypted fields for encrypted collections.
	//
	// This option is only valid for MongoDB versions >= 6.0
	EncryptedFields interface{}
}

// DropCollectionOptions contains options to configure collection drop
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type DropCollectionOptions struct {
	Opts []func(*DropCollectionArgs) error
}

// DropCollection creates a new DropCollectionOptions instance.
func DropCollection() *DropCollectionOptions {
	return &DropCollectionOptions{}
}

// ArgsSetters returns a list of DropCollectionArgs setter functions.
func (d *DropCollectionOptions) ArgsSetters() []func(*DropCollectionArgs) error {
	return d.Opts
}

// SetEncryptedFields sets the encrypted fields for encrypted collections.
func (d *DropCollectionOptions) SetEncryptedFields(encryptedFields interface{}) *DropCollectionOptions {
	d.Opts = append(d.Opts, func(args *DropCollectionArgs) error {
		args.EncryptedFields = encryptedFields

		return nil
	})

	return d
}
