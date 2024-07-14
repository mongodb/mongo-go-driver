// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// SearchIndexesArgs represents arguments that can be used to configure a
// SearchIndexView.
type SearchIndexesArgs struct {
	Name *string
	Type *string
}

// SearchIndexesOptions contains options to configure search index operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type SearchIndexesOptions struct {
	Opts []func(*SearchIndexesArgs) error
}

// SearchIndexes creates a new SearchIndexesOptions instance.
func SearchIndexes() *SearchIndexesOptions {
	return &SearchIndexesOptions{}
}

// ArgsSetters returns a list of CountArgs setter functions.
func (sio *SearchIndexesOptions) ArgsSetters() []func(*SearchIndexesArgs) error {
	return sio.Opts
}

// SetName sets the value for the Name field.
func (sio *SearchIndexesOptions) SetName(name string) *SearchIndexesOptions {
	sio.Opts = append(sio.Opts, func(args *SearchIndexesArgs) error {
		args.Name = &name

		return nil
	})

	return sio
}

// SetType sets the value for the Type field.
func (sio *SearchIndexesOptions) SetType(typ string) *SearchIndexesOptions {
	sio.Opts = append(sio.Opts, func(args *SearchIndexesArgs) error {
		args.Type = &typ

		return nil
	})

	return sio
}

// CreateSearchIndexesArgs represents arguments that can be used to configure a
// SearchIndexView.CreateOne or SearchIndexView.CreateMany operation.
type CreateSearchIndexesArgs struct{}

// CreateSearchIndexesOptions contains options to configure creating search
// indexes. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type CreateSearchIndexesOptions struct {
	Opts []func(*CreateSearchIndexesArgs) error
}

// ArgsSetters returns a list of CreateSearchIndexesArgs setter functions.
func (csio *CreateSearchIndexesOptions) ArgsSetters() []func(*CreateSearchIndexesArgs) error {
	return csio.Opts
}

// ListSearchIndexesArgs represents arguments that can be used to configure a
// SearchIndexView.List operation.
type ListSearchIndexesArgs struct {
	AggregateOptions *AggregateOptions
}

// ListSearchIndexesOptions represents options that can be used to configure a
// SearchIndexView.List operation.
type ListSearchIndexesOptions struct {
	Opts []func(*ListSearchIndexesArgs) error
}

// ArgsSetters returns a list of ListSearchIndexesArgs setter functions.
func (lsi *ListSearchIndexesOptions) ArgsSetters() []func(*ListSearchIndexesArgs) error {
	return lsi.Opts
}

// DropSearchIndexArgs represents arguments that can be used to configure a
// SearchIndexView.DropOne operation.
type DropSearchIndexArgs struct{}

// DropSearchIndexOptions contains options to configure dropping search indexes.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type DropSearchIndexOptions struct {
	Opts []func(*DropSearchIndexArgs) error
}

// ArgsSetters returns a list of DropSearchIndexArgs setter functions.
func (dsio *DropSearchIndexOptions) ArgsSetters() []func(*DropSearchIndexArgs) error {
	return dsio.Opts
}

// UpdateSearchIndexArgs represents arguments that can be used to configure a
// SearchIndexView.UpdateOne operation.
type UpdateSearchIndexArgs struct{}

// UpdateSearchIndexOptions contains options to configure updating search
// indexes. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type UpdateSearchIndexOptions struct {
	Opts []func(*UpdateSearchIndexArgs) error
}

// ArgsSetters returns a list of UpdateSearchIndexArgs setter functions.
func (usio *UpdateSearchIndexOptions) ArgsSetters() []func(*UpdateSearchIndexArgs) error {
	return usio.Opts
}
