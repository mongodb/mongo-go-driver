// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"
)

// FindOptions represents options that can be used to configure a Find operation.
type FindOptions struct {
	// AllowDiskUse specifies whether the server can write temporary data to disk while executing the Find operation.
	// This option is only valid for MongoDB versions >= 4.4. Server versions >= 3.2 will report an error if this option
	// is specified. For server versions < 3.2, the driver will return a client-side error if this option is specified.
	// The default value is false.
	AllowDiskUse *bool

	// AllowPartial results specifies whether the Find operation on a sharded cluster can return partial results if some
	// shards are down rather than returning an error. The default value is false.
	AllowPartialResults *bool

	// BatchSize is the maximum number of documents to be included in each batch returned by the server.
	BatchSize *int32

	// Collation specifies a collation to use for string comparisons during the operation. This option is only valid for
	// MongoDB versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs,
	// and currentOp queries to help trace the operation. The default is nil,
	// which means that no comment will be included in the logs.
	Comment interface{}

	// CursorType specifies the type of cursor that should be created for the operation. The default is NonTailable, which
	// means that the cursor will be closed by the server when the last batch of documents is retrieved.
	CursorType *CursorType

	// Hint is the index to use for the Find operation. This should either be the index name as a string or the index
	// specification as a document. The driver will return an error if the hint parameter is a multi-key map. The default
	// value is nil, which means that no hint will be sent.
	Hint interface{}

	// Limit is the maximum number of documents to return. The default value is 0, which means that all documents matching the
	// filter will be returned. A negative limit specifies that the resulting documents should be returned in a single
	// batch. The default value is 0.
	Limit *int64

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

	// NoCursorTimeout specifies whether the cursor created by the operation will not timeout after a period of inactivity.
	// The default value is false.
	NoCursorTimeout *bool

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

	// Let specifies parameters for the find expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// Find creates a new FindOptions instance.
func Find() *FindOptions {
	return &FindOptions{}
}

// SetAllowDiskUse sets the value for the AllowDiskUse field.
func (f *FindOptions) SetAllowDiskUse(b bool) *FindOptions {
	f.AllowDiskUse = &b
	return f
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOptions) SetAllowPartialResults(b bool) *FindOptions {
	f.AllowPartialResults = &b
	return f
}

// SetBatchSize sets the value for the BatchSize field.
func (f *FindOptions) SetBatchSize(i int32) *FindOptions {
	f.BatchSize = &i
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOptions) SetCollation(collation *Collation) *FindOptions {
	f.Collation = collation
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOptions) SetComment(comment interface{}) *FindOptions {
	f.Comment = comment
	return f
}

// SetCursorType sets the value for the CursorType field.
func (f *FindOptions) SetCursorType(ct CursorType) *FindOptions {
	f.CursorType = &ct
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOptions) SetHint(hint interface{}) *FindOptions {
	f.Hint = hint
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOptions) SetLet(let interface{}) *FindOptions {
	f.Let = let
	return f
}

// SetLimit sets the value for the Limit field.
func (f *FindOptions) SetLimit(i int64) *FindOptions {
	f.Limit = &i
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOptions) SetMax(max interface{}) *FindOptions {
	f.Max = max
	return f
}

// SetMaxAwaitTime sets the value for the MaxAwaitTime field.
func (f *FindOptions) SetMaxAwaitTime(d time.Duration) *FindOptions {
	f.MaxAwaitTime = &d
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOptions) SetMin(min interface{}) *FindOptions {
	f.Min = min
	return f
}

// SetNoCursorTimeout sets the value for the NoCursorTimeout field.
func (f *FindOptions) SetNoCursorTimeout(b bool) *FindOptions {
	f.NoCursorTimeout = &b
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOptions) SetProjection(projection interface{}) *FindOptions {
	f.Projection = projection
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOptions) SetReturnKey(b bool) *FindOptions {
	f.ReturnKey = &b
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOptions) SetShowRecordID(b bool) *FindOptions {
	f.ShowRecordID = &b
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOptions) SetSkip(i int64) *FindOptions {
	f.Skip = &i
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOptions) SetSort(sort interface{}) *FindOptions {
	f.Sort = sort
	return f
}

// FindOneOptions represents options that can be used to configure a FindOne operation.
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

// FindOne creates a new FindOneOptions instance.
func FindOne() *FindOneOptions {
	return &FindOneOptions{}
}

// SetAllowPartialResults sets the value for the AllowPartialResults field.
func (f *FindOneOptions) SetAllowPartialResults(b bool) *FindOneOptions {
	f.AllowPartialResults = &b
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneOptions) SetCollation(collation *Collation) *FindOneOptions {
	f.Collation = collation
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneOptions) SetComment(comment interface{}) *FindOneOptions {
	f.Comment = &comment
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneOptions) SetHint(hint interface{}) *FindOneOptions {
	f.Hint = hint
	return f
}

// SetMax sets the value for the Max field.
func (f *FindOneOptions) SetMax(max interface{}) *FindOneOptions {
	f.Max = max
	return f
}

// SetMin sets the value for the Min field.
func (f *FindOneOptions) SetMin(min interface{}) *FindOneOptions {
	f.Min = min
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneOptions) SetProjection(projection interface{}) *FindOneOptions {
	f.Projection = projection
	return f
}

// SetReturnKey sets the value for the ReturnKey field.
func (f *FindOneOptions) SetReturnKey(b bool) *FindOneOptions {
	f.ReturnKey = &b
	return f
}

// SetShowRecordID sets the value for the ShowRecordID field.
func (f *FindOneOptions) SetShowRecordID(b bool) *FindOneOptions {
	f.ShowRecordID = &b
	return f
}

// SetSkip sets the value for the Skip field.
func (f *FindOneOptions) SetSkip(i int64) *FindOneOptions {
	f.Skip = &i
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneOptions) SetSort(sort interface{}) *FindOneOptions {
	f.Sort = sort
	return f
}

// FindOneAndReplaceOptions represents options that can be used to configure a FindOneAndReplace instance.
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

// FindOneAndReplace creates a new FindOneAndReplaceOptions instance.
func FindOneAndReplace() *FindOneAndReplaceOptions {
	return &FindOneAndReplaceOptions{}
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndReplaceOptions) SetBypassDocumentValidation(b bool) *FindOneAndReplaceOptions {
	f.BypassDocumentValidation = &b
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndReplaceOptions) SetCollation(collation *Collation) *FindOneAndReplaceOptions {
	f.Collation = collation
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndReplaceOptions) SetComment(comment interface{}) *FindOneAndReplaceOptions {
	f.Comment = comment
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndReplaceOptions) SetProjection(projection interface{}) *FindOneAndReplaceOptions {
	f.Projection = projection
	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndReplaceOptions) SetReturnDocument(rd ReturnDocument) *FindOneAndReplaceOptions {
	f.ReturnDocument = &rd
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndReplaceOptions) SetSort(sort interface{}) *FindOneAndReplaceOptions {
	f.Sort = sort
	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndReplaceOptions) SetUpsert(b bool) *FindOneAndReplaceOptions {
	f.Upsert = &b
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndReplaceOptions) SetHint(hint interface{}) *FindOneAndReplaceOptions {
	f.Hint = hint
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndReplaceOptions) SetLet(let interface{}) *FindOneAndReplaceOptions {
	f.Let = let
	return f
}

// FindOneAndUpdateOptions represents options that can be used to configure a FindOneAndUpdate options.
type FindOneAndUpdateOptions struct {
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

// FindOneAndUpdate creates a new FindOneAndUpdateOptions instance.
func FindOneAndUpdate() *FindOneAndUpdateOptions {
	return &FindOneAndUpdateOptions{}
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (f *FindOneAndUpdateOptions) SetBypassDocumentValidation(b bool) *FindOneAndUpdateOptions {
	f.BypassDocumentValidation = &b
	return f
}

// SetArrayFilters sets the value for the ArrayFilters field.
func (f *FindOneAndUpdateOptions) SetArrayFilters(filters ArrayFilters) *FindOneAndUpdateOptions {
	f.ArrayFilters = &filters
	return f
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndUpdateOptions) SetCollation(collation *Collation) *FindOneAndUpdateOptions {
	f.Collation = collation
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndUpdateOptions) SetComment(comment interface{}) *FindOneAndUpdateOptions {
	f.Comment = comment
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndUpdateOptions) SetProjection(projection interface{}) *FindOneAndUpdateOptions {
	f.Projection = projection
	return f
}

// SetReturnDocument sets the value for the ReturnDocument field.
func (f *FindOneAndUpdateOptions) SetReturnDocument(rd ReturnDocument) *FindOneAndUpdateOptions {
	f.ReturnDocument = &rd
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndUpdateOptions) SetSort(sort interface{}) *FindOneAndUpdateOptions {
	f.Sort = sort
	return f
}

// SetUpsert sets the value for the Upsert field.
func (f *FindOneAndUpdateOptions) SetUpsert(b bool) *FindOneAndUpdateOptions {
	f.Upsert = &b
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndUpdateOptions) SetHint(hint interface{}) *FindOneAndUpdateOptions {
	f.Hint = hint
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndUpdateOptions) SetLet(let interface{}) *FindOneAndUpdateOptions {
	f.Let = let
	return f
}

// FindOneAndDeleteOptions represents options that can be used to configure a FindOneAndDelete operation.
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

// FindOneAndDelete creates a new FindOneAndDeleteOptions instance.
func FindOneAndDelete() *FindOneAndDeleteOptions {
	return &FindOneAndDeleteOptions{}
}

// SetCollation sets the value for the Collation field.
func (f *FindOneAndDeleteOptions) SetCollation(collation *Collation) *FindOneAndDeleteOptions {
	f.Collation = collation
	return f
}

// SetComment sets the value for the Comment field.
func (f *FindOneAndDeleteOptions) SetComment(comment interface{}) *FindOneAndDeleteOptions {
	f.Comment = comment
	return f
}

// SetProjection sets the value for the Projection field.
func (f *FindOneAndDeleteOptions) SetProjection(projection interface{}) *FindOneAndDeleteOptions {
	f.Projection = projection
	return f
}

// SetSort sets the value for the Sort field.
func (f *FindOneAndDeleteOptions) SetSort(sort interface{}) *FindOneAndDeleteOptions {
	f.Sort = sort
	return f
}

// SetHint sets the value for the Hint field.
func (f *FindOneAndDeleteOptions) SetHint(hint interface{}) *FindOneAndDeleteOptions {
	f.Hint = hint
	return f
}

// SetLet sets the value for the Let field.
func (f *FindOneAndDeleteOptions) SetLet(let interface{}) *FindOneAndDeleteOptions {
	f.Let = let
	return f
}
