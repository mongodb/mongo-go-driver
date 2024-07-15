// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// SearchIndexesOptions represents arguments that can be used to configure a
// SearchIndexView.
type SearchIndexesOptions struct {
	Name *string
	Type *string
}

// SearchIndexesOptionsBuilder contains options to configure search index
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type SearchIndexesOptionsBuilder struct {
	Opts []func(*SearchIndexesOptions) error
}

// SearchIndexes creates a new SearchIndexesOptions instance.
func SearchIndexes() *SearchIndexesOptionsBuilder {
	return &SearchIndexesOptionsBuilder{}
}

// ArgsSetters returns a list of CountArgs setter functions.
func (sio *SearchIndexesOptionsBuilder) ArgsSetters() []func(*SearchIndexesOptions) error {
	return sio.Opts
}

// SetName sets the value for the Name field.
func (sio *SearchIndexesOptionsBuilder) SetName(name string) *SearchIndexesOptionsBuilder {
	sio.Opts = append(sio.Opts, func(args *SearchIndexesOptions) error {
		args.Name = &name

		return nil
	})

	return sio
}

// SetType sets the value for the Type field.
func (sio *SearchIndexesOptionsBuilder) SetType(typ string) *SearchIndexesOptionsBuilder {
	sio.Opts = append(sio.Opts, func(args *SearchIndexesOptions) error {
		args.Type = &typ

		return nil
	})

	return sio
}

// CreateSearchIndexesOptions represents arguments that can be used to configure
// a SearchIndexView.CreateOne or SearchIndexView.CreateMany operation.
type CreateSearchIndexesOptions struct{}

// CreateSearchIndexesOptionsBuilder contains options to configure creating
// search indexes. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type CreateSearchIndexesOptionsBuilder struct {
	Opts []func(*CreateSearchIndexesOptions) error
}

// ArgsSetters returns a list of CreateSearchIndexesArgs setter functions.
func (csio *CreateSearchIndexesOptionsBuilder) ArgsSetters() []func(*CreateSearchIndexesOptions) error {
	return csio.Opts
}

// ListSearchIndexesOptions represents arguments that can be used to configure a
// SearchIndexView.List operation.
type ListSearchIndexesOptions struct {
	AggregateOptions *AggregateOptions
}

// ListSearchIndexesOptionsBuilder represents options that can be used to
// configure a SearchIndexView.List operation.
type ListSearchIndexesOptionsBuilder struct {
	Opts []func(*ListSearchIndexesOptions) error
}

// ArgsSetters returns a list of ListSearchIndexesArgs setter functions.
func (lsi *ListSearchIndexesOptionsBuilder) ArgsSetters() []func(*ListSearchIndexesOptions) error {
	return lsi.Opts
}

// DropSearchIndexOptions represents arguments that can be used to configure a
// SearchIndexView.DropOne operation.
type DropSearchIndexOptions struct{}

// DropSearchIndexOptionsBuilder contains options to configure dropping search
// indexes. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type DropSearchIndexOptionsBuilder struct {
	Opts []func(*DropSearchIndexOptions) error
}

// ArgsSetters returns a list of DropSearchIndexArgs setter functions.
func (dsio *DropSearchIndexOptionsBuilder) ArgsSetters() []func(*DropSearchIndexOptions) error {
	return dsio.Opts
}

// UpdateSearchIndexOptions represents arguments that can be used to configure a
// SearchIndexView.UpdateOne operation.
type UpdateSearchIndexOptions struct{}

// UpdateSearchIndexOptionsBuilder contains options to configure updating search
// indexes. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type UpdateSearchIndexOptionsBuilder struct {
	Opts []func(*UpdateSearchIndexOptions) error
}

// ArgsSetters returns a list of UpdateSearchIndexArgs setter functions.
func (usio *UpdateSearchIndexOptionsBuilder) ArgsSetters() []func(*UpdateSearchIndexOptions) error {
	return usio.Opts
}
