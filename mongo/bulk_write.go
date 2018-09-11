package mongo

import (
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

// WriteModel is the interface satisfied by all models for bulk writes.
type WriteModel interface {
	ConvertModel() dispatch.WriteModel
}

// InsertOneModel is the write model for insert operations.
type InsertOneModel struct {
	dispatch.InsertOneModel
}

// Document sets the BSON document for the InsertOneModel.
func (iom *InsertOneModel) Document(doc interface{}) *InsertOneModel {
	iom.InsertOneModel.Document = doc
	return iom
}

// ConvertModel satisfies the WriteModel interface.
func (iom *InsertOneModel) ConvertModel() dispatch.WriteModel {
	return iom.InsertOneModel
}

// DeleteOneModel is the write model for delete operations.
type DeleteOneModel struct {
	dispatch.DeleteOneModel
}

// Filter sets the filter for the DeleteOneModel.
func (dom *DeleteOneModel) Filter(filter interface{}) *DeleteOneModel {
	dom.DeleteOneModel.Filter = filter
	return dom
}

// Collation sets the collation for the DeleteOneModel.
func (dom *DeleteOneModel) Collation(collation *mongoopt.Collation) *DeleteOneModel {
	dom.DeleteOneModel.Collation = collation.Convert()
	dom.DeleteOneModel.CollationSet = true
	return dom
}

// DeleteManyModel is the write model for deleteMany operations.
type DeleteManyModel struct {
	dispatch.DeleteManyModel
}

// Filter sets the filter for the DeleteManyModel.
func (dmm *DeleteManyModel) Filter(filter interface{}) *DeleteManyModel {
	dmm.DeleteManyModel.Filter = filter
	return dmm
}

// Collation sets the collation for the DeleteManyModel.
func (dmm *DeleteManyModel) Collation(collation *mongoopt.Collation) *DeleteManyModel {
	dmm.DeleteManyModel.Collation = collation.Convert()
	dmm.DeleteManyModel.CollationSet = true
	return dmm
}

// ReplaceOneModel is the write model for replace operations.
type ReplaceOneModel struct {
	dispatch.ReplaceOneModel
}

// Filter sets the filter for the ReplaceOneModel.
func (rom *ReplaceOneModel) Filter(filter interface{}) *ReplaceOneModel {
	rom.ReplaceOneModel.Filter = filter
	return rom
}

// Replacement sets the replacement document for the ReplaceOneModel.
func (rom *ReplaceOneModel) Replacement(rep interface{}) *ReplaceOneModel {
	rom.ReplaceOneModel.Replacement = rep
	return rom
}

// Collation sets the collation for the ReplaceOneModel.
func (rom *ReplaceOneModel) Collation(collation *mongoopt.Collation) *ReplaceOneModel {
	rom.ReplaceOneModel.Collation = collation.Convert()
	rom.ReplaceOneModel.CollationSet = true
	return rom
}

// Upsert specifies if a new document should be created if no document matches the query.
func (rom *ReplaceOneModel) Upsert(upsert bool) *ReplaceOneModel {
	rom.ReplaceOneModel.Upsert = upsert
	rom.ReplaceOneModel.UpsertSet = true
	return rom
}

// UpdateOneModel is the write model for update operations.
type UpdateOneModel struct {
	dispatch.UpdateOneModel
}

// Filter sets the filter for the UpdateOneModel.
func (uom *UpdateOneModel) Filter(filter interface{}) *UpdateOneModel {
	uom.UpdateOneModel.Filter = filter
	return uom
}

// Update sets the update document for the UpdateOneModel.
func (uom *UpdateOneModel) Update(update interface{}) *UpdateOneModel {
	uom.UpdateOneModel.Update = update
	return uom
}

// ArrayFilters specifies a set of filters specifying to which array elements an update should apply.
func (uom *UpdateOneModel) ArrayFilters(filters []interface{}) *UpdateOneModel {
	uom.UpdateOneModel.ArrayFilters = filters
	uom.UpdateOneModel.ArrayFiltersSet = true
	return uom
}

// Collation sets the collation for the UpdateOneModel.
func (uom *UpdateOneModel) Collation(collation *mongoopt.Collation) *UpdateOneModel {
	uom.UpdateOneModel.Collation = collation.Convert()
	uom.UpdateOneModel.CollationSet = true
	return uom
}

// Upsert specifies if a new document should be created if no document matches the query.
func (uom *UpdateOneModel) Upsert(upsert bool) *UpdateOneModel {
	uom.UpdateOneModel.Upsert = upsert
	uom.UpdateOneModel.UpsertSet = true
	return uom
}

// UpdateManyModel is the write model for updateMany operations.
type UpdateManyModel struct {
	dispatch.UpdateManyModel
}

// Filter sets the filter for the UpdateManyModel.
func (umm *UpdateManyModel) Filter(filter interface{}) *UpdateManyModel {
	umm.UpdateManyModel.Filter = filter
	return umm
}

// Update sets the update document for the UpdateManyModel.
func (umm *UpdateManyModel) Update(update interface{}) *UpdateManyModel {
	umm.UpdateManyModel.Update = update
	return umm
}

// ArrayFilters specifies a set of filters specifying to which array elements an update should apply.
func (umm *UpdateManyModel) ArrayFilters(filters []interface{}) *UpdateManyModel {
	umm.UpdateManyModel.ArrayFilters = filters
	umm.UpdateManyModel.ArrayFiltersSet = true
	return umm
}

// Collation sets the collation for the UpdateManyModel.
func (umm *UpdateManyModel) Collation(collation *mongoopt.Collation) *UpdateManyModel {
	umm.UpdateManyModel.Collation = collation.Convert()
	umm.UpdateManyModel.CollationSet = true
	return umm
}

// Upsert specifies if a new document should be created if no document matches the query.
func (umm *UpdateManyModel) Upsert(upsert bool) *UpdateManyModel {
	umm.UpdateManyModel.Upsert = upsert
	umm.UpdateManyModel.UpsertSet = true
	return umm
}
