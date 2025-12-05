// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ClientWriteModel is an interface implemented by models that can be used in a client-level BulkWrite operation. Each
// ClientWriteModel represents a write.
//
// This interface is implemented by ClientDeleteOneModel, ClientDeleteManyModel, ClientInsertOneModel,
// ClientReplaceOneModel, ClientUpdateOneModel, and ClientUpdateManyModel. Custom implementations of this interface must
// not be used.
type ClientWriteModel interface {
	clientWriteModel()
}

// ClientInsertOneModel is used to insert a single document in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientInsertOneModel struct {
	Document any
}

// NewClientInsertOneModel creates a new ClientInsertOneModel.
func NewClientInsertOneModel() *ClientInsertOneModel {
	return &ClientInsertOneModel{}
}

func (*ClientInsertOneModel) clientWriteModel() {}

// SetDocument specifies the document to be inserted. The document cannot be nil. If it does not have an _id field when
// transformed into BSON, one will be added automatically to the marshalled document. The original document will not be
// modified.
func (iom *ClientInsertOneModel) SetDocument(doc any) *ClientInsertOneModel {
	iom.Document = doc
	return iom
}

// ClientUpdateOneModel is used to update at most one document in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientUpdateOneModel struct {
	Collation    *options.Collation
	Upsert       *bool
	Filter       any
	Update       any
	ArrayFilters []any
	Hint         any
	Sort         any
}

// NewClientUpdateOneModel creates a new ClientUpdateOneModel.
func NewClientUpdateOneModel() *ClientUpdateOneModel {
	return &ClientUpdateOneModel{}
}

func (*ClientUpdateOneModel) clientWriteModel() {}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (uom *ClientUpdateOneModel) SetHint(hint any) *ClientUpdateOneModel {
	uom.Hint = hint
	return uom
}

// SetFilter specifies a filter to use to select the document to update. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (uom *ClientUpdateOneModel) SetFilter(filter any) *ClientUpdateOneModel {
	uom.Filter = filter
	return uom
}

// SetUpdate specifies the modifications to be made to the selected document. The value must be a document containing
// update operators (https://www.mongodb.com/docs/manual/reference/operator/update/). It cannot be nil or empty.
func (uom *ClientUpdateOneModel) SetUpdate(update any) *ClientUpdateOneModel {
	uom.Update = update
	return uom
}

// SetArrayFilters specifies a set of filters to determine which elements should be modified when updating an array
// field.
func (uom *ClientUpdateOneModel) SetArrayFilters(filters []any) *ClientUpdateOneModel {
	uom.ArrayFilters = filters
	return uom
}

// SetCollation specifies a collation to use for string comparisons. The default is nil, meaning no collation will be
// used.
func (uom *ClientUpdateOneModel) SetCollation(collation *options.Collation) *ClientUpdateOneModel {
	uom.Collation = collation
	return uom
}

// SetUpsert specifies whether or not a new document should be inserted if no document matching the filter is found. If
// an upsert is performed, the _id of the upserted document can be retrieved from the UpdateResults field of the
// ClientBulkWriteResult.
func (uom *ClientUpdateOneModel) SetUpsert(upsert bool) *ClientUpdateOneModel {
	uom.Upsert = &upsert
	return uom
}

// SetSort specifies which document the operation updates if the query matches multiple documents. The first document
// matched by the sort order will be updated. This option is only valid for MongoDB versions >= 8.0. The sort parameter
// is evaluated sequentially, so the driver will return an error if it is a multi-key map (which is unordeded). The
// default value is nil.
func (uom *ClientUpdateOneModel) SetSort(sort any) *ClientUpdateOneModel {
	uom.Sort = sort
	return uom
}

// ClientUpdateManyModel is used to update multiple documents in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientUpdateManyModel struct {
	Collation    *options.Collation
	Upsert       *bool
	Filter       any
	Update       any
	ArrayFilters []any
	Hint         any
}

// NewClientUpdateManyModel creates a new ClientUpdateManyModel.
func NewClientUpdateManyModel() *ClientUpdateManyModel {
	return &ClientUpdateManyModel{}
}

func (*ClientUpdateManyModel) clientWriteModel() {}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (umm *ClientUpdateManyModel) SetHint(hint any) *ClientUpdateManyModel {
	umm.Hint = hint
	return umm
}

// SetFilter specifies a filter to use to select documents to update. The filter must be a document containing query
// operators. It cannot be nil.
func (umm *ClientUpdateManyModel) SetFilter(filter any) *ClientUpdateManyModel {
	umm.Filter = filter
	return umm
}

// SetUpdate specifies the modifications to be made to the selected documents. The value must be a document containing
// update operators (https://www.mongodb.com/docs/manual/reference/operator/update/). It cannot be nil or empty.
func (umm *ClientUpdateManyModel) SetUpdate(update any) *ClientUpdateManyModel {
	umm.Update = update
	return umm
}

// SetArrayFilters specifies a set of filters to determine which elements should be modified when updating an array
// field.
func (umm *ClientUpdateManyModel) SetArrayFilters(filters []any) *ClientUpdateManyModel {
	umm.ArrayFilters = filters
	return umm
}

// SetCollation specifies a collation to use for string comparisons. The default is nil, meaning no collation will be
// used.
func (umm *ClientUpdateManyModel) SetCollation(collation *options.Collation) *ClientUpdateManyModel {
	umm.Collation = collation
	return umm
}

// SetUpsert specifies whether or not a new document should be inserted if no document matching the filter is found. If
// an upsert is performed, the _id of the upserted document can be retrieved from the UpdateResults field of the
// ClientBulkWriteResult.
func (umm *ClientUpdateManyModel) SetUpsert(upsert bool) *ClientUpdateManyModel {
	umm.Upsert = &upsert
	return umm
}

// ClientReplaceOneModel is used to replace at most one document in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientReplaceOneModel struct {
	Collation   *options.Collation
	Upsert      *bool
	Filter      any
	Replacement any
	Hint        any
	Sort        any
}

// NewClientReplaceOneModel creates a new ClientReplaceOneModel.
func NewClientReplaceOneModel() *ClientReplaceOneModel {
	return &ClientReplaceOneModel{}
}

func (*ClientReplaceOneModel) clientWriteModel() {}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (rom *ClientReplaceOneModel) SetHint(hint any) *ClientReplaceOneModel {
	rom.Hint = hint
	return rom
}

// SetFilter specifies a filter to use to select the document to replace. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (rom *ClientReplaceOneModel) SetFilter(filter any) *ClientReplaceOneModel {
	rom.Filter = filter
	return rom
}

// SetReplacement specifies a document that will be used to replace the selected document. It cannot be nil and cannot
// contain any update operators (https://www.mongodb.com/docs/manual/reference/operator/update/).
func (rom *ClientReplaceOneModel) SetReplacement(rep any) *ClientReplaceOneModel {
	rom.Replacement = rep
	return rom
}

// SetCollation specifies a collation to use for string comparisons. The default is nil, meaning no collation will be
// used.
func (rom *ClientReplaceOneModel) SetCollation(collation *options.Collation) *ClientReplaceOneModel {
	rom.Collation = collation
	return rom
}

// SetUpsert specifies whether or not the replacement document should be inserted if no document matching the filter is
// found. If an upsert is performed, the _id of the upserted document can be retrieved from the UpdateResults field of the
// BulkWriteResult.
func (rom *ClientReplaceOneModel) SetUpsert(upsert bool) *ClientReplaceOneModel {
	rom.Upsert = &upsert
	return rom
}

// SetSort specifies which document the operation replaces if the query matches multiple documents. The first document
// matched by the sort order will be replaced. This option is only valid for MongoDB versions >= 8.0. The sort parameter
// is evaluated sequentially, so the driver will return an error if it is a multi-key map (which is unordeded). The
// default value is nil.
func (rom *ClientReplaceOneModel) SetSort(sort any) *ClientReplaceOneModel {
	rom.Sort = sort
	return rom
}

// ClientDeleteOneModel is used to delete at most one document in a client-level BulkWriteOperation.
//
// See corresponding setter methods for documentation.
type ClientDeleteOneModel struct {
	Filter    any
	Collation *options.Collation
	Hint      any
}

// NewClientDeleteOneModel creates a new ClientDeleteOneModel.
func NewClientDeleteOneModel() *ClientDeleteOneModel {
	return &ClientDeleteOneModel{}
}

func (*ClientDeleteOneModel) clientWriteModel() {}

// SetFilter specifies a filter to use to select the document to delete. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (dom *ClientDeleteOneModel) SetFilter(filter any) *ClientDeleteOneModel {
	dom.Filter = filter
	return dom
}

// SetCollation specifies a collation to use for string comparisons. The default is nil, meaning no collation will be
// used.
func (dom *ClientDeleteOneModel) SetCollation(collation *options.Collation) *ClientDeleteOneModel {
	dom.Collation = collation
	return dom
}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (dom *ClientDeleteOneModel) SetHint(hint any) *ClientDeleteOneModel {
	dom.Hint = hint
	return dom
}

// ClientDeleteManyModel is used to delete multiple documents in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientDeleteManyModel struct {
	Filter    any
	Collation *options.Collation
	Hint      any
}

// NewClientDeleteManyModel creates a new ClientDeleteManyModel.
func NewClientDeleteManyModel() *ClientDeleteManyModel {
	return &ClientDeleteManyModel{}
}

func (*ClientDeleteManyModel) clientWriteModel() {}

// SetFilter specifies a filter to use to select documents to delete. The filter must be a document containing query
// operators. It cannot be nil.
func (dmm *ClientDeleteManyModel) SetFilter(filter any) *ClientDeleteManyModel {
	dmm.Filter = filter
	return dmm
}

// SetCollation specifies a collation to use for string comparisons. The default is nil, meaning no collation will be
// used.
func (dmm *ClientDeleteManyModel) SetCollation(collation *options.Collation) *ClientDeleteManyModel {
	dmm.Collation = collation
	return dmm
}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (dmm *ClientDeleteManyModel) SetHint(hint any) *ClientDeleteManyModel {
	dmm.Hint = hint
	return dmm
}
