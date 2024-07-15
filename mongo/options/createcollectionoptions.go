// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"
)

// DefaultIndexOptions represents the default arguments for a collection to
// apply on new indexes. This type can be used when creating a new collection
// through the CreateCollectionOptions.SetDefaultIndexOptions method.
type DefaultIndexOptions struct {
	// Specifies the storage engine to use for the index. The value must be a document in the form
	// {<storage engine name>: <options>}. The default value is nil, which means that the default storage engine
	// will be used.
	StorageEngine interface{}
}

// DefaultIndexOptionsBuilder contains options to configure default index
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type DefaultIndexOptionsBuilder struct {
	Opts []func(*DefaultIndexOptions) error
}

// DefaultIndex creates a new DefaultIndexOptions instance.
func DefaultIndex() *DefaultIndexOptionsBuilder {
	return &DefaultIndexOptionsBuilder{}
}

// OptionsSetters returns a list of DefaultIndexopts setter functions.
func (d *DefaultIndexOptionsBuilder) OptionsSetters() []func(*DefaultIndexOptions) error {
	return d.Opts
}

// SetStorageEngine sets the value for the StorageEngine field.
func (d *DefaultIndexOptionsBuilder) SetStorageEngine(storageEngine interface{}) *DefaultIndexOptionsBuilder {
	d.Opts = append(d.Opts, func(opts *DefaultIndexOptions) error {
		opts.StorageEngine = storageEngine

		return nil
	})

	return d
}

// TimeSeriesOptions specifies arguments on a time-series collection.
type TimeSeriesOptions struct {
	// TimeField is the top-level field to be used for time. Inserted documents must have this field,
	// and the field must be of the BSON UTC datetime type (0x9).
	TimeField string

	// MetaField is the name of the top-level field describing the series. This field is used to group
	// related data and may be of any BSON type, except for array. This name may not be the same
	// as the TimeField or _id. This field is optional.
	MetaField *string

	// Granularity is the granularity of time-series data. Allowed granularity options are
	// "seconds", "minutes" and "hours". This field is optional.
	Granularity *string

	// BucketMaxSpan is the maximum range of time values for a bucket. The
	// time.Duration is rounded down to the nearest second and applied as
	// the command option: "bucketRoundingSeconds". This field is optional.
	BucketMaxSpan *time.Duration

	// BucketRounding is used to determine the minimum time boundary when
	// opening a new bucket by rounding the first timestamp down to the next
	// multiple of this value. The time.Duration is rounded down to the
	// nearest second and applied as the command option:
	// "bucketRoundingSeconds". This field is optional.
	BucketRounding *time.Duration
}

// TimeSeriesOptionsBuilder contains options to configure timeseries operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type TimeSeriesOptionsBuilder struct {
	Opts []func(*TimeSeriesOptions) error
}

// TimeSeries creates a new TimeSeriesOptions instance.
func TimeSeries() *TimeSeriesOptionsBuilder {
	return &TimeSeriesOptionsBuilder{}
}

// OptionsSetters returns a list of TimeSeriesopts setter functions.
func (tso *TimeSeriesOptionsBuilder) OptionsSetters() []func(*TimeSeriesOptions) error {
	return tso.Opts
}

// SetTimeField sets the value for the TimeField.
func (tso *TimeSeriesOptionsBuilder) SetTimeField(timeField string) *TimeSeriesOptionsBuilder {
	tso.Opts = append(tso.Opts, func(opts *TimeSeriesOptions) error {
		opts.TimeField = timeField

		return nil
	})

	return tso
}

// SetMetaField sets the value for the MetaField.
func (tso *TimeSeriesOptionsBuilder) SetMetaField(metaField string) *TimeSeriesOptionsBuilder {
	tso.Opts = append(tso.Opts, func(opts *TimeSeriesOptions) error {
		opts.MetaField = &metaField

		return nil
	})

	return tso
}

// SetGranularity sets the value for Granularity.
func (tso *TimeSeriesOptionsBuilder) SetGranularity(granularity string) *TimeSeriesOptionsBuilder {
	tso.Opts = append(tso.Opts, func(opts *TimeSeriesOptions) error {
		opts.Granularity = &granularity

		return nil
	})

	return tso
}

// SetBucketMaxSpan sets the value for BucketMaxSpan.
func (tso *TimeSeriesOptionsBuilder) SetBucketMaxSpan(dur time.Duration) *TimeSeriesOptionsBuilder {
	tso.Opts = append(tso.Opts, func(opts *TimeSeriesOptions) error {
		opts.BucketMaxSpan = &dur

		return nil
	})

	return tso
}

// SetBucketRounding sets the value for BucketRounding.
func (tso *TimeSeriesOptionsBuilder) SetBucketRounding(dur time.Duration) *TimeSeriesOptionsBuilder {
	tso.Opts = append(tso.Opts, func(opts *TimeSeriesOptions) error {
		opts.BucketRounding = &dur

		return nil
	})

	return tso
}

// CreateCollectionOptions represents options that can be used to configure a
// CreateCollection operation.
type CreateCollectionOptions struct {
	// Specifies if the collection is capped (see https://www.mongodb.com/docs/manual/core/capped-collections/). If true,
	// the SizeInBytes option must also be specified. The default value is false.
	Capped *bool

	// Specifies the default collation for the new collection. This option is only valid for MongoDB versions >= 3.4.
	// For previous server versions, the driver will return an error if this option is used. The default value is nil.
	Collation *Collation

	// Specifies how change streams opened against the collection can return pre- and post-images of updated
	// documents. The value must be a document in the form {<option name>: <options>}. This option is only valid for
	// MongoDB versions >= 6.0. The default value is nil, which means that change streams opened against the collection
	// will not return pre- and post-images of updated documents in any way.
	ChangeStreamPreAndPostImages interface{}

	// Specifies a default configuration for indexes on the collection. This option is only valid for MongoDB versions
	// >= 3.4. The default value is nil, meaning indexes will be configured using server defaults.
	DefaultIndexOptions *DefaultIndexOptionsBuilder

	// Specifies the maximum number of documents allowed in a capped collection. The limit specified by the SizeInBytes
	// option takes precedence over this option. If a capped collection reaches its size limit, old documents will be
	// removed, regardless of the number of documents in the collection. The default value is 0, meaning the maximum
	// number of documents is unbounded.
	MaxDocuments *int64

	// Specifies the maximum size in bytes for a capped collection. The default value is 0.
	SizeInBytes *int64

	// Specifies the storage engine to use for the index. The value must be a document in the form
	// {<storage engine name>: <options>}. The default value is nil, which means that the default storage engine
	// will be used.
	StorageEngine interface{}

	// Specifies what should happen if a document being inserted does not pass validation. Valid values are "error" and
	// "warn". See https://www.mongodb.com/docs/manual/core/schema-validation/#accept-or-reject-invalid-documents for more
	// information. This option is only valid for MongoDB versions >= 3.2. The default value is "error".
	ValidationAction *string

	// Specifies how strictly the server applies validation rules to existing documents in the collection during update
	// operations. Valid values are "off", "strict", and "moderate". See
	// https://www.mongodb.com/docs/manual/core/schema-validation/#existing-documents for more information. This option is
	// only valid for MongoDB versions >= 3.2. The default value is "strict".
	ValidationLevel *string

	// A document specifying validation rules for the collection. See
	// https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about schema validation. This option
	// is only valid for MongoDB versions >= 3.2. The default value is nil, meaning no validator will be used for the
	// collection.
	Validator interface{}

	// Value indicating after how many seconds old time-series data should be deleted. See
	// https://www.mongodb.com/docs/manual/reference/command/create/ for supported options, and
	// https://www.mongodb.com/docs/manual/core/timeseries-collections/ for more information on time-series
	// collections.
	//
	// This option is only valid for MongoDB versions >= 5.0
	ExpireAfterSeconds *int64

	// Options for specifying a time-series collection. See
	// https://www.mongodb.com/docs/manual/reference/command/create/ for supported options, and
	// https://www.mongodb.com/docs/manual/core/timeseries-collections/ for more information on time-series
	// collections.
	//
	// This option is only valid for MongoDB versions >= 5.0
	TimeSeriesOptions *TimeSeriesOptionsBuilder

	// EncryptedFields configures encrypted fields.
	//
	// This option is only valid for MongoDB versions >= 6.0
	EncryptedFields interface{}

	// ClusteredIndex is used to create a collection with a clustered index.
	//
	// This option is only valid for MongoDB versions >= 5.3
	ClusteredIndex interface{}
}

// CreateCollectionOptionsBuilder contains options to configure a new
// collection. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type CreateCollectionOptionsBuilder struct {
	Opts []func(*CreateCollectionOptions) error
}

// CreateCollection creates a new CreateCollectionOptions instance.
func CreateCollection() *CreateCollectionOptionsBuilder {
	return &CreateCollectionOptionsBuilder{}
}

// OptionsSetters returns a list of CreateCollectionopts setter functions.
func (c *CreateCollectionOptionsBuilder) OptionsSetters() []func(*CreateCollectionOptions) error {
	return c.Opts
}

// SetCapped sets the value for the Capped field.
func (c *CreateCollectionOptionsBuilder) SetCapped(capped bool) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.Capped = &capped

		return nil
	})

	return c
}

// SetCollation sets the value for the Collation field.
func (c *CreateCollectionOptionsBuilder) SetCollation(collation *Collation) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.Collation = collation

		return nil
	})

	return c
}

// SetChangeStreamPreAndPostImages sets the value for the ChangeStreamPreAndPostImages field.
func (c *CreateCollectionOptionsBuilder) SetChangeStreamPreAndPostImages(csppi interface{}) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.ChangeStreamPreAndPostImages = &csppi

		return nil
	})

	return c
}

// SetDefaultIndexOptions sets the value for the DefaultIndexOptions field.
func (c *CreateCollectionOptionsBuilder) SetDefaultIndexOptions(iopts *DefaultIndexOptionsBuilder) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.DefaultIndexOptions = iopts

		return nil
	})

	return c
}

// SetMaxDocuments sets the value for the MaxDocuments field.
func (c *CreateCollectionOptionsBuilder) SetMaxDocuments(max int64) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.MaxDocuments = &max

		return nil
	})

	return c
}

// SetSizeInBytes sets the value for the SizeInBytes field.
func (c *CreateCollectionOptionsBuilder) SetSizeInBytes(size int64) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.SizeInBytes = &size

		return nil
	})

	return c
}

// SetStorageEngine sets the value for the StorageEngine field.
func (c *CreateCollectionOptionsBuilder) SetStorageEngine(storageEngine interface{}) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.StorageEngine = &storageEngine

		return nil
	})

	return c
}

// SetValidationAction sets the value for the ValidationAction field.
func (c *CreateCollectionOptionsBuilder) SetValidationAction(action string) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.ValidationAction = &action

		return nil
	})

	return c
}

// SetValidationLevel sets the value for the ValidationLevel field.
func (c *CreateCollectionOptionsBuilder) SetValidationLevel(level string) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.ValidationLevel = &level

		return nil
	})

	return c
}

// SetValidator sets the value for the Validator field.
func (c *CreateCollectionOptionsBuilder) SetValidator(validator interface{}) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.Validator = validator

		return nil
	})

	return c
}

// SetExpireAfterSeconds sets the value for the ExpireAfterSeconds field.
func (c *CreateCollectionOptionsBuilder) SetExpireAfterSeconds(eas int64) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.ExpireAfterSeconds = &eas

		return nil
	})

	return c
}

// SetTimeSeriesOptions sets the options for time-series collections.
func (c *CreateCollectionOptionsBuilder) SetTimeSeriesOptions(timeSeriesOpts *TimeSeriesOptionsBuilder) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.TimeSeriesOptions = timeSeriesOpts

		return nil
	})

	return c
}

// SetEncryptedFields sets the encrypted fields for encrypted collections.
func (c *CreateCollectionOptionsBuilder) SetEncryptedFields(encryptedFields interface{}) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.EncryptedFields = encryptedFields

		return nil
	})

	return c
}

// SetClusteredIndex sets the value for the ClusteredIndex field.
func (c *CreateCollectionOptionsBuilder) SetClusteredIndex(clusteredIndex interface{}) *CreateCollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateCollectionOptions) error {
		opts.ClusteredIndex = clusteredIndex

		return nil
	})

	return c
}

// CreateViewOptions represents arguments that can be used to configure a
// CreateView operation.
type CreateViewOptions struct {
	// Specifies the default collation for the new collection. This option is only valid for MongoDB versions >= 3.4.
	// For previous server versions, the driver will return an error if this option is used. The default value is nil.
	Collation *Collation
}

// CreateViewOptionsBuilder contains options to configure a new view. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type CreateViewOptionsBuilder struct {
	Opts []func(*CreateViewOptions) error
}

// CreateView creates an new CreateViewOptions instance.
func CreateView() *CreateViewOptionsBuilder {
	return &CreateViewOptionsBuilder{}
}

// OptionsSetters returns a list of TimeSeriesopts setter functions.
func (c *CreateViewOptionsBuilder) OptionsSetters() []func(*CreateViewOptions) error {
	return c.Opts
}

// SetCollation sets the value for the Collation field.
func (c *CreateViewOptionsBuilder) SetCollation(collation *Collation) *CreateViewOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CreateViewOptions) error {
		opts.Collation = collation

		return nil
	})

	return c
}
