// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/bsonutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// This file contains helpers to execute client operations.

func executeCreateChangeStream(ctx context.Context, operation *operation) (*operationResult, error) {
	var watcher interface {
		Watch(context.Context, interface{}, ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	}
	var err error

	watcher, err = entities(ctx).client(operation.Object)
	if err != nil {
		watcher, err = entities(ctx).database(operation.Object)
	}
	if err != nil {
		watcher, err = entities(ctx).collection(operation.Object)
	}
	if err != nil {
		return nil, fmt.Errorf("no client, database, or collection entity found with ID %q", operation.Object)
	}

	var pipeline []interface{}
	opts := options.ChangeStream()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "batchSize":
			opts.SetBatchSize(val.Int32())
		case "collation":
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %w", err)
			}
			opts.SetCollation(*collation)
		case "comment":
			commentString, err := createCommentString(val)
			if err != nil {
				return nil, fmt.Errorf("error creating comment: %w", err)
			}
			opts.SetComment(commentString)
		case "fullDocument":
			switch fd := val.StringValue(); fd {
			case "default":
				opts.SetFullDocument(options.Default)
			case "required":
				opts.SetFullDocument(options.Required)
			case "updateLookup":
				opts.SetFullDocument(options.UpdateLookup)
			case "whenAvailable":
				opts.SetFullDocument(options.WhenAvailable)
			default:
				return nil, fmt.Errorf("unrecognized fullDocument value %q", fd)
			}
		case "fullDocumentBeforeChange":
			switch fdbc := val.StringValue(); fdbc {
			case "off":
				opts.SetFullDocumentBeforeChange(options.Off)
			case "required":
				opts.SetFullDocumentBeforeChange(options.Required)
			case "whenAvailable":
				opts.SetFullDocumentBeforeChange(options.WhenAvailable)
			}
		case "maxAwaitTimeMS":
			opts.SetMaxAwaitTime(time.Duration(val.Int32()) * time.Millisecond)
		case "pipeline":
			pipeline = bsonutil.RawToInterfaces(bsonutil.RawToDocuments(val.Array())...)
		case "resumeAfter":
			opts.SetResumeAfter(val.Document())
		case "showExpandedEvents":
			opts.SetShowExpandedEvents(val.Boolean())
		case "startAfter":
			opts.SetStartAfter(val.Document())
		case "startAtOperationTime":
			t, i := val.Timestamp()
			opts.SetStartAtOperationTime(&primitive.Timestamp{T: t, I: i})
		default:
			return nil, fmt.Errorf("unrecognized createChangeStream option %q", key)
		}
	}
	if pipeline == nil {
		return nil, newMissingArgumentError("pipeline")
	}

	stream, err := watcher.Watch(ctx, pipeline, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	// createChangeStream is sometimes used with no corresponding saveResultAsEntity field. Return an
	// empty result in this case.
	if operation.ResultEntityID != nil {
		if err := entities(ctx).addCursorEntity(*operation.ResultEntityID, stream); err != nil {
			return nil, fmt.Errorf("error storing result as cursor entity: %w", err)
		}
	}
	return newEmptyResult(), nil
}

func executeListDatabases(ctx context.Context, operation *operation, nameOnly bool) (*operationResult, error) {
	client, err := entities(ctx).client(operation.Object)
	if err != nil {
		return nil, err
	}

	// We set a default filter rather than erroring if the Arguments doc doesn't have a "filter" field because the
	// spec says drivers should support this field, not must.
	filter := emptyDocument
	opts := options.ListDatabases().SetNameOnly(nameOnly)

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "authorizedDatabases":
			opts.SetAuthorizedDatabases(val.Boolean())
		case "filter":
			filter = val.Document()
		case "nameOnly":
			opts.SetNameOnly(val.Boolean())
		default:
			return nil, fmt.Errorf("unrecognized listDatabases option %q", key)
		}
	}

	res, err := client.ListDatabases(ctx, filter, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	specsArray := bsoncore.NewArrayBuilder()
	for _, spec := range res.Databases {
		rawSpec := bsoncore.NewDocumentBuilder().
			AppendString("name", spec.Name).
			AppendInt64("sizeOnDisk", spec.SizeOnDisk).
			AppendBoolean("empty", spec.Empty).
			Build()

		specsArray.AppendDocument(rawSpec)
	}
	raw := bsoncore.NewDocumentBuilder().
		AppendArray("databases", specsArray.Build()).
		AppendInt64("totalSize", res.TotalSize).
		Build()
	return newDocumentResult(raw, nil), nil
}

func executeClientBulkWrite(ctx context.Context, operation *operation) (*operationResult, error) {
	client, err := entities(ctx).client(operation.Object)
	if err != nil {
		return nil, err
	}

	wirteModels := &mongo.ClientWriteModels{}
	opts := options.ClientBulkWrite()

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "models":
			models, err := val.Array().Values()
			if err != nil {
				return nil, err
			}
			for _, m := range models {
				model := m.Document().Index(0)
				err = appendClientBulkWriteModel(model.Key(), model.Value().Document(), wirteModels)
				if err != nil {
					return nil, err
				}
			}
		case "bypassDocumentValidation":
			opts.SetBypassDocumentValidation(val.Boolean())
		case "comment":
			opts.SetComment(val)
		case "let":
			opts.SetLet(val.Document())
		case "ordered":
			opts.SetOrdered(val.Boolean())
		case "verboseResults":
			opts.SetVerboseResults(val.Boolean())
		case "writeConcern":
			var wc writeConcern
			err := bson.Unmarshal(val.Value, &wc)
			if err != nil {
				return nil, err
			}
			c, err := wc.toWriteConcernOption()
			if err != nil {
				return nil, err
			}
			opts.SetWriteConcern(c)
		default:
			return nil, fmt.Errorf("unrecognized bulkWrite option %q", key)
		}
	}

	res, err := client.BulkWrite(ctx, wirteModels, opts)
	if res == nil {
		var bwe mongo.ClientBulkWriteException
		if !errors.As(err, &bwe) || bwe.PartialResult == nil {
			return newDocumentResult(emptyCoreDocument, err), nil
		}
		res = bwe.PartialResult
	}
	rawBuilder := bsoncore.NewDocumentBuilder().
		AppendInt64("deletedCount", res.DeletedCount).
		AppendInt64("insertedCount", res.InsertedCount).
		AppendInt64("matchedCount", res.MatchedCount).
		AppendInt64("modifiedCount", res.ModifiedCount).
		AppendInt64("upsertedCount", res.UpsertedCount)

	var resBuilder *bsoncore.DocumentBuilder

	resBuilder = bsoncore.NewDocumentBuilder()
	for k, v := range res.DeleteResults {
		resBuilder.AppendDocument(strconv.Itoa(k),
			bsoncore.NewDocumentBuilder().
				AppendInt64("deletedCount", v.DeletedCount).
				Build(),
		)
	}
	rawBuilder.AppendDocument("deleteResults", resBuilder.Build())

	resBuilder = bsoncore.NewDocumentBuilder()
	for k, v := range res.InsertResults {
		t, d, err := bson.MarshalValue(v.InsertedID)
		if err != nil {
			return nil, err
		}
		resBuilder.AppendDocument(strconv.Itoa(k),
			bsoncore.NewDocumentBuilder().
				AppendValue("insertedId", bsoncore.Value{Type: t, Data: d}).
				Build(),
		)
	}
	rawBuilder.AppendDocument("insertResults", resBuilder.Build())

	resBuilder = bsoncore.NewDocumentBuilder()
	for k, v := range res.UpdateResults {
		b := bsoncore.NewDocumentBuilder().
			AppendInt64("matchedCount", v.MatchedCount).
			AppendInt64("modifiedCount", v.ModifiedCount)
		if v.UpsertedID != nil {
			t, d, err := bson.MarshalValue(v.UpsertedID)
			if err != nil {
				return nil, err
			}
			b.AppendValue("upsertedId", bsoncore.Value{Type: t, Data: d})
		}
		resBuilder.AppendDocument(strconv.Itoa(k), b.Build())
	}
	rawBuilder.AppendDocument("updateResults", resBuilder.Build())

	return newDocumentResult(rawBuilder.Build(), err), nil
}

func appendClientBulkWriteModel(key string, value bson.Raw, model *mongo.ClientWriteModels) error {
	switch key {
	case "insertOne":
		namespace, m, err := createClientInsertOneModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendInsertOne(ns[0], ns[1], m)
	case "updateOne":
		namespace, m, err := createClientUpdateOneModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendUpdateOne(ns[0], ns[1], m)
	case "updateMany":
		namespace, m, err := createClientUpdateManyModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendUpdateMany(ns[0], ns[1], m)
	case "replaceOne":
		namespace, m, err := createClientReplaceOneModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendReplaceOne(ns[0], ns[1], m)
	case "deleteOne":
		namespace, m, err := createClientDeleteOneModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendDeleteOne(ns[0], ns[1], m)
	case "deleteMany":
		namespace, m, err := createClientDeleteManyModel(value)
		if err != nil {
			return err
		}
		ns := strings.SplitN(namespace, ".", 2)
		model.AppendDeleteMany(ns[0], ns[1], m)
	}
	return nil
}

func createClientInsertOneModel(value bson.Raw) (string, *mongo.ClientInsertOneModel, error) {
	var v struct {
		Namespace string
		Document  bson.Raw
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	return v.Namespace, &mongo.ClientInsertOneModel{
		Document: v.Document,
	}, nil
}

func createClientUpdateOneModel(value bson.Raw) (string, *mongo.ClientUpdateOneModel, error) {
	var v struct {
		Namespace    string
		Filter       bson.Raw
		Update       interface{}
		ArrayFilters []interface{}
		Collation    *options.Collation
		Hint         *bson.RawValue
		Upsert       *bool
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	var hint interface{}
	if v.Hint != nil {
		hint, err = createHint(*v.Hint)
		if err != nil {
			return "", nil, err
		}
	}
	model := &mongo.ClientUpdateOneModel{
		Filter:    v.Filter,
		Update:    v.Update,
		Collation: v.Collation,
		Hint:      hint,
		Upsert:    v.Upsert,
	}
	if len(v.ArrayFilters) > 0 {
		model.ArrayFilters = &options.ArrayFilters{Filters: v.ArrayFilters}
	}
	return v.Namespace, model, nil

}

func createClientUpdateManyModel(value bson.Raw) (string, *mongo.ClientUpdateManyModel, error) {
	var v struct {
		Namespace    string
		Filter       bson.Raw
		Update       interface{}
		ArrayFilters []interface{}
		Collation    *options.Collation
		Hint         *bson.RawValue
		Upsert       *bool
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	var hint interface{}
	if v.Hint != nil {
		hint, err = createHint(*v.Hint)
		if err != nil {
			return "", nil, err
		}
	}
	model := &mongo.ClientUpdateManyModel{
		Filter:    v.Filter,
		Update:    v.Update,
		Collation: v.Collation,
		Hint:      hint,
		Upsert:    v.Upsert,
	}
	if len(v.ArrayFilters) > 0 {
		model.ArrayFilters = &options.ArrayFilters{Filters: v.ArrayFilters}
	}
	return v.Namespace, model, nil
}

func createClientReplaceOneModel(value bson.Raw) (string, *mongo.ClientReplaceOneModel, error) {
	var v struct {
		Namespace   string
		Filter      bson.Raw
		Replacement bson.Raw
		Collation   *options.Collation
		Hint        *bson.RawValue
		Upsert      *bool
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	var hint interface{}
	if v.Hint != nil {
		hint, err = createHint(*v.Hint)
		if err != nil {
			return "", nil, err
		}
	}
	return v.Namespace, &mongo.ClientReplaceOneModel{
		Filter:      v.Filter,
		Replacement: v.Replacement,
		Collation:   v.Collation,
		Hint:        hint,
		Upsert:      v.Upsert,
	}, nil
}

func createClientDeleteOneModel(value bson.Raw) (string, *mongo.ClientDeleteOneModel, error) {
	var v struct {
		Namespace string
		Filter    bson.Raw
		Collation *options.Collation
		Hint      *bson.RawValue
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	var hint interface{}
	if v.Hint != nil {
		hint, err = createHint(*v.Hint)
		if err != nil {
			return "", nil, err
		}
	}
	return v.Namespace, &mongo.ClientDeleteOneModel{
		Filter:    v.Filter,
		Collation: v.Collation,
		Hint:      hint,
	}, nil
}

func createClientDeleteManyModel(value bson.Raw) (string, *mongo.ClientDeleteManyModel, error) {
	var v struct {
		Namespace string
		Filter    bson.Raw
		Collation *options.Collation
		Hint      *bson.RawValue
	}
	err := bson.Unmarshal(value, &v)
	if err != nil {
		return "", nil, err
	}
	var hint interface{}
	if v.Hint != nil {
		hint, err = createHint(*v.Hint)
		if err != nil {
			return "", nil, err
		}
	}
	return v.Namespace, &mongo.ClientDeleteManyModel{
		Filter:    v.Filter,
		Collation: v.Collation,
		Hint:      hint,
	}, nil
}
