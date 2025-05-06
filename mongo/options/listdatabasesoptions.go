// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// ListDatabasesOptions represents arguments that can be used to configure a
// ListDatabases operation.
//
// See corresponding setter methods for documentation.
type ListDatabasesOptions struct {
	NameOnly            *bool
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

// List returns a list of ListDatabasesOptions setter functions.
func (ld *ListDatabasesOptionsBuilder) List() []func(*ListDatabasesOptions) error {
	return ld.Opts
}

// SetNameOnly sets the value for the NameOnly field. If true, only the Name field of the returned
// DatabaseSpecification objects will be populated. The default value is false.
func (ld *ListDatabasesOptionsBuilder) SetNameOnly(b bool) *ListDatabasesOptionsBuilder {
	ld.Opts = append(ld.Opts, func(opts *ListDatabasesOptions) error {
		opts.NameOnly = &b
		return nil
	})
	return ld
}

// SetAuthorizedDatabases sets the value for the AuthorizedDatabases field. If true, only the
// databases which the user is authorized to see will be returned. For more information about the
// behavior of this option, see https://www.mongodb.com/docs/manual/reference/privilege-actions/#find.
// The default value is true.
func (ld *ListDatabasesOptionsBuilder) SetAuthorizedDatabases(b bool) *ListDatabasesOptionsBuilder {
	ld.Opts = append(ld.Opts, func(opts *ListDatabasesOptions) error {
		opts.AuthorizedDatabases = &b
		return nil
	})
	return ld
}
