// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

type encodetest struct {
	Field1String  string
	Field1Int64   int64
	Field1Float64 float64
	Field2String  string
	Field2Int64   int64
	Field2Float64 float64
	Field3String  string
	Field3Int64   int64
	Field3Float64 float64
	Field4String  string
	Field4Int64   int64
	Field4Float64 float64
}

type nestedtest1 struct {
	Nested nestedtest2
}

type nestedtest2 struct {
	Nested nestedtest3
}

type nestedtest3 struct {
	Nested nestedtest4
}

type nestedtest4 struct {
	Nested nestedtest5
}

type nestedtest5 struct {
	Nested nestedtest6
}

type nestedtest6 struct {
	Nested nestedtest7
}

type nestedtest7 struct {
	Nested nestedtest8
}

type nestedtest8 struct {
	Nested nestedtest9
}

type nestedtest9 struct {
	Nested nestedtest10
}

type nestedtest10 struct {
	Nested nestedtest11
}

type nestedtest11 struct {
	Nested encodetest
}

var encodetestInstance = encodetest{
	Field1String:  "foo",
	Field1Int64:   1,
	Field1Float64: 3.0,
	Field2String:  "bar",
	Field2Int64:   2,
	Field2Float64: 3.1,
	Field3String:  "baz",
	Field3Int64:   3,
	Field3Float64: 3.14,
	Field4String:  "qux",
	Field4Int64:   4,
	Field4Float64: 3.141,
}

var nestedInstance = nestedtest1{
	nestedtest2{
		nestedtest3{
			nestedtest4{
				nestedtest5{
					nestedtest6{
						nestedtest7{
							nestedtest8{
								nestedtest9{
									nestedtest10{
										nestedtest11{
											encodetest{
												Field1String:  "foo",
												Field1Int64:   1,
												Field1Float64: 3.0,
												Field2String:  "bar",
												Field2Int64:   2,
												Field2Float64: 3.1,
												Field3String:  "baz",
												Field3Int64:   3,
												Field3Float64: 3.14,
												Field4String:  "qux",
												Field4Int64:   4,
												Field4Float64: 3.141,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

const extendedBSONDir = "../data/extended_bson"

// readExtJSONFile reads the GZIP-compressed extended JSON document from the given filename in the
// "extended BSON" test data directory (../data/extended_bson) and returns it as a
// map[string]interface{}. It panics on any errors.
func readExtJSONFile(filename string) map[string]interface{} {
	filePath := path.Join(extendedBSONDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("error opening file %q: %s", filePath, err))
	}
	defer func() {
		_ = file.Close()
	}()

	gz, err := gzip.NewReader(file)
	if err != nil {
		panic(fmt.Sprintf("error creating GZIP reader: %s", err))
	}
	defer func() {
		_ = gz.Close()
	}()

	data, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(fmt.Sprintf("error reading GZIP contents of file: %s", err))
	}

	var v map[string]interface{}
	err = UnmarshalExtJSON(data, false, &v)
	if err != nil {
		panic(fmt.Sprintf("error unmarshalling extended JSON: %s", err))
	}

	return v
}

func BenchmarkMarshal(b *testing.B) {
	cases := []struct {
		desc  string
		value interface{}
	}{
		{
			desc:  "simple struct",
			value: encodetestInstance,
		},
		{
			desc:  "nested struct",
			value: nestedInstance,
		},
		{
			desc:  "deep_bson.json.gz",
			value: readExtJSONFile("deep_bson.json.gz"),
		},
		{
			desc:  "flat_bson.json.gz",
			value: readExtJSONFile("flat_bson.json.gz"),
		},
		{
			desc:  "full_bson.json.gz",
			value: readExtJSONFile("full_bson.json.gz"),
		},
	}

	for _, tc := range cases {
		b.Run(tc.desc, func(b *testing.B) {
			b.Run("BSON", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := Marshal(tc.value)
					if err != nil {
						b.Errorf("error marshalling BSON: %s", err)
					}
				}
			})

			b.Run("extJSON", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := MarshalExtJSON(tc.value, true, false)
					if err != nil {
						b.Errorf("error marshalling extended JSON: %s", err)
					}
				}
			})

			b.Run("JSON", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := json.Marshal(tc.value)
					if err != nil {
						b.Errorf("error marshalling JSON: %s", err)
					}
				}
			})
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	cases := []struct {
		desc  string
		value interface{}
	}{
		{
			desc:  "simple struct",
			value: encodetestInstance,
		},
		{
			desc:  "nested struct",
			value: nestedInstance,
		},
		{
			desc:  "deep_bson.json.gz",
			value: readExtJSONFile("deep_bson.json.gz"),
		},
		{
			desc:  "flat_bson.json.gz",
			value: readExtJSONFile("flat_bson.json.gz"),
		},
		{
			desc:  "full_bson.json.gz",
			value: readExtJSONFile("full_bson.json.gz"),
		},
	}

	for _, tc := range cases {
		b.Run(tc.desc, func(b *testing.B) {
			b.Run("BSON", func(b *testing.B) {
				data, err := Marshal(tc.value)
				if err != nil {
					b.Errorf("error marshalling BSON: %s", err)
					return
				}

				b.ResetTimer()
				var v2 map[string]interface{}
				for i := 0; i < b.N; i++ {
					err := Unmarshal(data, &v2)
					if err != nil {
						b.Errorf("error unmarshalling BSON: %s", err)
					}
				}
			})

			b.Run("extJSON", func(b *testing.B) {
				data, err := MarshalExtJSON(tc.value, true, false)
				if err != nil {
					b.Errorf("error marshalling extended JSON: %s", err)
					return
				}

				b.ResetTimer()
				var v2 map[string]interface{}
				for i := 0; i < b.N; i++ {
					err := UnmarshalExtJSON(data, true, &v2)
					if err != nil {
						b.Errorf("error unmarshalling extended JSON: %s", err)
					}
				}
			})

			b.Run("JSON", func(b *testing.B) {
				data, err := json.Marshal(tc.value)
				if err != nil {
					b.Errorf("error marshalling JSON: %s", err)
					return
				}

				b.ResetTimer()
				var v2 map[string]interface{}
				for i := 0; i < b.N; i++ {
					err := json.Unmarshal(data, &v2)
					if err != nil {
						b.Errorf("error unmarshalling JSON: %s", err)
					}
				}
			})
		})
	}
}
