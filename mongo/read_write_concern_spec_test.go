// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"errors"
	"os"
	"path"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/spectest"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

const (
	connstringTestsDir = "connection-string"
	documentTestsDir   = "document"
)

var (
	serverDefaultConcern = []byte{5, 0, 0, 0, 0} // server default read concern and write concern is empty document
	specTestRegistry     = func() *bson.Registry {
		reg := bson.NewRegistry()
		reg.RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(bson.Raw{}))
		return reg
	}()
	readWriteConcernTestsDir = spectest.Path("read-write-concern/tests")
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
	spectest.CheckSkip(t)

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

func runDocumentTest(t *testing.T, test documentTest) {
	if test.ReadConcern != nil {
		_, actual, err := driver.MarshalBSONReadConcern(readConcernFromRaw(t, test.ReadConcern))
		if !test.Valid {
			assert.NotNil(t, err, "expected an invalid read concern")
		} else {
			assert.Nil(t, err, "got error: %v", err)
			compareDocuments(t, *test.ReadConcernDocument, actual)
		}

		if test.IsServerDefault != nil {
			gotServerDefault := bytes.Equal(actual, serverDefaultConcern)
			assert.Equal(t, *test.IsServerDefault, gotServerDefault, "expected server default read concern, got %s", actual)
		}
	}
	if test.WriteConcern != nil {
		actualWc := writeConcernFromRaw(t, test.WriteConcern)
		_, actual, err := driver.MarshalBSONWriteConcern(actualWc.WriteConcern, 0)
		if !test.Valid {
			assert.NotNil(t, err, "expected an invalid write concern")
			return
		}
		if test.IsAcknowledged != nil {
			actualAck := actualWc.Acknowledged()
			assert.Equal(t, *test.IsAcknowledged, actualAck,
				"expected acknowledged %v, got %v", *test.IsAcknowledged, actualAck)
		}

		expected := *test.WriteConcernDocument
		if errors.Is(err, driver.ErrEmptyWriteConcern) {
			elems, _ := expected.Elements()
			if len(elems) == 0 {
				assert.NotNil(t, test.IsServerDefault, "expected write concern %s, got empty", expected)
				assert.True(t, *test.IsServerDefault, "expected write concern %s, got empty", expected)
				return
			}
			if _, jErr := expected.LookupErr("j"); jErr == nil && len(elems) == 1 {
				return
			}
		}

		assert.Nil(t, err, "MarshalBSONValue error: %v", err)
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
				wc.W = w
			case bson.TypeString:
				wc.W = val.StringValue()
			default:
				t.Fatalf("unexpected type for w: %v", val.Type)
			}
		case "journal":
			wc.jSet = true
			j := val.Boolean()
			wc.Journal = &j
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
