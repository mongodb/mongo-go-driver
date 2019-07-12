// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package mongo

import (
	"context"
	"io/ioutil"
	"math"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	encryptionTestsDir = "../data/client-side-encryption"
)

var majorityWcDbOpts = options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
var majorityWcCollOpts = options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

type cseTestFile struct {
	RunOn          []*runOn   `bson:"runOn"`
	DatabaseName   string     `bson:"database_name"`
	CollectionName string     `bson:"collection_name"`
	Data           []bson.Raw `bson:"data"`
	JSONSchema     bson.Raw   `bson:"json_schema"`
	KeyVaultData   []bson.Raw `bson:"key_vault_data"`
	Tests          []*cseTest `bson:"tests"`
}

type cseTest struct {
	Description   string            `bson:"description"`
	SkipReason    string            `bson:"skipReason"`
	ClientOptions bson.Raw          `bson:"clientOptions"`
	Operations    []*cseOperation   `bson:"operations"`
	Expectations  []*cseExpectation `bson:"expectations"`
	Outcome       *cseOutcome       `bson:"outcome"`
}

type cseOperation struct {
	Name              string      `bson:"name"`
	Object            string      `json:"object"`
	CollectionOptions bson.Raw    `bson:"collectionOptions"`
	Result            interface{} `bson:"result"`
	Arguments         bson.Raw    `bson:"arguments"`
	Error             bool        `bson:"error"`
}

type cseExpectation struct {
	CommandStartedEvent struct {
		CommandName  string   `bson:"command_name"`
		DatabaseName string   `bson:"database_name"`
		Command      bson.Raw `bson:"command"`
	} `bson:"command_started_event"`
}

type cseOutcome struct {
	Collection struct {
		Data interface{} `bson:"data"`
	} `bson:"collection"`
}

type cseError struct {
	ErrorContains      *string  `bson:"errorContains"`
	ErrorCodeName      *string  `bson:"errorCodeName"`
	ErrorLabelsContain []string `bson:"errorLabelsContain"`
	ErrorLabelsOmit    []string `bson:"errorLabelsOmit"`
}

var cseRegistry *bsoncodec.Registry
var cseURI string
var cseCommandStarted []*event.CommandStartedEvent

func TestClientSideEncryptionSpec(t *testing.T) {
	rawType := reflect.TypeOf(bson.Raw{})
	cseRegistry = bson.NewRegistryBuilder().RegisterTypeMapEntry(bson.TypeEmbeddedDocument, rawType).Build()
	cseURI = testutil.ConnString(t).Original

	for _, file := range testhelpers.FindJSONFilesInDir(t, encryptionTestsDir) {
		t.Run(file, func(t *testing.T) {
			runEncryptionTestFile(t, path.Join(encryptionTestsDir, file))
		})
	}
}

func runEncryptionTestFile(t *testing.T, filepath string) {
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err, "error reading file: %v", err)

	var testfile cseTestFile
	err = bson.UnmarshalExtJSONWithRegistry(cseRegistry, content, true, &testfile)
	require.NoError(t, err, "error creating testfile struct: %v", err)

	dbName := "admin"
	dbAdmin := createTestDatabase(t, &dbName)
	version, err := getServerVersion(dbAdmin)
	require.NoError(t, err, "error getting server version: %v", err)
	runTest := len(testfile.RunOn) == 0
	for _, reqs := range testfile.RunOn {
		if shouldExecuteTest(t, version, reqs) {
			runTest = true
			break
		}
	}
	if !runTest {
		t.Skip("skipping test file because no matching environmental constraint found")
	}

	for _, test := range testfile.Tests {
		t.Run(test.Description, func(t *testing.T) {
			runEncryptionTest(t, testfile, test)
		})
	}
}

func runEncryptionTest(t *testing.T, testfile cseTestFile, test *cseTest) {
	if test.Description == "operation fails with maxWireVersion < 8" {
		t.Skip("skipping maxWireVersion test")
	}
	if len(test.SkipReason) > 0 {
		t.Skip(test.SkipReason)
	}

	// setup key vault (admin.datakeys)
	admin := "admin"
	datakeys := "datakeys"
	coll := createTestCollection(t, &admin, &datakeys, majorityWcCollOpts)
	err := coll.Drop(ctx)
	require.NoError(t, err, "error dropping admin.datakeys: %v", err)
	if len(testfile.KeyVaultData) > 0 {
		_, err = coll.InsertMany(ctx, rawSliceToInterfaceSlice(testfile.KeyVaultData))
		require.NoError(t, err, "error inserting KeyVaultData: %v", err)
	}

	// setup collection
	setupClient := createTestClient(t)
	setupDb := setupClient.Database(testfile.DatabaseName, majorityWcDbOpts)
	setupColl := setupDb.Collection(testfile.CollectionName)
	err = setupColl.Drop(ctx)
	require.NoError(t, err, "error dropping test collection: %v", err)

	// explicitly create collection if JSON schema required
	if len(testfile.JSONSchema) != 0 {
		validator := bson.D{
			{"$jsonSchema", testfile.JSONSchema},
		}
		err = setupDb.RunCommand(ctx, bson.D{
			{"create", testfile.CollectionName},
			{"validator", validator},
		}).Err()
		require.NoError(t, err, "error creating collection with JSON schema: %v", err)
	}
	if len(testfile.Data) > 0 {
		_, err = setupColl.InsertMany(ctx, rawSliceToInterfaceSlice(testfile.Data))
		require.NoError(t, err, "error inserting test data: %v", err)
	}

	// create test client
	clientOpts := createClientOptions(t, test.ClientOptions)
	clientOpts.ApplyURI(cseURI)
	clientOpts.SetMonitor(&event.CommandMonitor{
		Started: func(_ context.Context, cse *event.CommandStartedEvent) {
			cseCommandStarted = append(cseCommandStarted, cse)
		},
	})

	client, err := Connect(ctx, clientOpts)
	require.NoError(t, err, "error creating client: %v", err)
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	testDb := client.Database(testfile.DatabaseName)
	cseCommandStarted = cseCommandStarted[:0]
	for _, op := range test.Operations {
		coll := testDb.Collection(testfile.CollectionName, createCollectionOptions(t, op.CollectionOptions))

		switch op.Object {
		case "database":
			err = cseExecuteDatabseOperation(t, op, testDb)
		case "", "collection":
			err = cseExecuteCollectionOperation(t, op, coll)
		}

		cseVerifyError(t, err, op.Result)
	}
	cseCheckExpectations(t, test.Expectations)
	if test.Outcome != nil {
		cur, err := setupColl.Find(ctx, bson.D{})
		require.NoError(t, err, "error getting collection contents: %v", err)
		cseVerifyCursorResult(t, cur, test.Outcome.Collection.Data)
	}
}

func cseVerifyError(t *testing.T, err error, result interface{}) {
	t.Helper()

	expected := cseErrorFromResult(t, result)
	expectErr := expected != nil
	gotErr := err != nil
	require.Equal(t, expectErr, gotErr, "expected error: %v, got err: %v", expectErr, gotErr)
	if !expectErr {
		return
	}

	// check ErrorContains for all error types
	if expected.ErrorContains != nil {
		emsg := strings.ToLower(*expected.ErrorContains)
		amsg := strings.ToLower(err.Error())
		require.Contains(t, amsg, emsg, "expected '%v' to contain '%v'", amsg, emsg)
	}

	cerr, ok := err.(CommandError)
	if ok {
		if expected.ErrorCodeName != nil {
			require.Equal(t, *expected.ErrorCodeName, cerr.Name,
				"error name mismatch; expected %v, got %v", expected.ErrorCodeName, cerr.Name)
		}
		for _, label := range expected.ErrorLabelsContain {
			require.True(t, cerr.HasErrorLabel(label),
				"expected error %v to contain label %v", err, label)
		}
		for _, label := range expected.ErrorLabelsOmit {
			require.False(t, cerr.HasErrorLabel(label),
				"expected error %v to not contain label %v", err, label)
		}
		return
	}
}

func cseExecuteDatabseOperation(t *testing.T, op *cseOperation, db *Database) error {
	switch op.Name {
	case "aggregate":
		res, err := cseAggregate(t, db, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyCursorResult(t, res, op.Result)
		}
		return err
	case "runCommand":
		res := cseRunCommand(t, db, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && res.Err() == nil {
			cseVerifySingleResult(t, res, op.Result)
		}
		return res.Err()
	default:
		t.Fatalf("unrecognized database operation: %v", op.Name)
	}
	return nil
}

func cseExecuteCollectionOperation(t *testing.T, op *cseOperation, coll *Collection) error {
	switch op.Name {
	case "count":
		t.Skip("count has been deprectated")
	case "distinct":
		res, err := cseDistinct(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyDistinctResult(t, res, op.Result)
		}
		return err
	case "insertOne":
		res, err := cseInsertOne(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyInsertOneResult(t, res, op.Result)
		}
		return err
	case "insertMany":
		res, err := cseInsertMany(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyInsertManyResult(t, res, op.Result)
		}
		return err
	case "find":
		res, err := cseFind(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyCursorResult(t, res, op.Result)
		}
		return err
	case "findOne":
		res := cseFindOne(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && res.Err() == nil {
			cseVerifySingleResult(t, res, op.Result)
		}
		return res.Err()
	case "findOneAndDelete":
		res := cseFindOneAndDelete(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && res.Err() == nil {
			cseVerifySingleResult(t, res, op.Result)
		}
		return res.Err()
	case "findOneAndUpdate":
		res := cseFindOneAndUpdate(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && res.Err() == nil {
			cseVerifySingleResult(t, res, op.Result)
		}
		return res.Err()
	case "findOneAndReplace":
		res := cseFindOneAndReplace(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && res.Err() == nil {
			cseVerifySingleResult(t, res, op.Result)
		}
		return res.Err()
	case "deleteOne":
		res, err := cseDeleteOne(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyDeleteResult(t, res, op.Result)
		}
		return err
	case "deleteMany":
		res, err := cseDeleteMany(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyDeleteResult(t, res, op.Result)
		}
		return err
	case "updateOne":
		res, err := cseUpdateOne(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyUpdateResult(t, res, op.Result)
		}
		return err
	case "updateMany":
		res, err := cseUpdateMany(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyUpdateResult(t, res, op.Result)
		}
		return err
	case "replaceOne":
		res, err := cseReplaceOne(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyUpdateResult(t, res, op.Result)
		}
		return err
	case "aggregate":
		res, err := cseAggregate(t, coll, op.Arguments)
		if cseErrorFromResult(t, op.Result) == nil && err == nil {
			cseVerifyCursorResult(t, res, op.Result)
		}
		return err
	case "bulkWrite":
		_, err := cseBulkWrite(t, coll, op.Arguments)
		return err
	case "mapReduce":
		t.Skip("skipping because mapReduce is not supported")
	default:
		t.Fatalf("unrecognized collection operation: %v", op.Name)
	}

	return nil
}

func cseErrorFromResult(t *testing.T, result interface{}) *cseError {
	t.Helper()

	// embedded doc will be unmarshalled as Raw
	raw, ok := result.(bson.Raw)
	if !ok {
		return nil
	}

	var expected cseError
	err := bson.Unmarshal(raw, &expected)
	if err != nil {
		return nil
	}
	if expected.ErrorCodeName == nil && expected.ErrorContains == nil && len(expected.ErrorLabelsOmit) == 0 &&
		len(expected.ErrorLabelsContain) == 0 {
		return nil
	}

	return &expected
}

func cseRunCommand(t *testing.T, db *Database, args bson.Raw) *SingleResult {
	var cmd bson.Raw
	opts := options.RunCmd()

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "command":
			cmd = val.Document()
		case "readPreference":
			opts.SetReadPreference(createReadPreferenceFromRawValue(val))
		default:
			t.Fatalf("unrecognized runCommand option: %v", key)
		}
	}

	return db.RunCommand(ctx, cmd, opts)
}

func cseDistinct(t *testing.T, coll *Collection, args bson.Raw) ([]interface{}, error) {
	var fieldName string
	var filter bson.Raw
	opts := options.Distinct()

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "fieldName":
			fieldName = val.StringValue()
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized distinct option: %v", key)
		}
	}

	return coll.Distinct(ctx, fieldName, filter, opts)
}

func cseInsertOne(t *testing.T, coll *Collection, args bson.Raw) (*InsertOneResult, error) {
	var doc bson.Raw
	opts := options.InsertOne()

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "document":
			doc = val.Document()
		case "bypassDocumentValidation":
			opts.SetBypassDocumentValidation(val.Boolean())
		default:
			t.Fatalf("unrecognized insertOne option: %v", key)
		}
	}

	return coll.InsertOne(context.Background(), doc, opts)
}

func cseInsertMany(t *testing.T, coll *Collection, args bson.Raw) (*InsertManyResult, error) {
	var rawDocs bson.Raw
	opts := options.InsertMany()

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "documents":
			rawDocs = val.Array()
		default:
			t.Fatalf("unregonized insertMany option: %v", key)
		}
	}

	return coll.InsertMany(context.Background(), interfaceSliceFromRawArray(t, rawDocs), opts)
}

func cseFind(t *testing.T, coll *Collection, args bson.Raw) (*Cursor, error) {
	opts := options.Find()
	var filter bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "sort":
			opts = opts.SetSort(val.Document())
		case "skip":
			opts = opts.SetSkip(int64(val.Int32()))
		case "limit":
			opts = opts.SetLimit(int64(val.Int32()))
		case "batchSize":
			opts = opts.SetBatchSize(val.Int32())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized find option: %v", key)
		}
	}

	return coll.Find(ctx, filter, opts)
}

func cseFindOne(t *testing.T, coll *Collection, args bson.Raw) *SingleResult {
	var filter bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		default:
			t.Fatalf("unrecognized findOne option: %v", key)
		}
	}

	return coll.FindOne(ctx, filter)
}

func cseFindOneAndDelete(t *testing.T, coll *Collection, args bson.Raw) *SingleResult {
	opts := options.FindOneAndDelete()
	var filter bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "sort":
			opts = opts.SetSort(val.Document())
		case "projection":
			opts = opts.SetProjection(val.Document())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized findOneAndDelete option: %v", key)
		}
	}

	return coll.FindOneAndDelete(ctx, filter, opts)
}

func cseFindOneAndUpdate(t *testing.T, coll *Collection, args bson.Raw) *SingleResult {
	opts := options.FindOneAndUpdate()
	var filter bson.Raw
	var update bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "update":
			update = val.Document()
		case "arrayFilters":
			opts = opts.SetArrayFilters(options.ArrayFilters{
				Filters: interfaceSliceFromRawArray(t, val.Array()),
			})
		case "sort":
			opts = opts.SetSort(val.Document())
		case "projection":
			opts = opts.SetProjection(val.Document())
		case "upsert":
			opts = opts.SetUpsert(val.Boolean())
		case "returnDocument":
			switch val.StringValue() {
			case "After":
				opts = opts.SetReturnDocument(options.After)
			case "Before":
				opts = opts.SetReturnDocument(options.Before)
			}
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized findOneAndUpdate option: %v", key)
		}
	}

	return coll.FindOneAndUpdate(ctx, filter, update, opts)
}

func cseFindOneAndReplace(t *testing.T, coll *Collection, args bson.Raw) *SingleResult {
	opts := options.FindOneAndReplace()
	var filter bson.Raw
	var replacement bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "replacement":
			replacement = val.Document()
		case "sort":
			opts = opts.SetSort(val.Document())
		case "projection":
			opts = opts.SetProjection(val.Document())
		case "upsert":
			opts = opts.SetUpsert(val.Boolean())
		case "returnDocument":
			switch val.StringValue() {
			case "After":
				opts = opts.SetReturnDocument(options.After)
			case "Before":
				opts = opts.SetReturnDocument(options.Before)
			}
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized findOneAndReplace option: %v", key)
		}
	}

	return coll.FindOneAndReplace(ctx, filter, replacement, opts)
}

func cseDeleteOne(t *testing.T, coll *Collection, args bson.Raw) (*DeleteResult, error) {
	opts := options.Delete()
	var filter bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized deleteOne option: %v", key)
		}
	}

	return coll.DeleteOne(ctx, filter, opts)
}

func cseDeleteMany(t *testing.T, coll *Collection, args bson.Raw) (*DeleteResult, error) {
	opts := options.Delete()
	var filter bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized deleteMany option: %v", key)
		}
	}

	return coll.DeleteMany(ctx, filter, opts)
}

func cseReplaceOne(t *testing.T, coll *Collection, args bson.Raw) (*UpdateResult, error) {
	opts := options.Replace()
	var filter bson.Raw
	var replacement bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "replacement":
			replacement = val.Document()
		case "upsert":
			opts = opts.SetUpsert(val.Boolean())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized replaceOne option: %v", key)
		}
	}

	// TODO temporarily default upsert to false explicitly to make test pass
	// because we do not send upsert=false by default
	//opts = opts.SetUpsert(false)
	if opts.Upsert == nil {
		opts = opts.SetUpsert(false)
	}
	return coll.ReplaceOne(ctx, filter, replacement, opts)
}

func cseUpdateOne(t *testing.T, coll *Collection, args bson.Raw) (*UpdateResult, error) {
	opts := options.Update()
	var filter bson.Raw
	var update bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "update":
			update = val.Document()
		case "arrayFilters":
			opts = opts.SetArrayFilters(options.ArrayFilters{Filters: interfaceSliceFromRawArray(t, val.Array())})
		case "upsert":
			opts = opts.SetUpsert(val.Boolean())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized updateOne option: %v", key)
		}
	}

	if opts.Upsert == nil {
		opts = opts.SetUpsert(false)
	}
	return coll.UpdateOne(ctx, filter, update, opts)
}

func cseUpdateMany(t *testing.T, coll *Collection, args bson.Raw) (*UpdateResult, error) {
	opts := options.Update()
	var filter bson.Raw
	var update bson.Raw

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filter":
			filter = val.Document()
		case "update":
			update = val.Document()
		case "arrayFilters":
			opts = opts.SetArrayFilters(options.ArrayFilters{Filters: interfaceSliceFromRawArray(t, val.Array())})
		case "upsert":
			opts = opts.SetUpsert(val.Boolean())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		default:
			t.Fatalf("unrecognized updateMany option: %v", key)
		}
	}

	if opts.Upsert == nil {
		opts = opts.SetUpsert(false)
	}
	return coll.UpdateMany(ctx, filter, update, opts)
}

func cseAggregate(t *testing.T, agg aggregator, args bson.Raw) (*Cursor, error) {
	var pipeline []interface{}
	opts := options.Aggregate()

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "pipeline":
			pipeline = interfaceSliceFromRawArray(t, val.Array())
		case "batchSize":
			opts = opts.SetBatchSize(val.Int32())
		case "collation":
			opts = opts.SetCollation(collationFromRaw(t, val.Document()))
		case "maxTimeMS":
			opts = opts.SetMaxTime(time.Duration(val.Int32()) * time.Millisecond)
		default:
			t.Fatalf("unrecognized aggregate option: %v", key)
		}
	}

	return agg.Aggregate(ctx, pipeline, opts)
}

func cseBulkWrite(t *testing.T, coll *Collection, args bson.Raw) (*BulkWriteResult, error) {
	models := createBulkWriteModels(t, args.Lookup("requests").Array())
	opts := options.BulkWrite()

	rawOpts, err := args.LookupErr("options")
	if err == nil {
		optsElems, _ := rawOpts.Document().Elements()
		for _, oe := range optsElems {
			optName := oe.Key()
			optVal := oe.Value()

			switch optName {
			case "ordered":
				opts.SetOrdered(optVal.Boolean())
			default:
				t.Fatalf("unrecognized bulk write option: %v", optName)
			}
		}
	}
	return coll.BulkWrite(ctx, models, opts)
}

func createBulkWriteModels(t *testing.T, rawModels bson.Raw) []WriteModel {
	var models []WriteModel

	vals, _ := rawModels.Values()
	for _, val := range vals {
		models = append(models, createBulkWriteModel(t, val.Document()))
	}
	return models
}

func createBulkWriteModel(t *testing.T, rawModel bson.Raw) WriteModel {
	name := rawModel.Lookup("name").StringValue()
	args := rawModel.Lookup("arguments").Document()

	switch name {
	case "insertOne":
		return NewInsertOneModel().SetDocument(args.Lookup("document").Document())
	case "updateOne":
		f := args.Lookup("filter").Document()
		u := args.Lookup("update").Document()
		return NewUpdateOneModel().SetFilter(f).SetUpdate(u)
	case "deleteOne":
		return NewDeleteOneModel().SetFilter(args.Lookup("filter").Document())
	default:
		t.Fatalf("unrecognized model type: %v", name)
	}

	return nil
}

// verification functions

func cseVerifyCursorResult(t *testing.T, cur *Cursor, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	require.NotNil(t, cur, "cursor was nil")
	for _, expectedDoc := range result.(bson.A) {
		require.True(t, cur.Next(ctx))

		var actual bsonx.Doc
		require.NoError(t, cur.Decode(&actual))
		var expected bsonx.Doc
		require.NoError(t, bson.Unmarshal(expectedDoc.(bson.Raw), &expected))

		compareDocs(t, expected, actual)
	}

	require.False(t, cur.Next(ctx))
	require.NoError(t, cur.Err())
}

func cseVerifyDistinctResult(t *testing.T, res []interface{}, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	for i, expected := range result.(bson.A) {
		var i64 int64
		foundType := true

		switch t := expected.(type) {
		case int32:
			i64 = int64(t)
		case int64:
			i64 = t
		case float64:
			i64 = int64(t)
		default:
			foundType = false
		}

		actual := res[i]
		iActual := testhelpers.GetIntFromInterface(actual)
		var iExpected *int64
		if foundType {
			iExpected = &i64
		}

		require.Equal(t, iExpected == nil, iActual == nil)
		if iExpected != nil {
			require.Equal(t, *iExpected, *iActual)
			continue
		}

		require.Equal(t, expected, res[i])
	}
}

func cseVerifyInsertOneResult(t *testing.T, res *InsertOneResult, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	var expected InsertOneResult
	err := bson.Unmarshal(result.(bson.Raw), &expected)
	require.Nil(t, err)

	expectedID := expected.InsertedID
	if f, ok := expectedID.(float64); ok && f == math.Floor(f) {
		expectedID = int32(f)
	}

	if expectedID != nil {
		require.NotNil(t, res)
		require.Equal(t, expectedID, res.InsertedID)
	}
}

func cseVerifyInsertManyResult(t *testing.T, res *InsertManyResult, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	var expected struct{ InsertedIds map[string]interface{} }
	err := bson.Unmarshal(result.(bson.Raw), &expected)
	require.Nil(t, err)

	if expected.InsertedIds != nil {
		require.NotNil(t, res)
		replaceFloatsWithInts(expected.InsertedIds)

		for _, val := range expected.InsertedIds {
			require.Contains(t, res.InsertedIDs, val)
		}
	}
}

func cseVerifyDeleteResult(t *testing.T, res *DeleteResult, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	var expected struct {
		DeletedCount int64 `bson:"deletedCount"`
	}
	err := bson.Unmarshal(result.(bson.Raw), &expected)
	require.Nil(t, err)

	require.Equal(t, expected.DeletedCount, res.DeletedCount)
}

func cseVerifyUpdateResult(t *testing.T, res *UpdateResult, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	var expected struct {
		MatchedCount  int64 `bson:"matchedCount"`
		ModifiedCount int64 `bson:"modifiedCount"`
		UpsertedCount int64 `bson:"upsertedCount"`
	}
	err := bson.Unmarshal(result.(bson.Raw), &expected)
	require.Nil(t, err)

	require.Equal(t, expected.MatchedCount, res.MatchedCount)
	require.Equal(t, expected.ModifiedCount, res.ModifiedCount)

	actualUpsertedCount := int64(0)
	if res.UpsertedID != nil {
		actualUpsertedCount = 1
	}

	require.Equal(t, expected.UpsertedCount, actualUpsertedCount)
}

func cseVerifyCountResult(t *testing.T, n int64, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	var expected int64
	switch res := result.(type) {
	case int32:
		expected = int64(res)
	case int64:
		expected = res
	case float64:
		expected = int64(res)
	default:
		t.Fatalf("invalid result type for count: %T", result)
	}

	require.Equal(t, expected, n, "count mismatch; expected %v, got %v", expected, n)
}

func cseVerifySingleResult(t *testing.T, res *SingleResult, result interface{}) {
	t.Helper()

	if result == nil {
		return
	}

	expectedRaw := result.(bson.Raw)
	var actual bsonx.Doc
	err := res.Decode(&actual)
	if err == ErrNoDocuments {
		require.Equal(t, bsoncore.EmptyDocumentLength, len(expectedRaw),
			"expected document %v, got nil", expectedRaw)
		return
	}

	var expected bsonx.Doc
	err = bson.Unmarshal(expectedRaw, &expected)
	require.NoError(t, err)
	require.True(t, expected.Equal(actual), "result document mismatch; expected %v, got %v", expected, actual)
}

func cseCheckExpectations(t *testing.T, expectations []*cseExpectation) {
	for i, expectation := range expectations {
		if i == len(cseCommandStarted) {
			require.Fail(t, "Expected command started event", expectation.CommandStartedEvent.CommandName)
		}
		evt := cseCommandStarted[i]

		expectedCmdName := expectation.CommandStartedEvent.CommandName
		require.Equal(t, expectedCmdName, evt.CommandName, "command name mismatch; expected %v, got %v", expectedCmdName, evt.CommandName)

		expectedDb := expectation.CommandStartedEvent.DatabaseName
		if expectedDb != "" {
			require.Equal(t, expectedDb, evt.DatabaseName,
				"database mismatch for cmd %v; expected %v, got %v", expectedCmdName, expectedDb, evt.DatabaseName)
		}

		var expected bsonx.Doc
		err := bson.Unmarshal(expectation.CommandStartedEvent.Command, &expected)
		require.NoError(t, err, "error unmarshalling expected command: %v", err)

		actual := evt.Command
		for _, elem := range expected {
			key := elem.Key
			val := elem.Value

			actualVal := actual.Lookup(key)

			// Keys that may be nil
			if val.Type() == bson.TypeNull {
				require.Equal(t, actual.Lookup(key), bson.RawValue{}, "Expected %s to be nil", key)
				continue
			}
			if key == "ordered" || key == "cursor" {
				// TODO: some tests specify that "ordered" must be a key in the event but ordered isn't a valid option for some of these cases (e.g. insertOne)
				// TODO: some tests specify "cursor" subdocument for listCollections
				continue
			}

			// Keys that should not be nil
			require.NotEqual(t, actualVal.Type, bsontype.Null, "Expected %v, got nil for key: %s", elem, key)
			require.NoError(t, actualVal.Validate(), "Expected %v, couldn't validate", elem)

			switch key {
			case "getMore":
				require.NotNil(t, actualVal, "Expected %v, got nil for key: %s", elem, key)
				// type assertion for $$type
				valDoc, ok := val.DocumentOK()
				if ok {
					typeStr := valDoc.Lookup("$$type").StringValue()
					assertType(t, actualVal.Type, typeStr)
					break
				}

				expectedCursorID := val.Int64()
				// ignore if equal to 42
				if expectedCursorID != 42 {
					require.Equal(t, expectedCursorID, actualVal.Int64())
				}
			case "readConcern":
				rcExpectDoc := val.Document()
				rcActualDoc := actualVal.Document()
				clusterTime := rcExpectDoc.Lookup("afterClusterTime")
				level := rcExpectDoc.Lookup("level")
				if clusterTime.Type() != bsontype.Null {
					require.NotNil(t, rcActualDoc.Lookup("afterClusterTime"))
				}
				if level.Type() != bsontype.Null {
					doc, err := bsonx.ReadDoc(rcActualDoc)
					require.NoError(t, err)
					compareElements(t, rcExpectDoc.LookupElement("level"), doc.LookupElement("level"))
				}
			default:
				doc, err := bsonx.ReadDoc(actual)
				require.NoError(t, err)
				compareElements(t, elem, doc.LookupElement(key))
			}
		}
	}
}
