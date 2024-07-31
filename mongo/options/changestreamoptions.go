// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ChangeStreamOptions represents arguments that can be used to configure a Watch operation.
type ChangeStreamOptions struct {
	// The maximum number of documents to be included in each batch returned by the server.
	BatchSize *int32

	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs,
	// and currentOp queries to help trace the operation. The default is nil,
	// which means that no comment will be included in the logs.
	Comment interface{}

	// Specifies how the updated document should be returned in change notifications for update operations. The default
	// is options.Default, which means that only partial update deltas will be included in the change notification.
	FullDocument *FullDocument

	// Specifies how the pre-update document should be returned in change notifications for update operations. The default
	// is options.Off, which means that the pre-update document will not be included in the change notification.
	FullDocumentBeforeChange *FullDocument

	// The maximum amount of time that the server should wait for new documents to satisfy a tailable cursor query.
	MaxAwaitTime *time.Duration

	// A document specifying the logical starting point for the change stream. Only changes corresponding to an oplog
	// entry immediately after the resume token will be returned. If this is specified, StartAtOperationTime and
	// StartAfter must not be set.
	ResumeAfter interface{}

	// ShowExpandedEvents specifies whether the server will return an expanded list of change stream events. Additional
	// events include: createIndexes, dropIndexes, modify, create, shardCollection, reshardCollection and
	// refineCollectionShardKey. This option is only valid for MongoDB versions >= 6.0.
	ShowExpandedEvents *bool

	// If specified, the change stream will only return changes that occurred at or after the given timestamp. This
	// option is only valid for MongoDB versions >= 4.0. If this is specified, ResumeAfter and StartAfter must not be
	// set.
	StartAtOperationTime *bson.Timestamp

	// A document specifying the logical starting point for the change stream. This is similar to the ResumeAfter
	// option, but allows a resume token from an "invalidate" notification to be used. This allows a change stream on a
	// collection to be resumed after the collection has been dropped and recreated or renamed. Only changes
	// corresponding to an oplog entry immediately after the specified token will be returned. If this is specified,
	// ResumeAfter and StartAtOperationTime must not be set. This option is only valid for MongoDB versions >= 4.1.1.
	StartAfter interface{}

	// Custom options to be added to the initial aggregate for the change stream. Key-value pairs of the BSON map should
	// correlate with desired option names and values. Values must be Marshalable. Custom options may conflict with
	// non-custom options, and custom options bypass client-side validation. Prefer using non-custom options where possible.
	Custom bson.M

	// Custom options to be added to the $changeStream stage in the initial aggregate. Key-value pairs of the BSON map should
	// correlate with desired option names and values. Values must be Marshalable. Custom pipeline options bypass client-side
	// validation. Prefer using non-custom options where possible.
	CustomPipeline bson.M
}

// ChangeStreamOptionsBuilder contains options to configure change stream
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type ChangeStreamOptionsBuilder struct {
	Opts []func(*ChangeStreamOptions) error
}

// ChangeStream creates a new ChangeStreamOptions instance.
func ChangeStream() *ChangeStreamOptionsBuilder {
	return &ChangeStreamOptionsBuilder{}
}

// List returns a list of ChangeStreamOptions setter functions.
func (cso *ChangeStreamOptionsBuilder) List() []func(*ChangeStreamOptions) error {
	return cso.Opts
}

// SetBatchSize sets the value for the BatchSize field.
func (cso *ChangeStreamOptionsBuilder) SetBatchSize(i int32) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.BatchSize = &i
		return nil
	})
	return cso
}

// SetCollation sets the value for the Collation field.
func (cso *ChangeStreamOptionsBuilder) SetCollation(c Collation) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.Collation = &c
		return nil
	})
	return cso
}

// SetComment sets the value for the Comment field.
func (cso *ChangeStreamOptionsBuilder) SetComment(comment interface{}) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.Comment = comment
		return nil
	})
	return cso
}

// SetFullDocument sets the value for the FullDocument field.
func (cso *ChangeStreamOptionsBuilder) SetFullDocument(fd FullDocument) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.FullDocument = &fd
		return nil
	})
	return cso
}

// SetFullDocumentBeforeChange sets the value for the FullDocumentBeforeChange field.
func (cso *ChangeStreamOptionsBuilder) SetFullDocumentBeforeChange(fdbc FullDocument) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.FullDocumentBeforeChange = &fdbc
		return nil
	})
	return cso
}

// SetMaxAwaitTime sets the value for the MaxAwaitTime field.
func (cso *ChangeStreamOptionsBuilder) SetMaxAwaitTime(d time.Duration) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.MaxAwaitTime = &d
		return nil
	})
	return cso
}

// SetResumeAfter sets the value for the ResumeAfter field.
func (cso *ChangeStreamOptionsBuilder) SetResumeAfter(rt interface{}) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.ResumeAfter = rt
		return nil
	})
	return cso
}

// SetShowExpandedEvents sets the value for the ShowExpandedEvents field.
func (cso *ChangeStreamOptionsBuilder) SetShowExpandedEvents(see bool) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.ShowExpandedEvents = &see
		return nil
	})
	return cso
}

// SetStartAtOperationTime sets the value for the StartAtOperationTime field.
func (cso *ChangeStreamOptionsBuilder) SetStartAtOperationTime(t *bson.Timestamp) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.StartAtOperationTime = t
		return nil
	})
	return cso
}

// SetStartAfter sets the value for the StartAfter field.
func (cso *ChangeStreamOptionsBuilder) SetStartAfter(sa interface{}) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.StartAfter = sa
		return nil
	})
	return cso
}

// SetCustom sets the value for the Custom field. Key-value pairs of the BSON map should correlate
// with desired option names and values. Values must be Marshalable. Custom options may conflict
// with non-custom options, and custom options bypass client-side validation. Prefer using non-custom
// options where possible.
func (cso *ChangeStreamOptionsBuilder) SetCustom(c bson.M) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.Custom = c
		return nil
	})
	return cso
}

// SetCustomPipeline sets the value for the CustomPipeline field. Key-value pairs of the BSON map
// should correlate with desired option names and values. Values must be Marshalable. Custom pipeline
// options bypass client-side validation. Prefer using non-custom options where possible.
func (cso *ChangeStreamOptionsBuilder) SetCustomPipeline(cp bson.M) *ChangeStreamOptionsBuilder {
	cso.Opts = append(cso.Opts, func(opts *ChangeStreamOptions) error {
		opts.CustomPipeline = cp
		return nil
	})
	return cso
}
