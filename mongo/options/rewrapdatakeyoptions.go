// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// RewrapManyDataKeyArgs represents all possible options used to decrypt and
// encrypt all matching data keys with a possibly new masterKey.
type RewrapManyDataKeyArgs struct {
	// Provider identifies the new KMS provider. If omitted, encrypting uses the current KMS provider.
	Provider *string

	// MasterKey identifies the new masterKey. If omitted, rewraps with the current masterKey.
	MasterKey interface{}
}

// RewrapManyDataKeyOptions contains options to configure rewraping a data key.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type RewrapManyDataKeyOptions struct {
	Opts []func(*RewrapManyDataKeyArgs) error
}

// RewrapManyDataKey creates a new RewrapManyDataKeyOptions instance.
func RewrapManyDataKey() *RewrapManyDataKeyOptions {
	return new(RewrapManyDataKeyOptions)
}

// ArgsSetters returns a list of CountArgs setter functions.
func (rmdko *RewrapManyDataKeyOptions) ArgsSetters() []func(*RewrapManyDataKeyArgs) error {
	return rmdko.Opts
}

// SetProvider sets the value for the Provider field.
func (rmdko *RewrapManyDataKeyOptions) SetProvider(provider string) *RewrapManyDataKeyOptions {
	rmdko.Opts = append(rmdko.Opts, func(args *RewrapManyDataKeyArgs) error {
		args.Provider = &provider

		return nil
	})

	return rmdko
}

// SetMasterKey sets the value for the MasterKey field.
func (rmdko *RewrapManyDataKeyOptions) SetMasterKey(masterKey interface{}) *RewrapManyDataKeyOptions {
	rmdko.Opts = append(rmdko.Opts, func(args *RewrapManyDataKeyArgs) error {
		args.MasterKey = masterKey

		return nil
	})

	return rmdko
}
