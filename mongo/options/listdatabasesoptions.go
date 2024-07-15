// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// ListDatabasesOptions represents arguments that can be used to configure a
// ListDatabases operation.
type ListDatabasesOptions struct {
	// If true, only the Name field of the returned DatabaseSpecification objects will be populated. The default value
	// is false.
	NameOnly *bool

	// If true, only the databases which the user is authorized to see will be returned. For more information about
	// the behavior of this option, see https://www.mongodb.com/docs/manual/reference/privilege-actions/#find. The default
	// value is true.
	AuthorizedDatabases *bool
}

// ListDatabasesOptionsBuilder represents functional options that configure a
// ListDatabasesopts.
type ListDatabasesOptionsBuilder struct {
	Opts []func(*ListDatabasesOptions) error
}

// ListDatabases creates a new ListDatabasesOptions instance.
func ListDatabases() *ListDatabasesOptionsBuilder {
	return &ListDatabasesOptionsBuilder{}
}

// OptionsSetters returns a list of ListDatabasesopts setter functions.
func (ld *ListDatabasesOptionsBuilder) OptionsSetters() []func(*ListDatabasesOptions) error {
	return ld.Opts
}

// SetNameOnly sets the value for the NameOnly field.
func (ld *ListDatabasesOptionsBuilder) SetNameOnly(b bool) *ListDatabasesOptionsBuilder {
	ld.Opts = append(ld.Opts, func(opts *ListDatabasesOptions) error {
		opts.NameOnly = &b
		return nil
	})
	return ld
}

// SetAuthorizedDatabases sets the value for the AuthorizedDatabases field.
func (ld *ListDatabasesOptionsBuilder) SetAuthorizedDatabases(b bool) *ListDatabasesOptionsBuilder {
	ld.Opts = append(ld.Opts, func(opts *ListDatabasesOptions) error {
		opts.AuthorizedDatabases = &b
		return nil
	})
	return ld
}
