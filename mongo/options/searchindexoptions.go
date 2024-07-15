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

// OptionsSetters returns a list of CountOptions setter functions.
func (sio *SearchIndexesOptionsBuilder) OptionsSetters() []func(*SearchIndexesOptions) error {
	return sio.Opts
}

// SetName sets the value for the Name field.
func (sio *SearchIndexesOptionsBuilder) SetName(name string) *SearchIndexesOptionsBuilder {
	sio.Opts = append(sio.Opts, func(opts *SearchIndexesOptions) error {
		opts.Name = &name

		return nil
	})

	return sio
}

// SetType sets the value for the Type field.
func (sio *SearchIndexesOptionsBuilder) SetType(typ string) *SearchIndexesOptionsBuilder {
	sio.Opts = append(sio.Opts, func(opts *SearchIndexesOptions) error {
		opts.Type = &typ

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

// OptionsSetters returns a list of CreateSearchIndexesOptions setter functions.
func (csio *CreateSearchIndexesOptionsBuilder) OptionsSetters() []func(*CreateSearchIndexesOptions) error {
	return csio.Opts
}

// ListSearchIndexesOptions represents arguments that can be used to configure a
// SearchIndexView.List operation.
type ListSearchIndexesOptions struct {
	AggregateOptions *AggregateOptions
}

// ListSearchIndexesOptionsBuilder contains options that can be used to
// configure a SearchIndexView.List operation.
type ListSearchIndexesOptionsBuilder struct {
	Opts []func(*ListSearchIndexesOptions) error
}

// OptionsSetters returns a list of ListSearchIndexesOptions setter functions.
func (lsi *ListSearchIndexesOptionsBuilder) OptionsSetters() []func(*ListSearchIndexesOptions) error {
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

// OptionsSetters returns a list of DropSearchIndexOptions setter functions.
func (dsio *DropSearchIndexOptionsBuilder) OptionsSetters() []func(*DropSearchIndexOptions) error {
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

// OptionsSetters returns a list of UpdateSearchIndexOptions setter functions.
func (usio *UpdateSearchIndexOptionsBuilder) OptionsSetters() []func(*UpdateSearchIndexOptions) error {
	return usio.Opts
}
