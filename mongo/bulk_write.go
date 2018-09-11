package mongo

import (
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

// WriteModel is the interface satisfied by all models for bulk writes.
type WriteModel interface {
	convertModel() dispatch.WriteModel
}

// InsertOneModel is the write model for insert operations.
type InsertOneModel struct {
	dispatch.InsertOneModel
}

// NewInsertOneModel creates a new InsertOneModel.
func NewInsertOneModel() *InsertOneModel {
	return &InsertOneModel{}
}

// Document sets the BSON document for the InsertOneModel.
func (iom *InsertOneModel) Document(doc interface{}) *InsertOneModel {
	iom.InsertOneModel.Document = doc
	return iom
}

func (iom *InsertOneModel) convertModel() dispatch.WriteModel {
	return iom.InsertOneModel
}

// DeleteOneModel is the write model for delete operations.
type DeleteOneModel struct {
	dispatch.DeleteOneModel
}

// NewDeleteOneModel creates a new DeleteOneModel.
func NewDeleteOneModel() *DeleteOneModel {
	return &DeleteOneModel{}
}

// Filter sets the filter for the DeleteOneModel.
func (dom *DeleteOneModel) Filter(filter interface{}) *DeleteOneModel {
	dom.DeleteOneModel.Filter = filter
	return dom
}

// Collation sets the collation for the DeleteOneModel.
func (dom *DeleteOneModel) Collation(collation *mongoopt.Collation) *DeleteOneModel {
	dom.DeleteOneModel.Collation = collation.Convert()
	return dom
}

func (dom *DeleteOneModel) convertModel() dispatch.WriteModel {
	return dom.DeleteOneModel
}

// DeleteManyModel is the write model for deleteMany operations.
type DeleteManyModel struct {
	dispatch.DeleteManyModel
}

// NewDeleteManyModel creates a new DeleteManyModel.
func NewDeleteManyModel() *DeleteManyModel {
	return &DeleteManyModel{}
}

// Filter sets the filter for the DeleteManyModel.
func (dmm *DeleteManyModel) Filter(filter interface{}) *DeleteManyModel {
	dmm.DeleteManyModel.Filter = filter
	return dmm
}

// Collation sets the collation for the DeleteManyModel.
func (dmm *DeleteManyModel) Collation(collation *mongoopt.Collation) *DeleteManyModel {
	dmm.DeleteManyModel.Collation = collation.Convert()
	return dmm
}

func (dmm *DeleteManyModel) convertModel() dispatch.WriteModel {
	return dmm.DeleteManyModel
}

// ReplaceOneModel is the write model for replace operations.
type ReplaceOneModel struct {
	dispatch.ReplaceOneModel
}

// NewReplaceOneModel creates a new ReplaceOneModel.
func NewReplaceOneModel() *ReplaceOneModel {
	return &ReplaceOneModel{}
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
	return rom
}

// Upsert specifies if a new document should be created if no document matches the query.
func (rom *ReplaceOneModel) Upsert(upsert bool) *ReplaceOneModel {
	rom.ReplaceOneModel.Upsert = upsert
	rom.ReplaceOneModel.UpsertSet = true
	return rom
}

func (rom *ReplaceOneModel) convertModel() dispatch.WriteModel {
	return rom.ReplaceOneModel
}

// UpdateOneModel is the write model for update operations.
type UpdateOneModel struct {
	dispatch.UpdateOneModel
}

// NewUpdateOneModel creates a new UpdateOneModel.
func NewUpdateOneModel() *UpdateOneModel {
	return &UpdateOneModel{}
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
	return uom
}

// Collation sets the collation for the UpdateOneModel.
func (uom *UpdateOneModel) Collation(collation *mongoopt.Collation) *UpdateOneModel {
	uom.UpdateOneModel.Collation = collation.Convert()
	return uom
}

// Upsert specifies if a new document should be created if no document matches the query.
func (uom *UpdateOneModel) Upsert(upsert bool) *UpdateOneModel {
	uom.UpdateOneModel.Upsert = upsert
	uom.UpdateOneModel.UpsertSet = true
	return uom
}

func (uom *UpdateOneModel) convertModel() dispatch.WriteModel {
	return uom.UpdateOneModel
}

// UpdateManyModel is the write model for updateMany operations.
type UpdateManyModel struct {
	dispatch.UpdateManyModel
}

// NewUpdateManyModel creates a new UpdateManyModel.
func NewUpdateManyModel() *UpdateManyModel {
	return &UpdateManyModel{}
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
	return umm
}

// Collation sets the collation for the UpdateManyModel.
func (umm *UpdateManyModel) Collation(collation *mongoopt.Collation) *UpdateManyModel {
	umm.UpdateManyModel.Collation = collation.Convert()
	return umm
}

// Upsert specifies if a new document should be created if no document matches the query.
func (umm *UpdateManyModel) Upsert(upsert bool) *UpdateManyModel {
	umm.UpdateManyModel.Upsert = upsert
	umm.UpdateManyModel.UpsertSet = true
	return umm
}

func (umm *UpdateManyModel) convertModel() dispatch.WriteModel {
	return umm.UpdateManyModel
}

func dispatchToMongoModel(model dispatch.WriteModel) WriteModel {
	switch conv := model.(type) {
	case dispatch.InsertOneModel:
		return &InsertOneModel{conv}
	case dispatch.DeleteOneModel:
		return &DeleteOneModel{conv}
	case dispatch.DeleteManyModel:
		return &DeleteManyModel{conv}
	case dispatch.ReplaceOneModel:
		return &ReplaceOneModel{conv}
	case dispatch.UpdateOneModel:
		return &UpdateOneModel{conv}
	case dispatch.UpdateManyModel:
		return &UpdateManyModel{conv}
	}

	return nil
}
