// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// ListDatabasesArgs represents arguments that can be used to configure a ListDatabases operation.
type ListDatabasesArgs struct {
	// If true, only the Name field of the returned DatabaseSpecification objects will be populated. The default value
	// is false.
	NameOnly *bool

	// If true, only the databases which the user is authorized to see will be returned. For more information about
	// the behavior of this option, see https://www.mongodb.com/docs/manual/reference/privilege-actions/#find. The default
	// value is true.
	AuthorizedDatabases *bool
}

// ListDatabasesOptions represents functional options that configure a ListDatabasesArgs.
type ListDatabasesOptions struct {
	Opts []func(*ListDatabasesArgs) error
}

// ListDatabases creates a new ListDatabasesOptions instance.
func ListDatabases() *ListDatabasesOptions {
	return &ListDatabasesOptions{}
}

// ArgsSetters returns a list of ListDatabasesArgs setter functions.
func (ld *ListDatabasesOptions) ArgsSetters() []func(*ListDatabasesArgs) error {
	return ld.Opts
}

// SetNameOnly sets the value for the NameOnly field.
func (ld *ListDatabasesOptions) SetNameOnly(b bool) *ListDatabasesOptions {
	ld.Opts = append(ld.Opts, func(args *ListDatabasesArgs) error {
		args.NameOnly = &b
		return nil
	})
	return ld
}

// SetAuthorizedDatabases sets the value for the AuthorizedDatabases field.
func (ld *ListDatabasesOptions) SetAuthorizedDatabases(b bool) *ListDatabasesOptions {
	ld.Opts = append(ld.Opts, func(args *ListDatabasesArgs) error {
		args.AuthorizedDatabases = &b
		return nil
	})
	return ld
}
