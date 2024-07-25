// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

const (
	readWriteConcernTestsDir = "../testdata/read-write-concern"
	connstringTestsDir       = "connection-string"
	documentTestsDir         = "document"
)

var (
	serverDefaultConcern = []byte{5, 0, 0, 0, 0} // server default read concern and write concern is empty document
	specTestRegistry     = func() *bson.Registry {
		reg := bson.NewRegistry()
		reg.RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(bson.Raw{}))
		return reg
	}()
)

type connectionStringTestFile struct {
	Tests []connectionStringTest `bson:"tests"`
}

type connectionStringTest struct {
	Description  string   `bson:"description"`
	URI          string   `bson:"uri"`
	Valid        bool     `bson:"valid"`
	ReadConcern  bson.Raw `bson:"readConcern"`
	WriteConcern bson.Raw `bson:"writeConcern"`
	SkipReason   string   `bson:"skipReason"`
}

type documentTestFile struct {
	Tests []documentTest `bson:"tests"`
}

type documentTest struct {
	Description          string    `bson:"description"`
	Valid                bool      `bson:"valid"`
	ReadConcern          bson.Raw  `bson:"readConcern"`
	ReadConcernDocument  *bson.Raw `bson:"readConcernDocument"`
	WriteConcern         bson.Raw  `bson:"writeConcern"`
	WriteConcernDocument *bson.Raw `bson:"writeConcernDocument"`
	IsServerDefault      *bool     `bson:"isServerDefault"`
	IsAcknowledged       *bool     `bson:"isAcknowledged"`
}

func TestReadWriteConcernSpec(t *testing.T) {
	t.Run("connstring", func(t *testing.T) {
		for _, file := range jsonFilesInDir(t, path.Join(readWriteConcernTestsDir, connstringTestsDir)) {
			t.Run(file, func(t *testing.T) {
				runConnectionStringTestFile(t, path.Join(readWriteConcernTestsDir, connstringTestsDir, file))
			})
		}
	})
	t.Run("document", func(t *testing.T) {
		for _, file := range jsonFilesInDir(t, path.Join(readWriteConcernTestsDir, documentTestsDir)) {
			t.Run(file, func(t *testing.T) {
				runDocumentTestFile(t, path.Join(readWriteConcernTestsDir, documentTestsDir, file))
			})
		}
	})
}

func runConnectionStringTestFile(t *testing.T, filePath string) {
	content, err := os.ReadFile(filePath)
	assert.Nil(t, err, "ReadFile error for %v: %v", filePath, err)

	var testFile connectionStringTestFile
	vr, err := bson.NewExtJSONValueReader(bytes.NewReader(content), false)
	assert.Nil(t, err, "NewExtJSONValueReader error: %v", err)
	dec := bson.NewDecoder(vr)
	dec.SetRegistry(specTestRegistry)
	err = dec.Decode(&testFile)
	assert.Nil(t, err, "decode error: %v", err)

	for _, test := range testFile.Tests {
		t.Run(test.Description, func(t *testing.T) {
			runConnectionStringTest(t, test)
		})
	}
}

func runConnectionStringTest(t *testing.T, test connectionStringTest) {
	if test.SkipReason != "" {
		t.Skip(test.SkipReason)
	}

	cs, err := connstring.ParseAndValidate(test.URI)
	if !test.Valid {
		assert.NotNil(t, err, "expected Parse error, got nil")
		return
	}
	assert.Nil(t, err, "Parse error: %v", err)

	if test.ReadConcern != nil {
		expected := readConcernFromRaw(t, test.ReadConcern)
		assert.Equal(t, expected.Level, cs.ReadConcernLevel,
			"expected level %v, got %v", expected.Level, cs.ReadConcernLevel)
	}
	if test.WriteConcern != nil {
		expectedWc := writeConcernFromRaw(t, test.WriteConcern)
		if expectedWc.wSet {
			expected := expectedWc.W
			if _, ok := expected.(int); ok {
				assert.True(t, cs.WNumberSet, "expected WNumberSet, got false")
				assert.Equal(t, expected, cs.WNumber, "expected w value %v, got %v", expected, cs.WNumber)
			} else {
				assert.False(t, cs.WNumberSet, "expected WNumberSet to be false, got true")
				assert.Equal(t, expected, cs.WString, "expected w value %v, got %v", expected, cs.WString)
			}
		}
		if expectedWc.jSet {
			assert.True(t, cs.JSet, "expected JSet, got false")
			assert.Equal(t, *expectedWc.Journal, cs.J, "expected j value %v, got %v", *expectedWc.Journal, cs.J)
		}
	}
}

func runDocumentTestFile(t *testing.T, filePath string) {
	content, err := os.ReadFile(filePath)
	assert.Nil(t, err, "ReadFile error: %v", err)

	var testFile documentTestFile
	vr, err := bson.NewExtJSONValueReader(bytes.NewReader(content), false)
	assert.Nil(t, err, "NewExtJSONValueReader error: %v", err)
	dec := bson.NewDecoder(vr)
	dec.SetRegistry(specTestRegistry)
	err = dec.Decode(&testFile)
	assert.Nil(t, err, "decode error: %v", err)

	for _, test := range testFile.Tests {
		t.Run(test.Description, func(t *testing.T) {
			runDocumentTest(t, test)
		})
	}
}

func marshalWC(t *testing.T, wc *writeConcern) ([]byte, error) {
	t.Helper()

	var elems []byte
	if wc.W != nil {
		// Only support string or int values for W. That aligns with the
		// documentation and the behavior of other functions, like Acknowledged.
		switch w := wc.W.(type) {
		case int:
			if w < 0 {
				return nil, errors.New("write concern `w` field cannot be a negative number")
			}

			// If Journal=true and W=0, return an error because that write
			// concern is ambiguous.
			if wc.Journal != nil && *wc.Journal && w == 0 {
				return nil, errors.New("a write concern cannot have both w=0 and j=true")
			}

			if w > math.MaxInt32 {
				return nil, fmt.Errorf("%d overflows int32", w)
			}
			elems = bsoncore.AppendInt32Element(elems, "w", int32(w))
		case string:
			elems = bsoncore.AppendStringElement(elems, "w", w)
		default:
			return nil,
				fmt.Errorf("WriteConcern.W must be a string or int, but is a %T", wc.W)
		}
	}

	if wc.Journal != nil {
		elems = bsoncore.AppendBooleanElement(elems, "j", *wc.Journal)
	}

	return elems, nil
}

func runDocumentTest(t *testing.T, test documentTest) {
	if test.ReadConcern != nil {
		rc := readConcernFromRaw(t, test.ReadConcern)

		var elems []byte
		if len(rc.Level) > 0 {
			elems = bsoncore.AppendStringElement(elems, "level", rc.Level)
		}
		actual := bsoncore.BuildDocument(nil, elems)

		if !test.Valid {
			assert.Fail(t, "expected an invalid read concern")
		} else {
			compareDocuments(t, *test.ReadConcernDocument, actual)
		}

		if test.IsServerDefault != nil {
			gotServerDefault := bytes.Equal(actual, serverDefaultConcern)
			assert.Equal(t, *test.IsServerDefault, gotServerDefault, "expected server default read concern, got %s", actual)
		}
	}
	if test.WriteConcern != nil {
		actualWc := writeConcernFromRaw(t, test.WriteConcern)
		actualElems, err := marshalWC(t, &actualWc)
		if !test.Valid {
			assert.Equal(t, 0, len(actualElems), "expected an invalid write concern")
			return
		}
		if test.IsAcknowledged != nil {
			actualAck := actualWc.Acknowledged()
			assert.Equal(t, *test.IsAcknowledged, actualAck,
				"expected acknowledged %v, got %v", *test.IsAcknowledged, actualAck)
		}

		assert.Nil(t, err, "MarshalBSONValue error: %v", err)

		expected := *test.WriteConcernDocument
		if len(actualElems) == 0 {
			elems, _ := expected.Elements()
			if len(elems) == 0 {
				assert.NotNil(t, test.IsServerDefault, "expected write concern %s, got empty", expected)
				assert.True(t, *test.IsServerDefault, "expected write concern %s, got empty", expected)
				return
			}
			if _, jErr := expected.LookupErr("j"); jErr == nil && len(elems) == 1 {
				return
			}
			assert.Fail(t, "got empty elements")
		}

		actual := bsoncore.BuildDocument(nil, actualElems)
		if jVal, err := expected.LookupErr("j"); err == nil && !jVal.Boolean() {
			actual = actual[:len(actual)-1]
			actual = bsoncore.AppendBooleanElement(actual, "j", false)
			actual, _ = bsoncore.AppendDocumentEnd(actual, 0)
		}
		compareDocuments(t, expected, actual)
	}
}

func readConcernFromRaw(t *testing.T, rc bson.Raw) *readconcern.ReadConcern {
	t.Helper()

	concern := &readconcern.ReadConcern{}
	elems, _ := rc.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "level":
			concern.Level = val.StringValue()
		default:
			t.Fatalf("unrecognized read concern field %v", key)
		}
	}
	return concern
}

type writeConcern struct {
	*writeconcern.WriteConcern
	jSet bool
	wSet bool
}

func writeConcernFromRaw(t *testing.T, wcRaw bson.Raw) writeConcern {
	var wc writeConcern
	wc.WriteConcern = &writeconcern.WriteConcern{}

	elems, _ := wcRaw.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "w":
			wc.wSet = true
			switch val.Type {
			case bson.TypeInt32:
				w := int(val.Int32())
				wc.WriteConcern.W = w
			case bson.TypeString:
				wc.WriteConcern.W = val.StringValue()
			default:
				t.Fatalf("unexpected type for w: %v", val.Type)
			}
		case "journal":
			wc.jSet = true
			j := val.Boolean()
			wc.WriteConcern.Journal = &j
		case "wtimeoutMS": // Do nothing, this field is deprecated
			t.Skip("the wtimeoutMS write concern option is not supported")
		default:
			t.Fatalf("unrecognized write concern field: %v", key)
		}
	}
	return wc
}

// generate a slice of all JSON file names in a directory
func jsonFilesInDir(t *testing.T, dir string) []string {
	t.Helper()

	files := make([]string, 0)

	entries, err := os.ReadDir(dir)
	assert.Nil(t, err, "unable to read json file: %v", err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}
