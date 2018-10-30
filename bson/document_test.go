// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
)

func ExampleDocument() {
	internalVersion := "1234567"

	f := func(appName string) Doc {
		doc := Doc{
			{"driver", Document(Doc{{"name", String("mongo-go-driver")}, {"version", String(internalVersion)}})},
			{"os", Document(Doc{{"type", String("darwin")}, {"architecture", String("amd64")}})},
			{"platform", String("go1.11.1")},
		}
		if appName != "" {
			doc = append(doc, Elem{"application", Document(MDoc{"name": String(appName)})})
		}

		return doc
	}
	buf, err := f("hello-world").MarshalBSON()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf)

	// Output: [178 0 0 0 3 100 114 105 118 101 114 0 52 0 0 0 2 110 97 109 101 0 16 0 0 0 109 111 110 103 111 45 103 111 45 100 114 105 118 101 114 0 2 118 101 114 115 105 111 110 0 8 0 0 0 49 50 51 52 53 54 55 0 0 3 111 115 0 46 0 0 0 2 116 121 112 101 0 7 0 0 0 100 97 114 119 105 110 0 2 97 114 99 104 105 116 101 99 116 117 114 101 0 6 0 0 0 97 109 100 54 52 0 0 2 112 108 97 116 102 111 114 109 0 9 0 0 0 103 111 49 46 49 49 46 49 0 3 97 112 112 108 105 99 97 116 105 111 110 0 27 0 0 0 2 110 97 109 101 0 12 0 0 0 104 101 108 108 111 45 119 111 114 108 100 0 0 0]
}

func BenchmarkDocument(b *testing.B) {
	b.ReportAllocs()
	internalVersion := "1234567"
	for i := 0; i < b.N; i++ {
		doc := Doc{
			{"driver", Document(Doc{{"name", String("mongo-go-driver")}, {"version", String(internalVersion)}})},
			{"os", Document(Doc{{"type", String("darwin")}, {"architecture", String("amd64")}})},
			{"platform", String("go1.11.1")},
		}
		_, _ = doc.MarshalBSON()
	}
}

func valueEqual(v1, v2 Val) bool { return v1.Equal(v2) }

func elementEqual(e1, e2 Elem) bool { return e1.Equal(e2) }

func documentComparer(d1, d2 Doc) bool { return d1.Equal(d2) }

func TestDocument(t *testing.T) {
	t.Parallel()
	t.Run("ReadDocument", func(t *testing.T) {
		t.Parallel()
		t.Run("UnmarshalingError", func(t *testing.T) {
			t.Parallel()
			invalid := []byte{0x01, 0x02}
			want := bsoncore.NewInsufficientBytesError(nil, nil)
			_, got := ReadDoc(invalid)
			if !compareErrors(got, want) {
				t.Errorf("Expected errors to match. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			t.Parallel()
			valid := bsoncore.BuildDocument(nil, bsoncore.AppendNullElement(nil, "foobar"))
			var want error
			wantDoc := Doc{{"foobar", Null()}}
			gotDoc, got := ReadDoc(valid)
			if !compareErrors(got, want) {
				t.Errorf("Expected errors to match. got %v; want %v", got, want)
			}
			if !cmp.Equal(gotDoc, wantDoc) {
				t.Errorf("Expected returned documents to match. got %v; want %v", gotDoc, wantDoc)
			}
		})
	})
	t.Run("Copy", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name  string
			start Doc
			copy  Doc
		}{
			{"nil", nil, Doc{}},
			{"not-nil", Doc{{"foobar", Null()}}, Doc{{"foobar", Null()}}},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				copy := tc.start.Copy()
				if !cmp.Equal(copy, tc.copy) {
					t.Errorf("Expected copies to be equal. got %v; want %v", copy, tc.copy)
				}
			})
		}
	})
	t.Run("Append-Prepend-Lookup-Delete", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name   string
			fn     interface{}   // method to call
			params []interface{} // parameters
			rets   []interface{} // returns
		}{
			{
				"Append", Doc{}.Append,
				[]interface{}{"foo", Null()},
				[]interface{}{Doc{{"foo", Null()}}},
			},
			{
				"Prepend", Doc{}.Prepend,
				[]interface{}{"foo", Null()},
				[]interface{}{Doc{{"foo", Null()}}},
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				params := make([]reflect.Value, 0, len(tc.params))
				for _, param := range tc.params {
					params = append(params, reflect.ValueOf(param))
				}
				fn := reflect.ValueOf(tc.fn)
				if fn.Kind() != reflect.Func {
					t.Fatalf("property fn must be a function, but it is a %v", fn.Kind())
				}
				if fn.Type().NumIn() != len(params) && !fn.Type().IsVariadic() {
					t.Fatalf("number of parameters does not match. fn takes %d, but was provided %d", fn.Type().NumIn(), len(params))
				}
				var rets []reflect.Value
				if fn.Type().IsVariadic() {
					rets = fn.CallSlice(params)
				} else {
					rets = fn.Call(params)
				}
				if len(rets) != len(tc.rets) {
					t.Fatalf("mismatched number of returns. recieved %d; expected %d", len(rets), len(tc.rets))
				}
				for idx := range rets {
					got, want := rets[idx].Interface(), tc.rets[idx]
					if !cmp.Equal(got, want) {
						t.Errorf("Return %d does not match. got %v; want %v", idx, got, want)
					}
				}
			})
		}
	})
}
