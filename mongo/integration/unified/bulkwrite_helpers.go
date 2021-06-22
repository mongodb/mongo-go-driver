// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// This file provides helper functions to convert BSON documents to WriteModel instances.

// createBulkWriteModels converts a bson.Raw that is internally an array to a slice of WriteModel. Each value in the
// array must be a document in the form { requestType: { optionKey1: optionValue1, ... } }. For example, the document
// { insertOne: { document: { x: 1 } } } would be translated to an InsertOneModel to insert the document { x: 1 }.
func createBulkWriteModels(rawModels bson.Raw) ([]mongo.WriteModel, error) {
	vals, _ := rawModels.Values()
	models := make([]mongo.WriteModel, 0, len(vals))

	for idx, val := range vals {
		model, err := createBulkWriteModel(val.Document())
		if err != nil {
			return nil, fmt.Errorf("error creating model at index %d: %v", idx, err)
		}
		models = append(models, model)
	}
	return models, nil
}

// createBulkWriteModel converts the provided BSON document to a WriteModel.
func createBulkWriteModel(rawModel bson.Raw) (mongo.WriteModel, error) {
	firstElem := rawModel.Index(0)
	requestType := firstElem.Key()
	args := firstElem.Value().Document()

	switch requestType {
	case "insertOne":
		var document bson.Raw
		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "document":
				document = val.Document()
			default:
				return nil, fmt.Errorf("unrecognized insertOne option %q", key)
			}
		}
		if document == nil {
			return nil, newMissingArgumentError("document")
		}

		return mongo.NewInsertOneModel().SetDocument(document), nil
	case "updateOne":
		uom := mongo.NewUpdateOneModel()
		var filter bson.Raw
		var update interface{}
		var err error

		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "arrayFilters":
				uom.SetArrayFilters(options.ArrayFilters{
					Filters: testhelpers.RawToInterfaceSlice(val.Array()),
				})
			case "collation":
				collation, err := createCollation(val.Document())
				if err != nil {
					return nil, fmt.Errorf("error creating collation: %v", err)
				}
				uom.SetCollation(collation)
			case "filter":
				filter = val.Document()
			case "hint":
				hint, err := createHint(val)
				if err != nil {
					return nil, fmt.Errorf("error creating hint: %v", err)
				}
				uom.SetHint(hint)
			case "update":
				update, err = createUpdateValue(val)
				if err != nil {
					return nil, fmt.Errorf("error creating update: %v", err)
				}
			case "upsert":
				uom.SetUpsert(val.Boolean())
			default:
				return nil, fmt.Errorf("unrecognized updateOne option %q", key)
			}
		}
		if filter == nil {
			return nil, newMissingArgumentError("filter")
		}
		if update == nil {
			return nil, newMissingArgumentError("update")
		}

		return uom.SetFilter(filter).SetUpdate(update), nil
	case "updateMany":
		umm := mongo.NewUpdateManyModel()
		var filter bson.Raw
		var update interface{}
		var err error

		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "arrayFilters":
				umm.SetArrayFilters(options.ArrayFilters{
					Filters: testhelpers.RawToInterfaceSlice(val.Array()),
				})
			case "collation":
				collation, err := createCollation(val.Document())
				if err != nil {
					return nil, fmt.Errorf("error creating collation: %v", err)
				}
				umm.SetCollation(collation)
			case "filter":
				filter = val.Document()
			case "hint":
				hint, err := createHint(val)
				if err != nil {
					return nil, fmt.Errorf("error creating hint: %v", err)
				}
				umm.SetHint(hint)
			case "update":
				update, err = createUpdateValue(val)
				if err != nil {
					return nil, fmt.Errorf("error creating update: %v", err)
				}
			case "upsert":
				umm.SetUpsert(val.Boolean())
			default:
				return nil, fmt.Errorf("unrecognized updateMany option %q", key)
			}
		}
		if filter == nil {
			return nil, newMissingArgumentError("filter")
		}
		if update == nil {
			return nil, newMissingArgumentError("update")
		}

		return umm.SetFilter(filter).SetUpdate(update), nil
	case "deleteOne":
		dom := mongo.NewDeleteOneModel()
		var filter bson.Raw

		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "filter":
				filter = val.Document()
			case "hint":
				hint, err := createHint(val)
				if err != nil {
					return nil, fmt.Errorf("error creating hint: %v", err)
				}
				dom.SetHint(hint)
			default:
				return nil, fmt.Errorf("unrecognized deleteOne option %q", key)
			}
		}
		if filter == nil {
			return nil, newMissingArgumentError("filter")
		}

		return dom.SetFilter(filter), nil
	case "deleteMany":
		dmm := mongo.NewDeleteManyModel()
		var filter bson.Raw

		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "collation":
				collation, err := createCollation(val.Document())
				if err != nil {
					return nil, fmt.Errorf("error creating collation: %v", err)
				}
				dmm.SetCollation(collation)
			case "filter":
				filter = val.Document()
			case "hint":
				hint, err := createHint(val)
				if err != nil {
					return nil, fmt.Errorf("error creating hint: %v", err)
				}
				dmm.SetHint(hint)
			default:
				return nil, fmt.Errorf("unrecognized deleteMany option %q", key)
			}
		}
		if filter == nil {
			return nil, newMissingArgumentError("filter")
		}

		return dmm.SetFilter(filter), nil
	case "replaceOne":
		rom := mongo.NewReplaceOneModel()
		var filter, replacement bson.Raw

		elems, _ := args.Elements()
		for _, elem := range elems {
			key := elem.Key()
			val := elem.Value()

			switch key {
			case "collation":
				collation, err := createCollation(val.Document())
				if err != nil {
					return nil, fmt.Errorf("error creating collation: %v", err)
				}
				rom.SetCollation(collation)
			case "filter":
				filter = val.Document()
			case "hint":
				hint, err := createHint(val)
				if err != nil {
					return nil, fmt.Errorf("error creating hint: %v", err)
				}
				rom.SetHint(hint)
			case "replacement":
				replacement = val.Document()
			case "upsert":
				rom.SetUpsert(val.Boolean())
			default:
				return nil, fmt.Errorf("unrecognized replaceOne option %q", key)
			}
		}
		if filter == nil {
			return nil, newMissingArgumentError("filter")
		}
		if replacement == nil {
			return nil, newMissingArgumentError("replacement")
		}

		return rom.SetFilter(filter).SetReplacement(replacement), nil
	default:
		return nil, fmt.Errorf("unrecongized request type: %v", requestType)
	}
}

// createUpdateValue converts the provided RawValue to a value that can be passed to UpdateOne/UpdateMany functions.
// This helper handles both document and pipeline-style updates.
func createUpdateValue(updateVal bson.RawValue) (interface{}, error) {
	switch updateVal.Type {
	case bson.TypeEmbeddedDocument:
		return updateVal.Document(), nil
	case bson.TypeArray:
		var updateDocs []bson.Raw
		docs, _ := updateVal.Array().Values()
		for _, doc := range docs {
			updateDocs = append(updateDocs, doc.Document())
		}

		return updateDocs, nil
	default:
		return nil, fmt.Errorf("unrecognized update type: %s", updateVal.Type)
	}
}
