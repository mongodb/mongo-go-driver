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
		return mongo.NewInsertOneModel().SetDocument(args.Lookup("document").Document()), nil
	case "updateOne":
		uom := mongo.NewUpdateOneModel().SetFilter(args.Lookup("filter").Document())
		update, err := createUpdateValue(args.Lookup("update"))
		if err != nil {
			return nil, fmt.Errorf("error creating update: %v", err)
		}
		uom.SetUpdate(update)

		if val, err := args.LookupErr("arrayFilters"); err == nil {
			uom.SetArrayFilters(options.ArrayFilters{
				Filters: testhelpers.RawToInterfaceSlice(val.Array()),
			})
		}
		if val, err := args.LookupErr("collation"); err == nil {
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			uom.SetCollation(collation)
		}
		if val, err := args.LookupErr("hint"); err == nil {
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			uom.SetHint(hint)
		}
		if val, err := args.LookupErr("upsert"); err == nil {
			uom.SetUpsert(val.Boolean())
		}

		return uom, nil
	case "updateMany":
		umm := mongo.NewUpdateManyModel().SetFilter(args.Lookup("filter").Document())
		update, err := createUpdateValue(args.Lookup("update"))
		if err != nil {
			return nil, fmt.Errorf("error creating update: %v", err)
		}
		umm.SetUpdate(update)

		if val, err := args.LookupErr("arrayFilters"); err == nil {
			umm.SetArrayFilters(options.ArrayFilters{
				Filters: testhelpers.RawToInterfaceSlice(val.Array()),
			})
		}
		if val, err := args.LookupErr("collation"); err == nil {
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			umm.SetCollation(collation)
		}
		if val, err := args.LookupErr("hint"); err == nil {
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			umm.SetHint(hint)
		}
		if val, err := args.LookupErr("upsert"); err == nil {
			umm.SetUpsert(val.Boolean())
		}

		return umm, nil
	case "deleteOne":
		dom := mongo.NewDeleteOneModel().SetFilter(args.Lookup("filter").Document())

		if val, err := args.LookupErr("collation"); err == nil {
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			dom.SetCollation(collation)
		}
		if val, err := args.LookupErr("hint"); err == nil {
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			dom.SetHint(hint)
		}

		return dom, nil
	case "deleteMany":
		dmm := mongo.NewDeleteManyModel().SetFilter(args.Lookup("filter").Document())

		if val, err := args.LookupErr("collation"); err == nil {
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			dmm.SetCollation(collation)
		}
		if val, err := args.LookupErr("hint"); err == nil {
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			dmm.SetHint(hint)
		}

		return dmm, nil
	case "replaceOne":
		rom := mongo.NewReplaceOneModel().
			SetFilter(args.Lookup("filter").Document()).
			SetReplacement(args.Lookup("replacement").Document())

		if val, err := args.LookupErr("collation"); err == nil {
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			rom.SetCollation(collation)
		}
		if val, err := args.LookupErr("hint"); err == nil {
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			rom.SetHint(hint)
		}
		if val, err := args.LookupErr("upsert"); err == nil {
			rom.SetUpsert(val.Boolean())
		}

		return rom, nil
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
