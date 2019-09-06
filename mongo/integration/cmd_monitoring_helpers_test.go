// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

// Helper functions to compare BSON values and command monitoring expectations.

// compare BSON values and fail if they are not equal. the key parameter is used for error strings.
func compareValues(mt *mtest.T, key string, expected, actual bson.RawValue) {
	mt.Helper()

	switch expected.Type {
	case bson.TypeInt64:
		e := expected.Int64()

		switch actual.Type {
		case bson.TypeInt32:
			a := actual.Int32()
			assert.Equal(mt, e, int64(a), "value mismatch for key %s; expected %v, got %v", key, e, a)
		case bson.TypeInt64:
			a := actual.Int64()
			assert.Equal(mt, e, a, "value mismatch for key %s; expected %v, got %v", key, e, a)
		case bson.TypeDouble:
			a := actual.Double()
			assert.Equal(mt, e, int64(a), "value mismatch for key %s; expected %v, got %v", key, e, a)
		default:
			mt.Fatalf("cannot convert value of type %v for key %s to int64", actual.Type, key)
		}
	case bson.TypeInt32:
		e := expected.Int32()

		switch actual.Type {
		case bson.TypeInt32:
			a := actual.Int32()
			assert.Equal(mt, e, a, "value mismatch for key %s; expected %v, got %v", key, e, a)
		case bson.TypeInt64:
			a := actual.Int64()
			assert.Equal(mt, e, int32(a), "value mismatch for key %s; expected %v, got %v", key, e, a)
		case bson.TypeDouble:
			a := actual.Double()
			assert.Equal(mt, e, int32(a), "value mismatch for key %s; expected %v, got %v", key, e, a)
		default:
			mt.Fatalf("cannot convert value of type %v for key %s to int32", actual.Type, key)
		}
	case bson.TypeEmbeddedDocument:
		e := expected.Document()
		a := actual.Document()
		compareDocs(mt, e, a)
	case bson.TypeArray:
		e := expected.Array()
		a := actual.Array()
		compareDocs(mt, e, a)
	default:
		assert.Equal(mt, expected.Value, actual.Value,
			"value mismatch; expected %v, got %v", expected.Value, actual.Value)
	}
}

// compare expected and actual BSON documents. comparison succeeds if actual contains each element in expected.
func compareDocs(mt *mtest.T, expected, actual bson.Raw) {
	mt.Helper()

	eElems, err := expected.Elements()
	assert.Nil(mt, err, "error getting expected elements: %v", err)

	for _, e := range eElems {
		eKey := e.Key()
		aVal, err := actual.LookupErr(eKey)
		assert.Nil(mt, err, "key %s not found in result", e.Key())

		eVal := e.Value()
		if doc, ok := eVal.DocumentOK(); ok {
			// nested doc
			compareDocs(mt, doc, aVal.Document())
			continue
		}

		compareValues(mt, eKey, eVal, aVal)
	}
}

func checkExpectations(mt *mtest.T, expectations []*expectation, id0 bsonx.Doc, id1 bsonx.Doc) {
	mt.Helper()

	started := mt.GetAllStartedEvents()

	for i, expectation := range expectations {
		expected := expectation.CommandStartedEvent
		if i == len(started) {
			mt.Fatalf("expected event for %s", expectation.CommandStartedEvent.CommandName)
		}

		evt := started[i]
		if expected.CommandName != "" {
			assert.Equal(mt, expected.CommandName, evt.CommandName,
				"cmd name mismatch; expected %s, got %s", expected.CommandName, evt.CommandName)
		}
		if expected.DatabaseName != "" {
			assert.Equal(mt, expected.DatabaseName, evt.DatabaseName,
				"db name mismatch; expected %s, got %s", expected.DatabaseName, evt.DatabaseName)
		}

		eElems, err := expected.Command.Elements()
		assert.Nil(mt, err, "error getting expected elements: %v", err)

		for _, elem := range eElems {
			key := elem.Key()
			val := elem.Value()

			actualVal := evt.Command.Lookup(key)

			// Keys that may be nil
			if val.Type == bson.TypeNull {
				assert.Equal(mt, bson.RawValue{}, actualVal, "expected value for key %s to be nil but got %v", key, actualVal)
				continue
			}
			if key == "ordered" {
				// TODO: some tests specify that "ordered" must be a key in the event but ordered isn't a valid option for some of these cases (e.g. insertOne)
				continue
			}

			// keys that should not be nil
			assert.NotEqual(mt, bson.TypeNull, actualVal.Type, "expected value %v for key %s but got nil", val, key)
			err = actualVal.Validate()
			assert.Nil(mt, err, "error validating value for key %s: %v", key, err)

			switch key {
			case "lsid":
				sessName := val.StringValue()
				var expectedID bson.Raw
				actualID := actualVal.Document()

				switch sessName {
				case "session0":
					expectedID, err = id0.MarshalBSON()
				case "session1":
					expectedID, err = id1.MarshalBSON()
				default:
					mt.Fatalf("unrecognized session identifier: %v", sessName)
				}
				assert.Nil(mt, err, "error getting expected session ID bytes: %v", err)

				assert.Equal(mt, expectedID, actualID,
					"session ID mismatch for session %v; expected %v, got %v", sessName, expectedID, actualID)
			case "getMore":
				expectedID := val.Int64()
				// ignore placeholder cursor ID (42)
				if expectedID != 42 {
					actualID := actualVal.Int64()
					assert.Equal(mt, expectedID, actualID, "cursor ID mismatch; expected %v, got %v", expectedID, actualID)
				}
			case "readConcern":
				expectedRc := val.Document()
				actualRc := actualVal.Document()
				eClusterTime := expectedRc.Lookup("afterClusterTime")
				eLevel := expectedRc.Lookup("level")

				if eClusterTime.Type != bson.TypeNull {
					aClusterTime := actualRc.Lookup("afterClusterTime")
					// ignore placeholder cluster time (42)
					ctInt32, ok := eClusterTime.Int32OK()
					if ok && ctInt32 == 42 {
						continue
					}

					assert.Equal(mt, eClusterTime, aClusterTime,
						"cluster time mismatch; expected %v, got %v", eClusterTime, aClusterTime)
				}
				if eLevel.Type != bson.TypeNull {
					aLevel := actualRc.Lookup("level")
					assert.Equal(mt, eLevel, aLevel, "level mismatch; expected %v, got %v", eLevel, aLevel)
				}
			case "recoveryToken":
				// recovery token is a document but can be "42" in the expectations
				if rt, ok := val.Int32OK(); ok {
					assert.Equal(mt, rt, int32(42), "expected int32 recovery token to be 42, got %v", rt)
					return
				}

				compareDocs(mt, val.Document(), actualVal.Document())
			default:
				compareValues(mt, key, val, actualVal)
			}
		}
	}
}
