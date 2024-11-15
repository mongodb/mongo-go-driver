// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ClientWriteModels is a struct that can be used in a client-level BulkWrite operation.
type ClientWriteModels struct {
	models []clientWriteModel
}
type clientWriteModel struct {
	namespace string
	model     interface{}
}

// AppendInsertOne appends ClientInsertOneModels.
func (m *ClientWriteModels) AppendInsertOne(database, collection string, models ...*ClientInsertOneModel) *ClientWriteModels {
	if m == nil {
		m = &ClientWriteModels{}
	}
	for _, model := range models {
		m.models = append(m.models, clientWriteModel{
			namespace: fmt.Sprintf("%s.%s", database, collection),
			model:     model,
		})
	}
	return m
}

// appendModels is a helper function to append models to ClientWriteModels.
func appendModels[T ClientUpdateOneModel |
	ClientUpdateManyModel |
	ClientReplaceOneModel |
	ClientDeleteOneModel |
	ClientDeleteManyModel](
	m *ClientWriteModels, database, collection string, models []*T) *ClientWriteModels {
	if m == nil {
		m = &ClientWriteModels{}
	}
	for _, model := range models {
		m.models = append(m.models, clientWriteModel{
			namespace: fmt.Sprintf("%s.%s", database, collection),
			model:     model,
		})
	}
	return m
}

// AppendUpdateOne appends ClientUpdateOneModels.
func (m *ClientWriteModels) AppendUpdateOne(database, collection string, models ...*ClientUpdateOneModel) *ClientWriteModels {
	return appendModels(m, database, collection, models)
}

// AppendUpdateMany appends ClientUpdateManyModels.
func (m *ClientWriteModels) AppendUpdateMany(database, collection string, models ...*ClientUpdateManyModel) *ClientWriteModels {
	return appendModels(m, database, collection, models)
}

// AppendReplaceOne appends ClientReplaceOneModels.
func (m *ClientWriteModels) AppendReplaceOne(database, collection string, models ...*ClientReplaceOneModel) *ClientWriteModels {
	return appendModels(m, database, collection, models)
}

// AppendDeleteOne appends ClientDeleteOneModels.
func (m *ClientWriteModels) AppendDeleteOne(database, collection string, models ...*ClientDeleteOneModel) *ClientWriteModels {
	return appendModels(m, database, collection, models)
}

// AppendDeleteMany appends ClientDeleteManyModels.
func (m *ClientWriteModels) AppendDeleteMany(database, collection string, models ...*ClientDeleteManyModel) *ClientWriteModels {
	return appendModels(m, database, collection, models)
}

// ClientInsertOneModel is used to insert a single document in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientInsertOneModel struct {
	Document interface{}
}

// NewClientInsertOneModel creates a new ClientInsertOneModel.
func NewClientInsertOneModel() *ClientInsertOneModel {
	return &ClientInsertOneModel{}
}

// SetDocument specifies the document to be inserted. The document cannot be nil. If it does not have an _id field when
// transformed into BSON, one will be added automatically to the marshalled document. The original document will not be
// modified.
func (iom *ClientInsertOneModel) SetDocument(doc interface{}) *ClientInsertOneModel {
	iom.Document = doc
	return iom
}

// ClientUpdateOneModel is used to update at most one document in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientUpdateOneModel struct {
	Collation    *options.Collation
	Upsert       *bool
	Filter       interface{}
	Update       interface{}
	ArrayFilters []interface{}
	Hint         interface{}
}

// NewClientUpdateOneModel creates a new ClientUpdateOneModel.
func NewClientUpdateOneModel() *ClientUpdateOneModel {
	return &ClientUpdateOneModel{}
}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (uom *ClientUpdateOneModel) SetHint(hint interface{}) *ClientUpdateOneModel {
	uom.Hint = hint
	return uom
}

// SetFilter specifies a filter to use to select the document to update. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (uom *ClientUpdateOneModel) SetFilter(filter interface{}) *ClientUpdateOneModel {
	uom.Filter = filter
	return uom
}

// SetUpdate specifies the modifications to be made to the selected document. The value must be a document containing
// update operators (https://www.mongodb.com/docs/manual/reference/operator/update/). It cannot be nil or empty.
func (uom *ClientUpdateOneModel) SetUpdate(update interface{}) *ClientUpdateOneModel {
	uom.Update = update
	return uom
}

// SetArrayFilters specifies a set of filters to determine which elements should be modified when updating an array
// field.
func (uom *ClientUpdateOneModel) SetArrayFilters(filters []interface{}) *ClientUpdateOneModel {
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

// ClientUpdateManyModel is used to update multiple documents in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientUpdateManyModel struct {
	Collation    *options.Collation
	Upsert       *bool
	Filter       interface{}
	Update       interface{}
	ArrayFilters []interface{}
	Hint         interface{}
}

// NewClientUpdateManyModel creates a new ClientUpdateManyModel.
func NewClientUpdateManyModel() *ClientUpdateManyModel {
	return &ClientUpdateManyModel{}
}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (umm *ClientUpdateManyModel) SetHint(hint interface{}) *ClientUpdateManyModel {
	umm.Hint = hint
	return umm
}

// SetFilter specifies a filter to use to select documents to update. The filter must be a document containing query
// operators. It cannot be nil.
func (umm *ClientUpdateManyModel) SetFilter(filter interface{}) *ClientUpdateManyModel {
	umm.Filter = filter
	return umm
}

// SetUpdate specifies the modifications to be made to the selected documents. The value must be a document containing
// update operators (https://www.mongodb.com/docs/manual/reference/operator/update/). It cannot be nil or empty.
func (umm *ClientUpdateManyModel) SetUpdate(update interface{}) *ClientUpdateManyModel {
	umm.Update = update
	return umm
}

// SetArrayFilters specifies a set of filters to determine which elements should be modified when updating an array
// field.
func (umm *ClientUpdateManyModel) SetArrayFilters(filters []interface{}) *ClientUpdateManyModel {
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
	Filter      interface{}
	Replacement interface{}
	Hint        interface{}
}

// NewClientReplaceOneModel creates a new ClientReplaceOneModel.
func NewClientReplaceOneModel() *ClientReplaceOneModel {
	return &ClientReplaceOneModel{}
}

// SetHint specifies the index to use for the operation. This should either be the index name as a string or the index
// specification as a document. The default value is nil, which means that no hint will be sent.
func (rom *ClientReplaceOneModel) SetHint(hint interface{}) *ClientReplaceOneModel {
	rom.Hint = hint
	return rom
}

// SetFilter specifies a filter to use to select the document to replace. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (rom *ClientReplaceOneModel) SetFilter(filter interface{}) *ClientReplaceOneModel {
	rom.Filter = filter
	return rom
}

// SetReplacement specifies a document that will be used to replace the selected document. It cannot be nil and cannot
// contain any update operators (https://www.mongodb.com/docs/manual/reference/operator/update/).
func (rom *ClientReplaceOneModel) SetReplacement(rep interface{}) *ClientReplaceOneModel {
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

// ClientDeleteOneModel is used to delete at most one document in a client-level BulkWriteOperation.
//
// See corresponding setter methods for documentation.
type ClientDeleteOneModel struct {
	Filter    interface{}
	Collation *options.Collation
	Hint      interface{}
}

// NewClientDeleteOneModel creates a new ClientDeleteOneModel.
func NewClientDeleteOneModel() *ClientDeleteOneModel {
	return &ClientDeleteOneModel{}
}

// SetFilter specifies a filter to use to select the document to delete. The filter must be a document containing query
// operators. It cannot be nil. If the filter matches multiple documents, one will be selected from the matching
// documents.
func (dom *ClientDeleteOneModel) SetFilter(filter interface{}) *ClientDeleteOneModel {
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
func (dom *ClientDeleteOneModel) SetHint(hint interface{}) *ClientDeleteOneModel {
	dom.Hint = hint
	return dom
}

// ClientDeleteManyModel is used to delete multiple documents in a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientDeleteManyModel struct {
	Filter    interface{}
	Collation *options.Collation
	Hint      interface{}
}

// NewClientDeleteManyModel creates a new ClientDeleteManyModel.
func NewClientDeleteManyModel() *ClientDeleteManyModel {
	return &ClientDeleteManyModel{}
}

// SetFilter specifies a filter to use to select documents to delete. The filter must be a document containing query
// operators. It cannot be nil.
func (dmm *ClientDeleteManyModel) SetFilter(filter interface{}) *ClientDeleteManyModel {
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
func (dmm *ClientDeleteManyModel) SetHint(hint interface{}) *ClientDeleteManyModel {
	dmm.Hint = hint
	return dmm
}
