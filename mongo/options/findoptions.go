// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"
)

// FindArgs represents arguments that can be used to configure a Find operation.
type FindArgs struct {
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

	// MaxTime is the maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used in its
	// place to control the amount of time that a single operation can run before returning an error. MaxTime is ignored if
	// Timeout is set on the client.
	MaxTime *time.Duration

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

	// The above are in common with FindOneArgs.

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

	// MaxAwaitTime is the maximum amount of time that the server should wait for new documents to satisfy a tailable cursor
	// query. This option is only valid for tailable await cursors (see the CursorType option for more information) and
	// MongoDB versions >= 3.2. For other cursor types or previous server versions, this option is ignored.
	MaxAwaitTime *time.Duration

	// NoCursorTimeout specifies whether the cursor created by the operation will not timeout after a period of inactivity.
	// The default value is false.
	NoCursorTimeout *bool
}

// FindOptions represents functional options that configure an FindArgs.
type FindOptions struct {
	Opts []func(*FindArgs) error
}

// Find creates a new FindOptions instance.
func Find() *FindOptions {
	return &FindOptions{}
}

// ArgsSetters returns a list of FindArgs setter functions.
func (f *FindOptions) ArgsSetters() []func(*FindArgs) error {
	return f.Opts
}

// SetAllowDiskUse sets the value for the AllowDiskUse field.
func (f *FindOptions) SetAllowDiskUse(b bool) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.AllowDiskUse = &b
		return nil
	})
	return f
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOptions) SetAllowPartialResults(b bool) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.AllowPartialResults = &b
		return nil
	})
	return f
}

// SetBatchSize sets the value for the BatchSize field.
func (f *FindOptions) SetBatchSize(i int32) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.BatchSize = &i
		return nil
	})
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOptions) SetCollation(collation *Collation) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Collation = collation
		return nil
	})
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOptions) SetComment(comment interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Comment = &comment
		return nil
	})
	return f
}

// SetCursorType sets the value for the CursorType field.
func (f *FindOptions) SetCursorType(ct CursorType) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.CursorType = &ct
		return nil
	})
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOptions) SetHint(hint interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Hint = hint
		return nil
	})
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOptions) SetLet(let interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Let = let
		return nil
	})
	return f
}

// SetLimit sets the value for the Limit field.
func (f *FindOptions) SetLimit(i int64) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Limit = &i
		return nil
	})
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOptions) SetMax(max interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Max = max
		return nil
	})
	return f
}

// SetMaxAwaitTime sets the value for the MaxAwaitTime field.
func (f *FindOptions) SetMaxAwaitTime(d time.Duration) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.MaxAwaitTime = &d
		return nil
	})
	return f
}

// SetMaxTime specifies the max time to allow the query to run.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used used in its place to control the amount of time that a single operation
// can run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *FindOptions) SetMaxTime(d time.Duration) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.MaxTime = &d
		return nil
	})
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOptions) SetMin(min interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Min = min
		return nil
	})
	return f
}

// SetNoCursorTimeout sets the value for the NoCursorTimeout field.
func (f *FindOptions) SetNoCursorTimeout(b bool) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.NoCursorTimeout = &b
		return nil
	})
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOptions) SetProjection(projection interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Projection = projection
		return nil
	})
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOptions) SetReturnKey(b bool) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.ReturnKey = &b
		return nil
	})
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOptions) SetShowRecordID(b bool) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.ShowRecordID = &b
		return nil
	})
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOptions) SetSkip(i int64) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Skip = &i
		return nil
	})
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOptions) SetSort(sort interface{}) *FindOptions {
	f.Opts = append(f.Opts, func(args *FindArgs) error {
		args.Sort = sort
		return nil
	})
	return f
}

// FindOneArgs represents arguments that can be used to configure a FindOne operation.
type FindOneArgs struct {
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

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime
	// is ignored if Timeout is set on the client.
	MaxTime *time.Duration

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

// FindOneOptions represents functional options that configure an FindOneArgs.
type FindOneOptions struct {
	Opts []func(*FindOneArgs) error
}

// FindOne creates a new FindOneOptions instance.
func FindOne() *FindOneOptions {
	return &FindOneOptions{}
}

// ArgsSetters returns a list of FindOneArgs setter functions.
func (f *FindOneOptions) ArgsSetters() []func(*FindOneArgs) error {
	return f.Opts
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOneOptions) SetAllowPartialResults(b bool) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.AllowPartialResults = &b
		return nil
	})
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneOptions) SetCollation(collation *Collation) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Collation = collation
		return nil
	})
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneOptions) SetComment(comment interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Comment = &comment
		return nil
	})
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneOptions) SetHint(hint interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Hint = hint
		return nil
	})
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOneOptions) SetMax(max interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Max = max
		return nil
	})
	return f
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can
// run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *FindOneOptions) SetMaxTime(d time.Duration) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.MaxTime = &d
		return nil
	})
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOneOptions) SetMin(min interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Min = min
		return nil
	})
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneOptions) SetProjection(projection interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Projection = projection
		return nil
	})
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOneOptions) SetReturnKey(b bool) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.ReturnKey = &b
		return nil
	})
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOneOptions) SetShowRecordID(b bool) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.ShowRecordID = &b
		return nil
	})
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOneOptions) SetSkip(i int64) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Skip = &i
		return nil
	})
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneOptions) SetSort(sort interface{}) *FindOneOptions {
	f.Opts = append(f.Opts, func(args *FindOneArgs) error {
		args.Sort = sort
		return nil
	})
	return f
}

// FindOneAndReplaceArgs represents arguments that can be used to configure a
// FindOneAndReplace instance.
type FindOneAndReplaceArgs struct {
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

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime
	// is ignored if Timeout is set on the client.
	MaxTime *time.Duration

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

// FindOneAndReplaceOptions contains options to perform a findAndModify
// operation. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type FindOneAndReplaceOptions struct {
	Opts []func(*FindOneAndReplaceArgs) error
}

// FindOneAndReplace creates a new FindOneAndReplaceOptions instance.
func FindOneAndReplace() *FindOneAndReplaceOptions {
	return &FindOneAndReplaceOptions{}
}

// ArgsSetters returns a list of FindOneAndReplaceArgs setter functions.
func (f *FindOneAndReplaceOptions) ArgsSetters() []func(*FindOneAndReplaceArgs) error {
	return f.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndReplaceOptions) SetBypassDocumentValidation(b bool) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.BypassDocumentValidation = &b

		return nil
	})

	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndReplaceOptions) SetCollation(collation *Collation) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndReplaceOptions) SetComment(comment interface{}) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Comment = comment

		return nil
	})

	return f
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can
// run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *FindOneAndReplaceOptions) SetMaxTime(d time.Duration) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.MaxTime = &d

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndReplaceOptions) SetProjection(projection interface{}) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Projection = projection

		return nil
	})

	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndReplaceOptions) SetReturnDocument(rd ReturnDocument) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.ReturnDocument = &rd

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndReplaceOptions) SetSort(sort interface{}) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Sort = sort

		return nil
	})

	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndReplaceOptions) SetUpsert(b bool) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Upsert = &b

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndReplaceOptions) SetHint(hint interface{}) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndReplaceOptions) SetLet(let interface{}) *FindOneAndReplaceOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndReplaceArgs) error {
		args.Let = let

		return nil
	})

	return f
}

// FindOneAndUpdateArgs represents arguments that can be used to configure a
// FindOneAndUpdate options.
type FindOneAndUpdateArgs struct {
	// A set of filters specifying to which array elements an update should apply. This option is only valid for MongoDB
	// versions >= 3.6. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the update will apply to all array elements.
	ArrayFilters *ArrayFilters

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

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime is
	// ignored if Timeout is set on the client.
	MaxTime *time.Duration

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

// FindOneAndUpdateOptions contains options to configure a findOneAndUpdate
// operation. Each option can be set through setter functions. See documentation
// for each setter function for an explanation of the option.
type FindOneAndUpdateOptions struct {
	Opts []func(*FindOneAndUpdateArgs) error
}

// FindOneAndUpdate creates a new FindOneAndUpdateOptions instance.
func FindOneAndUpdate() *FindOneAndUpdateOptions {
	return &FindOneAndUpdateOptions{}
}

// ArgsSetters returns a list of FindOneAndUpdateArgs setter functions.
func (f *FindOneAndUpdateOptions) ArgsSetters() []func(*FindOneAndUpdateArgs) error {
	return f.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndUpdateOptions) SetBypassDocumentValidation(b bool) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.BypassDocumentValidation = &b

		return nil
	})

	return f
}

// SetArrayFilters sets the value for the ArrayFilters field.
func (f *FindOneAndUpdateOptions) SetArrayFilters(filters ArrayFilters) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.ArrayFilters = &filters

		return nil
	})

	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndUpdateOptions) SetCollation(collation *Collation) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndUpdateOptions) SetComment(comment interface{}) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Comment = comment

		return nil
	})

	return f
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can
// run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *FindOneAndUpdateOptions) SetMaxTime(d time.Duration) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.MaxTime = &d

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndUpdateOptions) SetProjection(projection interface{}) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Projection = projection

		return nil
	})

	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndUpdateOptions) SetReturnDocument(rd ReturnDocument) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.ReturnDocument = &rd

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndUpdateOptions) SetSort(sort interface{}) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Sort = sort

		return nil
	})

	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndUpdateOptions) SetUpsert(b bool) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Upsert = &b

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndUpdateOptions) SetHint(hint interface{}) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndUpdateOptions) SetLet(let interface{}) *FindOneAndUpdateOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndUpdateArgs) error {
		args.Let = let

		return nil
	})

	return f
}

// FindOneAndDeleteArgs represents arguments that can be used to configure a
// FindOneAndDelete operation.
type FindOneAndDeleteArgs struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime
	// is ignored if Timeout is set on the client.
	MaxTime *time.Duration

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

// FindOneAndDeleteOptions contains options to configure delete operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type FindOneAndDeleteOptions struct {
	Opts []func(*FindOneAndDeleteArgs) error
}

// FindOneAndDelete creates a new FindOneAndDeleteOptions instance.
func FindOneAndDelete() *FindOneAndDeleteOptions {
	return &FindOneAndDeleteOptions{}
}

// ArgsSetters returns a list of FindOneAndDeleteArgs setter functions.
func (f *FindOneAndDeleteOptions) ArgsSetters() []func(*FindOneAndDeleteArgs) error {
	return f.Opts
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndDeleteOptions) SetCollation(collation *Collation) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Collation = collation

		return nil
	})

	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndDeleteOptions) SetComment(comment interface{}) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Comment = comment

		return nil
	})

	return f
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can
// run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *FindOneAndDeleteOptions) SetMaxTime(d time.Duration) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.MaxTime = &d

		return nil
	})

	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndDeleteOptions) SetProjection(projection interface{}) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Projection = projection

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndDeleteOptions) SetSort(sort interface{}) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Sort = sort

		return nil
	})

	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndDeleteOptions) SetHint(hint interface{}) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Hint = hint

		return nil
	})

	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndDeleteOptions) SetLet(let interface{}) *FindOneAndDeleteOptions {
	f.Opts = append(f.Opts, func(args *FindOneAndDeleteArgs) error {
		args.Let = let

		return nil
	})

	return f
}
