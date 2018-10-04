// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/runcmdopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
	"github.com/stretchr/testify/require"
)

// Various helper functions for crud related operations

// Mutates the client to add options
func addClientOptions(c *Client, opts map[string]interface{}) {
	for name, opt := range opts {
		switch name {
		case "retryWrites":
			c.retryWrites = opt.(bool)
		case "w":
			switch opt.(type) {
			case float64:
				c.writeConcern = writeconcern.New(writeconcern.W(int(opt.(float64))))
			case string:
				c.writeConcern = writeconcern.New(writeconcern.WMajority())
			}
		case "readConcernLevel":
			c.readConcern = readconcern.New(readconcern.Level(opt.(string)))
		case "readPreference":
			c.readPreference = readPrefFromString(opt.(string))
		}
	}
}

// Mutates the collection to add options
func addCollectionOptions(c *Collection, opts map[string]interface{}) {
	for name, opt := range opts {
		switch name {
		case "readConcern":
			c.readConcern = getReadConcern(opt)
		case "writeConcern":
			c.writeConcern = getWriteConcern(opt)
		case "readPreference":
			c.readPreference = readPrefFromString(opt.(map[string]interface{})["mode"].(string))
		}
	}
}

func executeCount(sess *sessionImpl, coll *Collection, args map[string]interface{}) (int64, error) {
	var filter map[string]interface{}
	var bundle *countopt.CountBundle
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "skip":
			bundle = bundle.Skip(int64(opt.(float64)))
		case "limit":
			bundle = bundle.Limit(int64(opt.(float64)))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		// EXAMPLE:
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.Count(sessCtx, filter, bundle)
	}
	return coll.Count(ctx, filter, bundle)
}

func executeDistinct(sess *sessionImpl, coll *Collection, args map[string]interface{}) ([]interface{}, error) {
	var fieldName string
	var filter map[string]interface{}
	var bundle *distinctopt.DistinctBundle
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "fieldName":
			fieldName = opt.(string)
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.Distinct(sessCtx, fieldName, filter, bundle)
	}
	return coll.Distinct(ctx, fieldName, filter, bundle)
}

func executeInsertOne(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*InsertOneResult, error) {
	document := args["document"].(map[string]interface{})

	// For some reason, the insertion document is unmarshaled with a float rather than integer,
	// but the documents that are used to initially populate the collection are unmarshaled
	// correctly with integers. To ensure that the tests can correctly compare them, we iterate
	// through the insertion document and change any valid integers stored as floats to actual
	// integers.
	replaceFloatsWithInts(document)

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.InsertOne(sessCtx, document)
	}
	return coll.InsertOne(context.Background(), document)
}

func executeInsertMany(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*InsertManyResult, error) {
	documents := args["documents"].([]interface{})

	// For some reason, the insertion documents are unmarshaled with a float rather than
	// integer, but the documents that are used to initially populate the collection are
	// unmarshaled correctly with integers. To ensure that the tests can correctly compare
	// them, we iterate through the insertion documents and change any valid integers stored
	// as floats to actual integers.
	for i, doc := range documents {
		docM := doc.(map[string]interface{})
		replaceFloatsWithInts(docM)

		documents[i] = docM
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.InsertMany(sessCtx, documents)
	}
	return coll.InsertMany(context.Background(), documents)
}

func executeFind(sess *sessionImpl, coll *Collection, args map[string]interface{}) (Cursor, error) {
	var bundle *findopt.FindBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "skip":
			bundle = bundle.Skip(int64(opt.(float64)))
		case "limit":
			bundle = bundle.Limit(int64(opt.(float64)))
		case "batchSize":
			bundle = bundle.BatchSize(int32(opt.(float64)))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.Find(sessCtx, filter, bundle)
	}
	return coll.Find(ctx, filter, bundle)
}

func executeFindOneAndDelete(sess *sessionImpl, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.DeleteOneBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.FindOneAndDelete(sessCtx, filter, bundle)
	}
	return coll.FindOneAndDelete(ctx, filter, bundle)
}

func executeFindOneAndUpdate(sess *sessionImpl, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.UpdateOneBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "returnDocument":
			switch opt.(string) {
			case "After":
				bundle = bundle.ReturnDocument(mongoopt.After)
			case "Before":
				bundle = bundle.ReturnDocument(mongoopt.Before)
			}
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.FindOneAndUpdate(sessCtx, filter, update, bundle)
	}
	return coll.FindOneAndUpdate(ctx, filter, update, bundle)
}

func executeFindOneAndReplace(sess *sessionImpl, coll *Collection, args map[string]interface{}) *DocumentResult {
	var bundle *findopt.ReplaceOneBundle
	var filter map[string]interface{}
	var replacement map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "replacement":
			replacement = opt.(map[string]interface{})
		case "sort":
			bundle = bundle.Sort(opt.(map[string]interface{}))
		case "projection":
			bundle = bundle.Projection(opt.(map[string]interface{}))
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "returnDocument":
			switch opt.(string) {
			case "After":
				bundle = bundle.ReturnDocument(mongoopt.After)
			case "Before":
				bundle = bundle.ReturnDocument(mongoopt.Before)
			}
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and replacement documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(replacement)

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.FindOneAndReplace(sessCtx, filter, replacement, bundle)
	}
	return coll.FindOneAndReplace(ctx, filter, replacement, bundle)
}

func executeDeleteOne(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*DeleteResult, error) {
	var bundle *deleteopt.DeleteBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter document is unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.DeleteOne(sessCtx, filter, bundle)
	}
	return coll.DeleteOne(ctx, filter, bundle)
}

func executeDeleteMany(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*DeleteResult, error) {
	var bundle *deleteopt.DeleteBundle
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter document is unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.DeleteMany(sessCtx, filter, bundle)
	}
	return coll.DeleteMany(ctx, filter, bundle)
}

func executeReplaceOne(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *replaceopt.ReplaceBundle
	var filter map[string]interface{}
	var replacement map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "replacement":
			replacement = opt.(map[string]interface{})
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and replacement documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(replacement)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = replaceopt.BundleReplace(replaceopt.Upsert(false), bundle)
	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.ReplaceOne(sessCtx, filter, replacement, bundle)
	}
	return coll.ReplaceOne(ctx, filter, replacement, bundle)
}

func executeUpdateOne(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *updateopt.UpdateBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = updateopt.BundleUpdate(updateopt.Upsert(false), bundle)
	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.UpdateOne(sessCtx, filter, update, bundle)
	}
	return coll.UpdateOne(ctx, filter, update, bundle)
}

func executeUpdateMany(sess *sessionImpl, coll *Collection, args map[string]interface{}) (*UpdateResult, error) {
	var bundle *updateopt.UpdateBundle
	var filter map[string]interface{}
	var update map[string]interface{}
	for name, opt := range args {
		switch name {
		case "filter":
			filter = opt.(map[string]interface{})
		case "update":
			update = opt.(map[string]interface{})
		case "arrayFilters":
			bundle = bundle.ArrayFilters(opt.([]interface{})...)
		case "upsert":
			bundle = bundle.Upsert(opt.(bool))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	// For some reason, the filter and update documents are unmarshaled with floats
	// rather than integers, but the documents that are used to initially populate the
	// collection are unmarshaled correctly with integers. To ensure that the tests can
	// correctly compare them, we iterate through the filter and replacement documents and
	// change any valid integers stored as floats to actual integers.
	replaceFloatsWithInts(filter)
	replaceFloatsWithInts(update)

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	bundle = updateopt.BundleUpdate(updateopt.Upsert(false), bundle)
	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.UpdateMany(sessCtx, filter, update, bundle)
	}
	return coll.UpdateMany(ctx, filter, update, bundle)
}

func executeAggregate(sess *sessionImpl, coll *Collection, args map[string]interface{}) (Cursor, error) {
	var bundle *aggregateopt.AggregateBundle
	var pipeline []interface{}
	for name, opt := range args {
		switch name {
		case "pipeline":
			pipeline = opt.([]interface{})
		case "batchSize":
			bundle = bundle.BatchSize(int32(opt.(float64)))
		case "collation":
			bundle = bundle.Collation(collationFromMap(opt.(map[string]interface{})))
		}
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return coll.Aggregate(sessCtx, pipeline, bundle)
	}
	return coll.Aggregate(ctx, pipeline, bundle)
}

func executeRunCommand(sess *sessionImpl, db *Database, argmap map[string]interface{}, args json.RawMessage) (bson.Reader, error) {
	var bundle *runcmdopt.RunCmdBundle
	cmd := bson.NewDocument()
	for name, opt := range argmap {
		switch name {
		case "command":
			argBytes, err := args.MarshalJSON()
			if err != nil {
				return nil, err
			}

			var argCmdStruct struct {
				Cmd json.RawMessage `json:"command"`
			}
			err = json.NewDecoder(bytes.NewBuffer(argBytes)).Decode(&argCmdStruct)
			if err != nil {
				return nil, err
			}

			err = bson.UnmarshalExtJSON(argCmdStruct.Cmd, true, &cmd)
			if err != nil {
				return nil, err
			}
		case "readPreference":
			bundle = bundle.ReadPreference(getReadPref(opt))
		}
	}

	if sess != nil {
		sessCtx := sessionContext{
			Context: context.WithValue(ctx, sessionKey{}, sess),
			Session: sess,
		}
		return db.RunCommand(sessCtx, cmd, bundle)
	}
	return db.RunCommand(ctx, cmd, bundle)
}

func verifyBulkWriteResult(t *testing.T, res *BulkWriteResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected BulkWriteResult
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	require.Equal(t, expected.DeletedCount, res.DeletedCount)
	require.Equal(t, expected.InsertedCount, res.InsertedCount)
	require.Equal(t, expected.MatchedCount, res.MatchedCount)
	require.Equal(t, expected.ModifiedCount, res.ModifiedCount)
	require.Equal(t, expected.UpsertedCount, res.UpsertedCount)

	// replace floats with ints
	for opID, upsertID := range expected.UpsertedIDs {
		if floatID, ok := upsertID.(float64); ok {
			expected.UpsertedIDs[opID] = int32(floatID)
		}
	}

	for operationID, upsertID := range expected.UpsertedIDs {
		require.Equal(t, upsertID, res.UpsertedIDs[operationID])
	}
}

func verifyInsertOneResult(t *testing.T, res *InsertOneResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected InsertOneResult
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	expectedID := expected.InsertedID
	if f, ok := expectedID.(float64); ok && f == math.Floor(f) {
		expectedID = int32(f)
	}

	if expectedID != nil {
		require.NotNil(t, res)
		require.Equal(t, expectedID, res.InsertedID.(*bson.Element).Value().Interface())
	}
}

func verifyInsertManyResult(t *testing.T, res *InsertManyResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected struct{ InsertedIds map[string]interface{} }
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	if expected.InsertedIds != nil {
		replaceFloatsWithInts(expected.InsertedIds)

		for i, elem := range res.InsertedIDs {
			res.InsertedIDs[i] = elem.(*bson.Element).Value().Interface()
		}

		for _, val := range expected.InsertedIds {
			require.Contains(t, res.InsertedIDs, val)
		}
	}
}

func verifyCursorResult(t *testing.T, cur Cursor, result json.RawMessage) {
	for _, expected := range docSliceFromRaw(t, result) {
		require.NotNil(t, cur)
		require.True(t, cur.Next(context.Background()))

		var actual *bson.Document
		require.NoError(t, cur.Decode(&actual))

		compareDocs(t, expected, actual)
	}

	require.False(t, cur.Next(ctx))
	require.NoError(t, cur.Err())
}

func verifyDocumentResult(t *testing.T, res *DocumentResult, result json.RawMessage) {
	jsonBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var actual *bson.Document
	err = res.Decode(&actual)
	if err == ErrNoDocuments {
		var expected map[string]interface{}
		err := json.NewDecoder(bytes.NewBuffer(jsonBytes)).Decode(&expected)
		require.NoError(t, err)

		require.Nil(t, expected)
		return
	}

	require.NoError(t, err)

	doc := bson.NewDocument()
	err = bson.UnmarshalExtJSON(jsonBytes, true, &doc)
	require.NoError(t, err)

	require.True(t, doc.Equal(actual))
}

func verifyDistinctResult(t *testing.T, res []interface{}, result json.RawMessage) {
	resultBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected []interface{}
	require.NoError(t, json.NewDecoder(bytes.NewBuffer(resultBytes)).Decode(&expected))

	require.Equal(t, len(expected), len(res))

	for i := range expected {
		expectedElem := expected[i]
		actualElem := res[i]

		iExpected := testhelpers.GetIntFromInterface(expectedElem)
		iActual := testhelpers.GetIntFromInterface(actualElem)

		require.Equal(t, iExpected == nil, iActual == nil)
		if iExpected != nil {
			require.Equal(t, *iExpected, *iActual)
			continue
		}

		require.Equal(t, expected[i], res[i])
	}
}

func verifyDeleteResult(t *testing.T, res *DeleteResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected DeleteResult
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)
	require.NoError(t, err)

	require.Equal(t, expected.DeletedCount, res.DeletedCount)
}

func verifyUpdateResult(t *testing.T, res *UpdateResult, result json.RawMessage) {
	expectedBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	var expected struct {
		MatchedCount  int64
		ModifiedCount int64
		UpsertedCount int64
	}
	err = json.NewDecoder(bytes.NewBuffer(expectedBytes)).Decode(&expected)

	require.Equal(t, expected.MatchedCount, res.MatchedCount)
	require.Equal(t, expected.ModifiedCount, res.ModifiedCount)

	actualUpsertedCount := int64(0)
	if res.UpsertedID != nil {
		actualUpsertedCount = 1
	}

	require.Equal(t, expected.UpsertedCount, actualUpsertedCount)
}

func verifyRunCommandResult(t *testing.T, res bson.Reader, result json.RawMessage) {
	jsonBytes, err := result.MarshalJSON()
	require.NoError(t, err)

	expected := bson.NewDocument()
	err = bson.UnmarshalExtJSON(jsonBytes, true, &expected)
	require.NoError(t, err)

	require.NotNil(t, res)
	actual, err := bson.ReadDocument(res)
	require.NoError(t, err)

	// All runcommand results in tests are for key "n" only
	compareElements(t, expected.LookupElement("n"), actual.LookupElement("n"))
}

func verifyCollectionContents(t *testing.T, coll *Collection, result json.RawMessage) {
	cursor, err := coll.Find(context.Background(), nil)
	require.NoError(t, err)

	verifyCursorResult(t, cursor, result)
}

func sanitizeCollectionName(kind string, name string) string {
	// Collections can't have "$" in their names, so we substitute it with "%".
	name = strings.Replace(name, "$", "%", -1)

	// Namespaces can only have 120 bytes max.
	if len(kind+"."+name) >= 119 {
		name = name[:119-len(kind+".")]
	}

	return name
}

func compareElements(t *testing.T, expected *bson.Element, actual *bson.Element) {
	if expected.Value().IsNumber() {
		if expectedNum, ok := expected.Value().Int64OK(); ok {
			switch actual.Value().Type() {
			case bson.TypeInt32:
				require.Equal(t, expectedNum, int64(actual.Value().Int32()), "For key %v", expected.Key())
			case bson.TypeInt64:
				require.Equal(t, expectedNum, actual.Value().Int64(), "For key %v\n", expected.Key())
			case bson.TypeDouble:
				require.Equal(t, expectedNum, int64(actual.Value().Double()), "For key %v\n", expected.Key())
			}
		} else {
			expectedNum := expected.Value().Int32()
			switch actual.Value().Type() {
			case bson.TypeInt32:
				require.Equal(t, expectedNum, actual.Value().Int32(), "For key %v", expected.Key())
			case bson.TypeInt64:
				require.Equal(t, expectedNum, int32(actual.Value().Int64()), "For key %v\n", expected.Key())
			case bson.TypeDouble:
				require.Equal(t, expectedNum, int32(actual.Value().Double()), "For key %v\n", expected.Key())
			}
		}
	} else if conv, ok := expected.Value().MutableDocumentOK(); ok {
		actualConv, actualOk := actual.Value().MutableDocumentOK()
		require.True(t, actualOk)
		compareDocs(t, conv, actualConv)
	} else if conv, ok := expected.Value().MutableArrayOK(); ok {
		actualConv, actualOk := actual.Value().MutableArrayOK()
		require.True(t, actualOk)
		compareArrays(t, conv, actualConv)
	} else {
		exp, err := expected.MarshalBSON()
		require.NoError(t, err)
		act, err := actual.MarshalBSON()
		require.NoError(t, err)

		require.True(t, bytes.Equal(exp, act), "For key %s, expected %v\nactual: %v", expected.Key(), expected, actual)
	}
}

func compareArrays(t *testing.T, expected *bson.Array, actual *bson.Array) {
	if expected.Len() != actual.Len() {
		t.Errorf("array length mismatch. expected %d got %d", expected.Len(), actual.Len())
		t.FailNow()
	}

	expectedIter, err := expected.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for expected array: %s", err)

	actualIter, err := actual.Iterator()
	testhelpers.RequireNil(t, err, "error creating iterator for actual array: %s", err)

	for expectedIter.Next() && actualIter.Next() {
		expectedDoc := expectedIter.Value().MutableDocument()
		actualDoc := actualIter.Value().MutableDocument()
		compareDocs(t, expectedDoc, actualDoc)
	}
}

func collationFromMap(m map[string]interface{}) *mongoopt.Collation {
	var collation mongoopt.Collation

	if locale, found := m["locale"]; found {
		collation.Locale = locale.(string)
	}

	if caseLevel, found := m["caseLevel"]; found {
		collation.CaseLevel = caseLevel.(bool)
	}

	if caseFirst, found := m["caseFirst"]; found {
		collation.CaseFirst = caseFirst.(string)
	}

	if strength, found := m["strength"]; found {
		collation.Strength = int(strength.(float64))
	}

	if numericOrdering, found := m["numericOrdering"]; found {
		collation.NumericOrdering = numericOrdering.(bool)
	}

	if alternate, found := m["alternate"]; found {
		collation.Alternate = alternate.(string)
	}

	if maxVariable, found := m["maxVariable"]; found {
		collation.MaxVariable = maxVariable.(string)
	}

	if backwards, found := m["backwards"]; found {
		collation.Backwards = backwards.(bool)
	}

	return &collation
}

func docSliceFromRaw(t *testing.T, raw json.RawMessage) []*bson.Document {
	jsonBytes, err := raw.MarshalJSON()
	require.NoError(t, err)

	array := bson.NewArray()
	err = bson.UnmarshalExtJSON(jsonBytes, true, &array)
	require.NoError(t, err)

	docs := make([]*bson.Document, 0)

	for i := 0; i < array.Len(); i++ {
		item, err := array.Lookup(uint(i))
		require.NoError(t, err)
		docs = append(docs, item.MutableDocument())
	}

	return docs
}

func docSliceToInterfaceSlice(docs []*bson.Document) []interface{} {
	out := make([]interface{}, 0, len(docs))

	for _, doc := range docs {
		out = append(out, doc)
	}

	return out
}

func replaceFloatsWithInts(m map[string]interface{}) {
	for key, val := range m {
		if f, ok := val.(float64); ok && f == math.Floor(f) {
			m[key] = int32(f)
			continue
		}

		if innerM, ok := val.(map[string]interface{}); ok {
			replaceFloatsWithInts(innerM)
			m[key] = innerM
		}
	}
}
