// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: (%T)%v", err, err)
		t.FailNow()
	}
}

func requireErrEqual(t *testing.T, err1 error, err2 error) { require.True(t, compareErrors(err1, err2)) }

func TestTimeRoundTrip(t *testing.T) {
	val := struct {
		Value time.Time
		ID    string
	}{
		ID: "time-rt-test",
	}

	if !val.Value.IsZero() {
		t.Errorf("Did not get zero time as expected.")
	}

	bsonOut, err := Marshal(val)
	noerr(t, err)
	rtval := struct {
		Value time.Time
		ID    string
	}{}

	err = Unmarshal(bsonOut, &rtval)
	noerr(t, err)
	if !cmp.Equal(val, rtval) {
		t.Errorf("Did not round trip properly. got %v; want %v", val, rtval)
	}
	if !rtval.Value.IsZero() {
		t.Errorf("Did not get zero time as expected.")
	}
}

func TestNonNullTimeRoundTrip(t *testing.T) {
	now := time.Now()
	now = time.Unix(now.Unix(), 0)
	val := struct {
		Value time.Time
		ID    string
	}{
		ID:    "time-rt-test",
		Value: now,
	}

	bsonOut, err := Marshal(val)
	noerr(t, err)
	rtval := struct {
		Value time.Time
		ID    string
	}{}

	err = Unmarshal(bsonOut, &rtval)
	noerr(t, err)
	if !cmp.Equal(val, rtval) {
		t.Errorf("Did not round trip properly. got %v; want %v", val, rtval)
	}
}

func TestD(t *testing.T) {
	t.Run("can marshal", func(t *testing.T) {
		d := D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
		idx, want := bsoncore.AppendDocumentStart(nil)
		want = bsoncore.AppendStringElement(want, "foo", "bar")
		want = bsoncore.AppendStringElement(want, "hello", "world")
		want = bsoncore.AppendDoubleElement(want, "pi", 3.14159)
		want, err := bsoncore.AppendDocumentEnd(want, idx)
		noerr(t, err)
		got, err := Marshal(d)
		noerr(t, err)
		if !bytes.Equal(got, want) {
			t.Errorf("Marshaled documents do not match. got %v; want %v", Raw(got), Raw(want))
		}
	})
	t.Run("can unmarshal", func(t *testing.T) {
		want := D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
		idx, doc := bsoncore.AppendDocumentStart(nil)
		doc = bsoncore.AppendStringElement(doc, "foo", "bar")
		doc = bsoncore.AppendStringElement(doc, "hello", "world")
		doc = bsoncore.AppendDoubleElement(doc, "pi", 3.14159)
		doc, err := bsoncore.AppendDocumentEnd(doc, idx)
		noerr(t, err)
		var got D
		err = Unmarshal(doc, &got)
		noerr(t, err)
		if !cmp.Equal(got, want) {
			t.Errorf("Unmarshaled documents do not match. got %v; want %v", got, want)
		}
	})
}

func TestExtJSONEscapeKey(t *testing.T) {
	doc := D{{Key: "\\usb#", Value: int32(1)}}
	b, err := MarshalExtJSON(&doc, false, false)
	noerr(t, err)

	want := "{\"\\\\usb#\":1}"
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("Marshaled documents do not match. got %v, want %v", string(b), want)
	}

	var got D
	err = UnmarshalExtJSON(b, false, &got)
	noerr(t, err)
	if !cmp.Equal(got, doc) {
		t.Errorf("Unmarshaled documents do not match. got %v; want %v", got, doc)
	}
}

func TestDepthTracking(t *testing.T) {
	t.Run("marshal bson and extjson", func(t *testing.T) {
		// Test the D, M, and [1]E types. All of these types become documents. For each type, assert that values of depth
		// MaxBSONDepth-1 and MaxBSONDepth can be marshalled and a value of depth MaxBSONDepth + 1 fails. Documents in this
		// section have one element and are in the form {x: {x: {x: ... {x: 1}}}}
		shallowD := createNestedD(bsonrw.MaxBSONDepth - 1)
		maxDepthD := createNestedD(bsonrw.MaxBSONDepth)
		overflowD := createNestedD(bsonrw.MaxBSONDepth + 1)

		shallowM := createNestedM(bsonrw.MaxBSONDepth - 1)
		maxDepthM := createNestedM(bsonrw.MaxBSONDepth)
		overflowM := createNestedM(bsonrw.MaxBSONDepth + 1)

		var shallowArrayOfE [1]E
		var maxDepthArrayOfE [1]E
		var overflowArrayofE [1]E
		copy(shallowArrayOfE[:], shallowD)
		copy(maxDepthArrayOfE[:], maxDepthD)
		copy(overflowArrayofE[:], overflowD)

		// Test nested arrays. We pass one less than the intended depth to createNestedArray because the test adds an
		// additional level of depth by wrapping the returned array in a document. Documents in this section have one
		// element and are in the form {x: [[[[[[[... [1] ...]]]]]]]}.
		shallowDocOfArrays := D{{"x", createNestedArray(bsonrw.MaxBSONDepth - 2)}}
		maxDepthDocOfArrays := D{{"x", createNestedArray(bsonrw.MaxBSONDepth - 1)}}
		overflowDocOfArrays := D{{"x", createNestedArray(bsonrw.MaxBSONDepth)}}

		// Test a value that has a cycle.
		type cycle struct {
			A     int
			Cycle *cycle
		}
		cycleValue := cycle{
			A: 10,
		}
		cycleValue.Cycle = &cycleValue

		// Test documents with many values and no nesting. Documents in this section are of length MaxBSONDepth + 10
		// and their values are types that will force the nesting depth to be incremented in the ValueWriter. This is
		// testing that the ValueWriter decrements the nesting depth when an array or document is closed. If it does not,
		// these would fail with ErrMaxDepthExceeded.
		largeShallowDocOfDocs := createShallowDoc(bsonrw.MaxBSONDepth+10, D{{"x", 1}})
		largeShallowDocOfArrays := createShallowDoc(bsonrw.MaxBSONDepth+10, A{1})

		testCases := []struct {
			name        string
			val         interface{}
			expectError bool
		}{
			{"shallow D", shallowD, false},
			{"max depth D", maxDepthD, false},
			{"overflow D", overflowD, true},
			{"shallow M", shallowM, false},
			{"max depth M", maxDepthM, false},
			{"overflow M", overflowM, true},
			{"shallow array of E", shallowArrayOfE, false},
			{"max depth array of E", maxDepthArrayOfE, false},
			{"overflow array of E", overflowArrayofE, true},
			{"shallow doc of nested arrays", shallowDocOfArrays, false},
			{"max depth doc of nested arrays", maxDepthDocOfArrays, false},
			{"overflow doc of nested arrays", overflowDocOfArrays, true},
			{"cyclical struct", cycleValue, true},
			{"large shallow doc of docs", largeShallowDocOfDocs, false},
			{"large shallow doc of arrays", largeShallowDocOfArrays, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, bsonMarshalErr := Marshal(tc.val)
				_, jsonErr := MarshalExtJSON(tc.val, true, false)

				marshalErrors := []error{
					bsonMarshalErr,
					jsonErr,
				}

				for idx, err := range marshalErrors {
					if tc.expectError {
						assert.Equal(t, bsonrw.ErrMaxDepthExceeded, err, "expected error %v at index %d, got %v",
							bsonrw.ErrMaxDepthExceeded, idx, err)
						return
					}

					assert.Nil(t, err, "Marshal error at index %d: %v", idx, err)
				}
			})
		}
	})
	t.Run("marshal non-document value", func(t *testing.T) {
		shallowArray := createNestedArray(bsonrw.MaxBSONDepth - 10)
		deepArray := createNestedArray(bsonrw.MaxBSONDepth + 10)

		testCases := []struct {
			name      string
			val       interface{}
			expectErr bool
		}{
			{"shallow array succeeds", shallowArray, false},
			{"deep array succeeds", deepArray, true},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := MarshalValue(tc.val)
				if tc.expectErr {
					assert.Equal(t, bsonrw.ErrMaxDepthExceeded, err, "expected error %v, got %v",
						bsonrw.ErrMaxDepthExceeded, err)
					return
				}
				assert.Nil(t, err, "MarshalValue error: %v", err)
			})
		}
	})
}

// creates a document of the given depth in the form {x: {x: {... {x: 1}}}}
func createNestedD(depth int) D {
	doc := D{{"x", 1}}
	for i := 0; i < depth; i++ {
		doc = D{{"x", doc}}
	}

	return doc
}

// creates a document of the given depth in the form {x: {x: {... {x: 1}}}}
func createNestedM(depth int) M {
	docM := M{"x": 1}
	for i := 0; i < depth; i++ {
		docM = M{"x": docM}
	}

	return docM
}

// creates an array of the given depth in the form [[[[[[...1...]]]]]]
func createNestedArray(depth int) A {
	a := A{1}
	for i := 0; i < depth; i++ {
		a = A{a}
	}

	return a
}

// creates a document with the given number of elements in the form {x: innerVal, x: innerVal, x: innerVal, ...}
func createShallowDoc(numElements int, innerVal interface{}) D {
	var doc D
	for i := 0; i < numElements; i++ {
		doc = append(doc, E{"x", innerVal})
	}
	return doc
}
