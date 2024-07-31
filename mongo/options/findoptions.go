// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"
)

// FindOptions represents arguments that can be used to configure a Find
// operation.
type FindOptions struct {
	// AllowPartial results specifies whether the Find operation on a sharded cluster can return partial results if some
	// shards are down rather than returning an error. The default value is false.
	AllowPartialResults *bool

	// Collation specifies a collation to use for string comparisons during the operation. This option is only valid for
	// MongoDB versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs,
	// and currentOp queries to help trace the operation. The default is nil,
	// which means that no comment will be included in the logs.
	Comment interface{}

	// Hint is the index to use for the Find operation. This should either be the index name as a string or the index
	// specification as a document. The driver will return an error if the hint parameter is a multi-key map. The default
	// value is nil, which means that no hint will be sent.
	Hint interface{}

	// Max is a document specifying the exclusive upper bound for a specific index. The default value is nil, which means that
	// there is no maximum value.
	Max interface{}

	// MaxAwaitTime is the maximum amount of time that the server should wait for new documents to satisfy a tailable cursor
	// query. This option is only valid for tailable await cursors (see the CursorType option for more information) and
	// MongoDB versions >= 3.2. For other cursor types or previous server versions, this option is ignored.
	MaxAwaitTime *time.Duration

	// Min is a document specifying the inclusive lower bound for a specific index. The default value is 0, which means that
	// there is no minimum value.
	Min interface{}

	// Project is a document describing which fields will be included in the documents returned by the Find operation. The
	// default value is nil, which means all fields will be included.
	Projection interface{}

	// ReturnKey specifies whether the documents returned by the Find operation will only contain fields corresponding to the
	// index used. The default value is false.
	ReturnKey *bool

	// ShowRecordID specifies whether a $recordId field with a record identifier will be included in the documents returned by
	// the Find operation. The default value is false.
	ShowRecordID *bool

	// Skip is the number of documents to skip before adding documents to the result. The default value is 0.
	Skip *int64

	// Sort is a document specifying the order in which documents should be returned.  The driver will return an error if the
	// sort parameter is a multi-key map.
	Sort interface{}

	// The above are in common with FindOneopts.

	// AllowDiskUse specifies whether the server can write temporary data to disk while executing the Find operation.
	// This option is only valid for MongoDB versions >= 4.4. Server versions >= 3.2 will report an error if this option
	// is specified. For server versions < 3.2, the driver will return a client-side error if this option is specified.
	// The default value is false.
	AllowDiskUse *bool

	// BatchSize is the maximum number of documents to be included in each batch returned by the server.
	BatchSize *int32

	// CursorType specifies the type of cursor that should be created for the operation. The default is NonTailable, which
	// means that the cursor will be closed by the server when the last batch of documents is retrieved.
	CursorType *CursorType

	// Let specifies parameters for the find expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}

	// Limit is the maximum number of documents to return. The default value is 0, which means that all documents matching the
	// filter will be returned. A negative limit specifies that the resulting documents should be returned in a single
	// batch. The default value is 0.
	Limit *int64

	// NoCursorTimeout specifies whether the cursor created by the operation will not timeout after a period of inactivity.
	// The default value is false.
	NoCursorTimeout *bool
}

// FindOptionsBuilder represents functional options that configure an Findopts.
type FindOptionsBuilder struct {
	Opts []func(*FindOptions) error
}

// Find creates a new FindOptions instance.
func Find() *FindOptionsBuilder {
	return &FindOptionsBuilder{}
}

// List returns a list of FindOptions setter functions.
func (f *FindOptionsBuilder) List() []func(*FindOptions) error {
	return f.Opts
}

// SetAllowDiskUse sets the value for the AllowDiskUse field.
func (f *FindOptionsBuilder) SetAllowDiskUse(b bool) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.AllowDiskUse = &b
		return nil
	})
	return f
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOptionsBuilder) SetAllowPartialResults(b bool) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.AllowPartialResults = &b
		return nil
	})
	return f
}

// SetBatchSize sets the value for the BatchSize field.
func (f *FindOptionsBuilder) SetBatchSize(i int32) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.BatchSize = &i
		return nil
	})
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOptionsBuilder) SetCollation(collation *Collation) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Collation = collation
		return nil
	})
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOptionsBuilder) SetComment(comment interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Comment = &comment
		return nil
	})
	return f
}

// SetCursorType sets the value for the CursorType field.
func (f *FindOptionsBuilder) SetCursorType(ct CursorType) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.CursorType = &ct
		return nil
	})
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOptionsBuilder) SetHint(hint interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Hint = hint
		return nil
	})
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOptionsBuilder) SetLet(let interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Let = let
		return nil
	})
	return f
}

// SetLimit sets the value for the Limit field.
func (f *FindOptionsBuilder) SetLimit(i int64) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Limit = &i
		return nil
	})
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOptionsBuilder) SetMax(max interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Max = max
		return nil
	})
	return f
}

// SetMaxAwaitTime sets the value for the MaxAwaitTime field.
func (f *FindOptionsBuilder) SetMaxAwaitTime(d time.Duration) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.MaxAwaitTime = &d
		return nil
	})
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOptionsBuilder) SetMin(min interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Min = min
		return nil
	})
	return f
}

// SetNoCursorTimeout sets the value for the NoCursorTimeout field.
func (f *FindOptionsBuilder) SetNoCursorTimeout(b bool) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.NoCursorTimeout = &b
		return nil
	})
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOptionsBuilder) SetProjection(projection interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Projection = projection
		return nil
	})
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOptionsBuilder) SetReturnKey(b bool) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.ReturnKey = &b
		return nil
	})
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOptionsBuilder) SetShowRecordID(b bool) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.ShowRecordID = &b
		return nil
	})
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOptionsBuilder) SetSkip(i int64) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Skip = &i
		return nil
	})
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOptionsBuilder) SetSort(sort interface{}) *FindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOptions) error {
		opts.Sort = sort
		return nil
	})
	return f
}

// FindOneOptions represents arguments that can be used to configure a FindOne
// operation.
type FindOneOptions struct {
	// If true, an operation on a sharded cluster can return partial results if some shards are down rather than
	// returning an error. The default value is false.
	AllowPartialResults *bool

	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs,
	// and currentOp queries to help trace the operation. The default is nil,
	// which means that no comment will be included in the logs.
	Comment interface{}

	// The index to use for the aggregation. This should either be the index name as a string or the index specification
	// as a document. The driver will return an error if the hint parameter is a multi-key map. The default value is nil,
	// which means that no hint will be sent.
	Hint interface{}

	// A document specifying the exclusive upper bound for a specific index. The default value is nil, which means that
	// there is no maximum value.
	Max interface{}

	// A document specifying the inclusive lower bound for a specific index. The default value is 0, which means that
	// there is no minimum value.
	Min interface{}

	// A document describing which fields will be included in the document returned by the operation. The default value
	// is nil, which means all fields will be included.
	Projection interface{}

	// If true, the document returned by the operation will only contain fields corresponding to the index used. The
	// default value is false.
	ReturnKey *bool

	// If true, a $recordId field with a record identifier will be included in the document returned by the operation.
	// The default value is false.
	ShowRecordID *bool

	// The number of documents to skip before selecting the document to be returned. The default value is 0.
	Skip *int64

	// A document specifying the sort order to apply to the query. The first document in the sorted order will be
	// returned. The driver will return an error if the sort parameter is a multi-key map.
	Sort interface{}
}

// FindOneOptionsBuilder represents functional options that configure an
// FindOneopts.
type FindOneOptionsBuilder struct {
	Opts []func(*FindOneOptions) error
}

// FindOne creates a new FindOneOptions instance.
func FindOne() *FindOneOptionsBuilder {
	return &FindOneOptionsBuilder{}
}

// List returns a list of FindOneOptions setter functions.
func (f *FindOneOptionsBuilder) List() []func(*FindOneOptions) error {
	return f.Opts
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOneOptionsBuilder) SetAllowPartialResults(b bool) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.AllowPartialResults = &b
		return nil
	})
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneOptionsBuilder) SetCollation(collation *Collation) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Collation = collation
		return nil
	})
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneOptionsBuilder) SetComment(comment interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Comment = &comment
		return nil
	})
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneOptionsBuilder) SetHint(hint interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Hint = hint
		return nil
	})
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOneOptionsBuilder) SetMax(max interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Max = max
		return nil
	})
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOneOptionsBuilder) SetMin(min interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Min = min
		return nil
	})
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneOptionsBuilder) SetProjection(projection interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Projection = projection
		return nil
	})
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOneOptionsBuilder) SetReturnKey(b bool) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.ReturnKey = &b
		return nil
	})
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOneOptionsBuilder) SetShowRecordID(b bool) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.ShowRecordID = &b
		return nil
	})
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOneOptionsBuilder) SetSkip(i int64) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Skip = &i
		return nil
	})
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneOptionsBuilder) SetSort(sort interface{}) *FindOneOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneOptions) error {
		opts.Sort = sort
		return nil
	})
	return f
}

// FindOneAndReplaceOptions represents arguments that can be used to configure a
// FindOneAndReplace instance.
type FindOneAndReplaceOptions struct {
	// If true, writes executed as part of the operation will opt out of document-level validation on the server. This
	// option is valid for MongoDB versions >= 3.2 and is ignored for previous server versions. The default value is
	// false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about document
	// validation.
	BypassDocumentValidation *bool

	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// A document describing which fields will be included in the document returned by the operation. The default value
	// is nil, which means all fields will be included.
	Projection interface{}

	// Specifies whether the original or replaced document should be returned by the operation. The default value is
	// Before, which means the original document will be returned from before the replacement is performed.
	ReturnDocument *ReturnDocument

	// A document specifying which document should be replaced if the filter used by the operation matches multiple
	// documents in the collection. If set, the first document in the sorted order will be replaced. The driver will
	// return an error if the sort parameter is a multi-key map. The default value is nil.
	Sort interface{}

	// If true, a new document will be inserted if the filter does not match any documents in the collection. The
	// default value is false.
	Upsert *bool

	// The index to use for the operation. This should either be the index name as a string or the index specification
	// as a document. This option is only valid for MongoDB versions >= 4.4. MongoDB version 4.2 will report an error if
	// this option is specified. For server versions < 4.2, the driver will return an error if this option is specified.
	// The driver will return an error if this option is used with during an unacknowledged write operation. The driver
	// will return an error if the hint parameter is a multi-key map. The default value is nil, which means that no hint
	// will be sent.
	Hint interface{}

	// Specifies parameters for the find one and replace expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// FindOneAndReplaceOptionsBuilder contains options to perform a findAndModify
// operation. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type FindOneAndReplaceOptionsBuilder struct {
	Opts []func(*FindOneAndReplaceOptions) error
}

// FindOneAndReplace creates a new FindOneAndReplaceOptions instance.
func FindOneAndReplace() *FindOneAndReplaceOptionsBuilder {
	return &FindOneAndReplaceOptionsBuilder{}
}

// List returns a list of FindOneAndReplaceOptions setter functions.
func (f *FindOneAndReplaceOptionsBuilder) List() []func(*FindOneAndReplaceOptions) error {
	return f.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndReplaceOptionsBuilder) SetBypassDocumentValidation(b bool) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.BypassDocumentValidation = &b

		return nil
	})

	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndReplaceOptionsBuilder) SetCollation(collation *Collation) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndReplaceOptionsBuilder) SetComment(comment interface{}) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Comment = comment

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndReplaceOptionsBuilder) SetProjection(projection interface{}) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Projection = projection

		return nil
	})

	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndReplaceOptionsBuilder) SetReturnDocument(rd ReturnDocument) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.ReturnDocument = &rd

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndReplaceOptionsBuilder) SetSort(sort interface{}) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Sort = sort

		return nil
	})

	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndReplaceOptionsBuilder) SetUpsert(b bool) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Upsert = &b

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndReplaceOptionsBuilder) SetHint(hint interface{}) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndReplaceOptionsBuilder) SetLet(let interface{}) *FindOneAndReplaceOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndReplaceOptions) error {
		opts.Let = let

		return nil
	})

	return f
}

// FindOneAndUpdateOptions represents arguments that can be used to configure a
// FindOneAndUpdate options.
type FindOneAndUpdateOptions struct {
	// A set of filters specifying to which array elements an update should apply. This option is only valid for MongoDB
	// versions >= 3.6. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the update will apply to all array elements.
	ArrayFilters []interface{}

	// If true, writes executed as part of the operation will opt out of document-level validation on the server. This
	// option is valid for MongoDB versions >= 3.2 and is ignored for previous server versions. The default value is
	// false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about document
	// validation.
	BypassDocumentValidation *bool

	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// A document describing which fields will be included in the document returned by the operation. The default value
	// is nil, which means all fields will be included.
	Projection interface{}

	// Specifies whether the original or replaced document should be returned by the operation. The default value is
	// Before, which means the original document will be returned before the replacement is performed.
	ReturnDocument *ReturnDocument

	// A document specifying which document should be updated if the filter used by the operation matches multiple
	// documents in the collection. If set, the first document in the sorted order will be updated. The driver will
	// return an error if the sort parameter is a multi-key map. The default value is nil.
	Sort interface{}

	// If true, a new document will be inserted if the filter does not match any documents in the collection. The
	// default value is false.
	Upsert *bool

	// The index to use for the operation. This should either be the index name as a string or the index specification
	// as a document. This option is only valid for MongoDB versions >= 4.4. MongoDB version 4.2 will report an error if
	// this option is specified. For server versions < 4.2, the driver will return an error if this option is specified.
	// The driver will return an error if this option is used with during an unacknowledged write operation. The driver
	// will return an error if the hint parameter is a multi-key map. The default value is nil, which means that no hint
	// will be sent.
	Hint interface{}

	// Specifies parameters for the find one and update expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// FindOneAndUpdateOptionsBuilder contains options to configure a
// findOneAndUpdate operation. Each option can be set through setter functions.
// See documentation for each setter function for an explanation of the option.
type FindOneAndUpdateOptionsBuilder struct {
	Opts []func(*FindOneAndUpdateOptions) error
}

// FindOneAndUpdate creates a new FindOneAndUpdateOptions instance.
func FindOneAndUpdate() *FindOneAndUpdateOptionsBuilder {
	return &FindOneAndUpdateOptionsBuilder{}
}

// List returns a list of FindOneAndUpdateOptions setter functions.
func (f *FindOneAndUpdateOptionsBuilder) List() []func(*FindOneAndUpdateOptions) error {
	return f.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndUpdateOptionsBuilder) SetBypassDocumentValidation(b bool) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.BypassDocumentValidation = &b

		return nil
	})

	return f
}

// SetArrayFilters sets the value for the ArrayFilters field.
func (f *FindOneAndUpdateOptionsBuilder) SetArrayFilters(filters []interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.ArrayFilters = filters

		return nil
	})

	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndUpdateOptionsBuilder) SetCollation(collation *Collation) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndUpdateOptionsBuilder) SetComment(comment interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Comment = comment

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndUpdateOptionsBuilder) SetProjection(projection interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Projection = projection

		return nil
	})

	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndUpdateOptionsBuilder) SetReturnDocument(rd ReturnDocument) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.ReturnDocument = &rd

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndUpdateOptionsBuilder) SetSort(sort interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Sort = sort

		return nil
	})

	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndUpdateOptionsBuilder) SetUpsert(b bool) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Upsert = &b

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndUpdateOptionsBuilder) SetHint(hint interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndUpdateOptionsBuilder) SetLet(let interface{}) *FindOneAndUpdateOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndUpdateOptions) error {
		opts.Let = let

		return nil
	})

	return f
}

// FindOneAndDeleteOptions represents arguments that can be used to configure a
// FindOneAndDelete operation.
type FindOneAndDeleteOptions struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// A document describing which fields will be included in the document returned by the operation. The default value
	// is nil, which means all fields will be included.
	Projection interface{}

	// A document specifying which document should be replaced if the filter used by the operation matches multiple
	// documents in the collection. If set, the first document in the sorted order will be selected for replacement.
	// The driver will return an error if the sort parameter is a multi-key map. The default value is nil.
	Sort interface{}

	// The index to use for the operation. This should either be the index name as a string or the index specification
	// as a document. This option is only valid for MongoDB versions >= 4.4. MongoDB version 4.2 will report an error if
	// this option is specified. For server versions < 4.2, the driver will return an error if this option is specified.
	// The driver will return an error if this option is used with during an unacknowledged write operation. The driver
	// will return an error if the hint parameter is a multi-key map. The default value is nil, which means that no hint
	// will be sent.
	Hint interface{}

	// Specifies parameters for the find one and delete expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// FindOneAndDeleteOptionsBuilder contains options to configure delete
// operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type FindOneAndDeleteOptionsBuilder struct {
	Opts []func(*FindOneAndDeleteOptions) error
}

// FindOneAndDelete creates a new FindOneAndDeleteOptions instance.
func FindOneAndDelete() *FindOneAndDeleteOptionsBuilder {
	return &FindOneAndDeleteOptionsBuilder{}
}

// List returns a list of FindOneAndDeleteOptions setter functions.
func (f *FindOneAndDeleteOptionsBuilder) List() []func(*FindOneAndDeleteOptions) error {
	return f.Opts
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndDeleteOptionsBuilder) SetCollation(collation *Collation) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndDeleteOptionsBuilder) SetComment(comment interface{}) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Comment = comment

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndDeleteOptionsBuilder) SetProjection(projection interface{}) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Projection = projection

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndDeleteOptionsBuilder) SetSort(sort interface{}) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Sort = sort

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndDeleteOptionsBuilder) SetHint(hint interface{}) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndDeleteOptionsBuilder) SetLet(let interface{}) *FindOneAndDeleteOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *FindOneAndDeleteOptions) error {
		opts.Let = let

		return nil
	})

	return f
}
