// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// RewrapManyDataKeyOptions represents all possible options used to decrypt and
// encrypt all matching data keys with a possibly new masterKey.
//
// See corresponding setter methods for documentation.
type RewrapManyDataKeyOptions struct {
	Provider  *string
	MasterKey interface{}
}

// RewrapManyDataKeyOptionsBuilder contains options to configure rewraping a
// data key. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type RewrapManyDataKeyOptionsBuilder struct {
	Opts []func(*RewrapManyDataKeyOptions) error
}

// RewrapManyDataKey creates a new RewrapManyDataKeyOptions instance.
func RewrapManyDataKey() *RewrapManyDataKeyOptionsBuilder {
	return new(RewrapManyDataKeyOptionsBuilder)
}

// List returns a list of CountOptions setter functions.
func (rmdko *RewrapManyDataKeyOptionsBuilder) List() []func(*RewrapManyDataKeyOptions) error {
	return rmdko.Opts
}

// SetProvider sets the value for the Provider field. Provider identifies the new KMS provider.
// If omitted, encrypting uses the current KMS provider.
func (rmdko *RewrapManyDataKeyOptionsBuilder) SetProvider(provider string) *RewrapManyDataKeyOptionsBuilder {
	rmdko.Opts = append(rmdko.Opts, func(opts *RewrapManyDataKeyOptions) error {
		opts.Provider = &provider

		return nil
	})

	return rmdko
}

// SetMasterKey sets the value for the MasterKey field. MasterKey identifies the new masterKey.
// If omitted, rewraps with the current masterKey.
func (rmdko *RewrapManyDataKeyOptionsBuilder) SetMasterKey(masterKey interface{}) *RewrapManyDataKeyOptionsBuilder {
	rmdko.Opts = append(rmdko.Opts, func(opts *RewrapManyDataKeyOptions) error {
		opts.MasterKey = masterKey

		return nil
	})

	return rmdko
}
