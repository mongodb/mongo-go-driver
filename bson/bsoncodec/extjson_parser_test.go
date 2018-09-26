// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
)

var (
	keyDiff = specificDiff("key")
	typDiff = specificDiff("type")
	valDiff = specificDiff("value")

	expectErrEOD = expectSpecificError(ErrEOD)
	expectErrEOA = expectSpecificError(ErrEOA)
)

type expectedErrorFunc func(t *testing.T, err error, desc string)

type peekTypeTestCase struct {
	desc  string
	input string
	typs  []bson.Type
	errFs []expectedErrorFunc
}

type readKeyValueTestCase struct {
	desc  string
	input string
	keys  []string
	typs  []bson.Type
	vals  []*extJSONValue

	keyEFs []expectedErrorFunc
	valEFs []expectedErrorFunc
}

func expectSpecificError(expected error) expectedErrorFunc {
	return func(t *testing.T, err error, desc string) {
		if err != expected {
			t.Helper()
			t.Errorf("%s: Expected %v but got: %v", desc, expected, err)
			t.FailNow()
		}
	}
}

func specificDiff(name string) func(t *testing.T, expected, actual interface{}, desc string) {
	return func(t *testing.T, expected, actual interface{}, desc string) {
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Helper()
			t.Errorf("%s: Incorrect JSON %s (-want, +got): %s\n", desc, name, diff)
			t.FailNow()
		}
	}
}

func expectErrorNOOP(_ *testing.T, _ error, _ string) {
}

func readKeyDiff(t *testing.T, eKey, aKey string, eTyp, aTyp bson.Type, err error, errF expectedErrorFunc, desc string) {
	keyDiff(t, eKey, aKey, desc)
	typDiff(t, eTyp, aTyp, desc)
	errF(t, err, desc)
}

func readValueDiff(t *testing.T, eVal, aVal *extJSONValue, err error, errF expectedErrorFunc, desc string) {
	if aVal != nil {
		typDiff(t, eVal.t, aVal.t, desc)
		valDiff(t, eVal.v, aVal.v, desc)
	} else {
		valDiff(t, eVal, aVal, desc)
	}

	errF(t, err, desc)
}

func TestExtJSONParserPeekType(t *testing.T) {
	makeValidPeekTypeTestCase := func(input string, typ bson.Type, desc string) peekTypeTestCase {
		return peekTypeTestCase{
			desc: desc, input: input,
			typs:  []bson.Type{typ},
			errFs: []expectedErrorFunc{expectNoError},
		}
	}

	makeInvalidPeekTypeTestCase := func(desc, input string, lastEF expectedErrorFunc) peekTypeTestCase {
		return peekTypeTestCase{
			desc: desc, input: input,
			typs:  []bson.Type{bson.TypeArray, bson.TypeString, bson.Type(0)},
			errFs: []expectedErrorFunc{expectNoError, expectNoError, lastEF},
		}
	}

	cases := []peekTypeTestCase{
		makeValidPeekTypeTestCase(`null`, bson.TypeNull, "Null"),
		makeValidPeekTypeTestCase(`"string"`, bson.TypeString, "String"),
		makeValidPeekTypeTestCase(`true`, bson.TypeBoolean, "Boolean--true"),
		makeValidPeekTypeTestCase(`false`, bson.TypeBoolean, "Boolean--false"),
		makeValidPeekTypeTestCase(`{"$minKey": 1}`, bson.TypeMinKey, "MinKey"),
		makeValidPeekTypeTestCase(`{"$maxKey": 1}`, bson.TypeMaxKey, "MaxKey"),
		makeValidPeekTypeTestCase(`{"$numberInt": "42"}`, bson.TypeInt32, "Int32"),
		makeValidPeekTypeTestCase(`{"$numberLong": "42"}`, bson.TypeInt64, "Int64"),
		makeValidPeekTypeTestCase(`{"$symbol": "symbol"}`, bson.TypeSymbol, "Symbol"),
		makeValidPeekTypeTestCase(`{"$numberDouble": "42.42"}`, bson.TypeDouble, "Double"),
		makeValidPeekTypeTestCase(`{"$undefined": true}`, bson.TypeUndefined, "Undefined"),
		makeValidPeekTypeTestCase(`{"$numberDouble": "NaN"}`, bson.TypeDouble, "Double--NaN"),
		makeValidPeekTypeTestCase(`{"$numberDecimal": "1234"}`, bson.TypeDecimal128, "Decimal"),
		makeValidPeekTypeTestCase(`{"foo": "bar"}`, bson.TypeEmbeddedDocument, "Toplevel document"),
		makeValidPeekTypeTestCase(`{"$date": {"$numberLong": "0"}}`, bson.TypeDateTime, "Datetime"),
		makeValidPeekTypeTestCase(`{"$code": "function() {}"}`, bson.TypeJavaScript, "Code no scope"),
		makeValidPeekTypeTestCase(`[{"$numberInt": "1"},{"$numberInt": "2"}]`, bson.TypeArray, "Array"),
		makeValidPeekTypeTestCase(`{"$timestamp": {"t": 42, "i": 1}}`, bson.TypeTimestamp, "Timestamp"),
		makeValidPeekTypeTestCase(`{"$oid": "57e193d7a9cc81b4027498b5"}`, bson.TypeObjectID, "Object ID"),
		makeValidPeekTypeTestCase(`{"$binary": {"base64": "AQIDBAU=", "subType": "80"}}`, bson.TypeBinary, "Binary"),
		makeValidPeekTypeTestCase(`{"$code": "function() {}", "$scope": {}}`, bson.TypeCodeWithScope, "Code With Scope"),
		makeValidPeekTypeTestCase(`{"$binary": {"base64": "o0w498Or7cijeBSpkquNtg==", "subType": "03"}}`, bson.TypeBinary, "Binary"),
		makeValidPeekTypeTestCase(`{"$regularExpression": {"pattern": "foo*", "options": "ix"}}`, bson.TypeRegex, "Regular expression"),
		makeValidPeekTypeTestCase(`{"$dbPointer": {"$ref": "db.collection", "$id": {"$oid": "57e193d7a9cc81b4027498b1"}}}`, bson.TypeDBPointer, "DBPointer"),
		makeValidPeekTypeTestCase(`{"$ref": "collection", "$id": {"$oid": "57fd71e96e32ab4225b723fb"}, "$db": "database"}`, bson.TypeEmbeddedDocument, "DBRef"),
		makeInvalidPeekTypeTestCase("invalid array--missing ]", `["a"`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--colon in array", `["a":`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--extra comma", `["a",,`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--trailing comma", `["a",]`, expectError),
		makeInvalidPeekTypeTestCase("peekType after end of array", `["a"]`, expectErrEOA),
		{
			desc:  "invalid array--leading comma",
			input: `[,`,
			typs:  []bson.Type{bson.TypeArray, bson.Type(0)},
			errFs: []expectedErrorFunc{expectNoError, expectError},
		},
	}

	for _, tc := range cases {
		ejp := newExtJSONParser(strings.NewReader(tc.input), true)

		for i, eTyp := range tc.typs {
			errF := tc.errFs[i]

			typ, err := ejp.peekType()
			typDiff(t, eTyp, typ, tc.desc)
			errF(t, err, tc.desc)
		}
	}
}

func TestExtJSONParserReadKeyReadValue(t *testing.T) {
	// several test cases will use the same keys, types, and values, and only differ on input structure

	keys := []string{"_id", "Symbol", "String", "Int32", "Int64", "Int", "MinKey"}
	types := []bson.Type{bson.TypeObjectID, bson.TypeSymbol, bson.TypeString, bson.TypeInt32, bson.TypeInt64, bson.TypeInt32, bson.TypeMinKey}
	values := []*extJSONValue{
		{t: bson.TypeString, v: "57e193d7a9cc81b4027498b5"},
		{t: bson.TypeString, v: "symbol"},
		{t: bson.TypeString, v: "string"},
		{t: bson.TypeString, v: "42"},
		{t: bson.TypeString, v: "42"},
		{t: bson.TypeInt32, v: int32(42)},
		{t: bson.TypeInt32, v: int32(1)},
	}

	errFuncs := make([]expectedErrorFunc, 7)
	for i := 0; i < 7; i++ {
		errFuncs[i] = expectNoError
	}

	firstKeyError := func(desc, input string) readKeyValueTestCase {
		return readKeyValueTestCase{
			desc:   desc,
			input:  input,
			keys:   []string{""},
			typs:   []bson.Type{bson.Type(0)},
			vals:   []*extJSONValue{nil},
			keyEFs: []expectedErrorFunc{expectError},
			valEFs: []expectedErrorFunc{expectErrorNOOP},
		}
	}

	secondKeyError := func(desc, input, firstKey string, firstType bson.Type, firstValue *extJSONValue) readKeyValueTestCase {
		return readKeyValueTestCase{
			desc:   desc,
			input:  input,
			keys:   []string{firstKey, ""},
			typs:   []bson.Type{firstType, bson.Type(0)},
			vals:   []*extJSONValue{firstValue, nil},
			keyEFs: []expectedErrorFunc{expectNoError, expectError},
			valEFs: []expectedErrorFunc{expectNoError, expectErrorNOOP},
		}
	}

	cases := []readKeyValueTestCase{
		{
			desc: "normal spacing",
			input: `{
					"_id": { "$oid": "57e193d7a9cc81b4027498b5" },
					"Symbol": { "$symbol": "symbol" },
					"String": "string",
					"Int32": { "$numberInt": "42" },
					"Int64": { "$numberLong": "42" },
					"Int": 42,
					"MinKey": { "$minKey": 1 }
				}`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc: "single quoted strings",
			input: `{
					'_id': { '$oid': '57e193d7a9cc81b4027498b5' },
					'Symbol': { '$symbol': 'symbol' },
					'String': 'string',
					'Int32': { '$numberInt': '42' },
					'Int64': { '$numberLong': '42' },
					'Int': 42,
					'MinKey': { '$minKey': 1 }
				}`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc: "unquoted keys",
			input: `{
					_id: { "$oid": "57e193d7a9cc81b4027498b5" },
					Symbol: { "$symbol": "symbol" },
					String: "string",
					Int32: { "$numberInt": "42" },
					Int64: { "$numberLong": "42" },
					Int: 42,
					MinKey: { "$minKey": 1 }
				}`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc: "new line before comma",
			input: `{ "_id": { "$oid": "57e193d7a9cc81b4027498b5" }
				 , "Symbol": { "$symbol": "symbol" }
				 , "String": "string"
				 , "Int32": { "$numberInt": "42" }
				 , "Int64": { "$numberLong": "42" }
				 , "Int": 42
				 , "MinKey": { "$minKey": 1 }
				 }`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc: "tabs around colons",
			input: `{
					"_id":    { "$oid"       : "57e193d7a9cc81b4027498b5" },
					"Symbol": { "$symbol"    : "symbol" },
					"String": "string",
					"Int32":  { "$numberInt" : "42" },
					"Int64":  { "$numberLong": "42" },
					"Int":    42,
					"MinKey": { "$minKey": 1 }
				}`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc:  "no whitespace",
			input: `{"_id":{"$oid":"57e193d7a9cc81b4027498b5"},"Symbol":{"$symbol":"symbol"},"String":"string","Int32":{"$numberInt":"42"},"Int64":{"$numberLong":"42"},"Int":42,"MinKey":{"$minKey":1}}`,
			keys:  keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc: "mixed whitespace",
			input: `	{
					"_id"		: { "$oid": "57e193d7a9cc81b4027498b5" },
			        "Symbol"	: { "$symbol": "symbol" }	,
				    "String"	: "string",
					"Int32"		: { "$numberInt": "42" }    ,
					"Int64"		: {"$numberLong" : "42"},
					"Int"		: 42,
			      	"MinKey"	: { "$minKey": 1 } 	}	`,
			keys: keys, typs: types, vals: values,
			keyEFs: errFuncs, valEFs: errFuncs,
		},
		{
			desc:  "nested object",
			input: `{"k1": 1, "k2": { "k3": { "k4": 4 } }, "k5": 5}`,
			keys:  []string{"k1", "k2", "k3", "k4", "", "", "k5", ""},
			typs:  []bson.Type{bson.TypeInt32, bson.TypeEmbeddedDocument, bson.TypeEmbeddedDocument, bson.TypeInt32, bson.Type(0), bson.Type(0), bson.TypeInt32, bson.Type(0)},
			vals: []*extJSONValue{
				{t: bson.TypeInt32, v: int32(1)}, nil, nil, {t: bson.TypeInt32, v: int32(4)}, nil, nil, {t: bson.TypeInt32, v: int32(5)}, nil,
			},
			keyEFs: []expectedErrorFunc{
				expectNoError, expectNoError, expectNoError, expectNoError, expectErrEOD,
				expectErrEOD, expectNoError, expectErrEOD,
			},
			valEFs: []expectedErrorFunc{
				expectNoError, expectError, expectError, expectNoError, expectErrorNOOP,
				expectErrorNOOP, expectNoError, expectErrorNOOP,
			},
		},
		{
			desc:   "invalid input: invalid values for extended type",
			input:  `{"a": {"$numberInt": "1", "x"`,
			keys:   []string{"a"},
			typs:   []bson.Type{bson.TypeInt32},
			vals:   []*extJSONValue{nil},
			keyEFs: []expectedErrorFunc{expectNoError},
			valEFs: []expectedErrorFunc{expectError},
		},
		firstKeyError("invalid input: missing key--EOF", "{"),
		firstKeyError("invalid input: missing key--colon first", "{:"),
		firstKeyError("invalid input: missing value", `{"a":`),
		firstKeyError("invalid input: missing colon", `{"a" 1`),
		firstKeyError("invalid input: extra colon", `{"a"::`),
		secondKeyError("invalid input: missing }", `{"a": 1`, "a", bson.TypeInt32, &extJSONValue{t: bson.TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: missing comma", `{"a": 1 "b"`, "a", bson.TypeInt32, &extJSONValue{t: bson.TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: extra comma", `{"a": 1,, "b"`, "a", bson.TypeInt32, &extJSONValue{t: bson.TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: trailing comma in object", `{"a": 1,}`, "a", bson.TypeInt32, &extJSONValue{t: bson.TypeInt32, v: int32(1)}),
	}

	for _, tc := range cases {
		ejp := newExtJSONParser(strings.NewReader(tc.input), true)

		for i, eKey := range tc.keys {
			eTyp := tc.typs[i]
			eVal := tc.vals[i]

			keyErrF := tc.keyEFs[i]
			valErrF := tc.valEFs[i]

			k, typ, err := ejp.readKey()
			readKeyDiff(t, eKey, k, eTyp, typ, err, keyErrF, tc.desc)

			v, err := ejp.readValue(typ)
			readValueDiff(t, eVal, v, err, valErrF, tc.desc)
		}
	}
}

type ejpExpectationTest func(t *testing.T, p *extJSONParser, expectedKey string, expectedType bson.Type, expectedValue interface{})

type ejpTestCase struct {
	f ejpExpectationTest
	p *extJSONParser
	k string
	t bson.Type
	v interface{}
}

// expectSingleValue is used for simple JSON types (strings, numbers, literals) and for extended JSON types that
// have single key-value pairs (i.e. { "$minKey": 1 }, { "$numberLong": "42.42" })
func expectSingleValue(t *testing.T, p *extJSONParser, expectedKey string, expectedType bson.Type, expectedValue interface{}) {
	eVal := expectedValue.(*extJSONValue)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	v, err := p.readValue(typ)
	readValueDiff(t, eVal, v, err, expectNoError, expectedKey)
}

// expectMultipleValues is used for values that are subdocuments of known size and with known keys (such as extended
// JSON types { "$timestamp": {"t": 1, "i": 1} } and { "$regularExpression": {"pattern": "", options: ""} })
func expectMultipleValues(t *testing.T, p *extJSONParser, expectedKey string, expectedType bson.Type, expectedValue interface{}) {
	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	v, err := p.readValue(typ)
	typDiff(t, bson.TypeEmbeddedDocument, v.t, expectedKey)
	expectNoError(t, err, "")

	actObj := v.v.(*extJSONObject)
	expObj := expectedValue.(*extJSONObject)

	for i, actKey := range actObj.keys {
		expKey := expObj.keys[i]
		actVal := actObj.values[i]
		expVal := expObj.values[i]

		keyDiff(t, expKey, actKey, expectedKey)
		typDiff(t, expVal.t, actVal.t, expectedKey)
		valDiff(t, expVal.v, actVal.v, expectedKey)
	}
}

type ejpKeyTypValTriple struct {
	key string
	typ bson.Type
	val *extJSONValue
}

type ejpSubDocumentTestValue struct {
	code string               // code is only used for TypeCodeWithScope (and is ignored for TypeEmbeddedDocument
	ktvs []ejpKeyTypValTriple // list of (key, type, value) triples; this is "scope" for TypeCodeWithScope
}

// expectSubDocument is used for embedded documents and code with scope types; it reads all the keys and values
// in the embedded document (or scope for codeWithScope) and compares them to the expectedValue's list of (key, type,
// value) triples
func expectSubDocument(t *testing.T, p *extJSONParser, expectedKey string, expectedType bson.Type, expectedValue interface{}) {
	subdoc := expectedValue.(ejpSubDocumentTestValue)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	if expectedType == bson.TypeCodeWithScope {
		v, err := p.readValue(typ)
		readValueDiff(t, &extJSONValue{t: bson.TypeString, v: subdoc.code}, v, err, expectNoError, expectedKey)
	}

	for _, ktv := range subdoc.ktvs {
		eKey := ktv.key
		eTyp := ktv.typ
		eVal := ktv.val

		k, typ, err = p.readKey()
		readKeyDiff(t, eKey, k, eTyp, typ, err, expectNoError, expectedKey)

		v, err := p.readValue(typ)
		readValueDiff(t, eVal, v, err, expectNoError, expectedKey)
	}

	if expectedType == bson.TypeCodeWithScope {
		// expect scope doc to close
		k, typ, err = p.readKey()
		readKeyDiff(t, "", k, bson.Type(0), typ, err, expectErrEOD, expectedKey)
	}

	// expect subdoc to close
	k, typ, err = p.readKey()
	readKeyDiff(t, "", k, bson.Type(0), typ, err, expectErrEOD, expectedKey)
}

// expectArray takes the expectedKey, ignores the expectedType, and uses the expectedValue
// as a slice of (type Type, value *extJSONValue) pairs
func expectArray(t *testing.T, p *extJSONParser, expectedKey string, _ bson.Type, expectedValue interface{}) {
	ktvs := expectedValue.([]ejpKeyTypValTriple)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, bson.TypeArray, typ, err, expectNoError, expectedKey)

	for _, ktv := range ktvs {
		eTyp := ktv.typ
		eVal := ktv.val

		typ, err = p.peekType()
		typDiff(t, eTyp, typ, expectedKey)
		expectNoError(t, err, expectedKey)

		v, err := p.readValue(typ)
		readValueDiff(t, eVal, v, err, expectNoError, expectedKey)
	}

	// expect array to end
	typ, err = p.peekType()
	typDiff(t, bson.Type(0), typ, expectedKey)
	expectErrEOA(t, err, expectedKey)
}

func TestExtJSONParserAllTypes(t *testing.T) {
	in := ` { "_id"					: { "$oid": "57e193d7a9cc81b4027498b5"}
			, "Symbol"				: { "$symbol": "symbol"}
			, "String"				: "string"
			, "Int32"				: { "$numberInt": "42"}
			, "Int64"				: { "$numberLong": "42"}
			, "Double"				: { "$numberDouble": "42.42"}
			, "SpecialFloat"		: { "$numberDouble": "NaN" }
			, "Decimal"				: { "$numberDecimal": "1234" }
			, "Binary"			 	: { "$binary": { "base64": "o0w498Or7cijeBSpkquNtg==", "subType": "03" } }
			, "BinaryUserDefined"	: { "$binary": { "base64": "AQIDBAU=", "subType": "80" } }
			, "Code"				: { "$code": "function() {}" }
			, "CodeWithEmptyScope"	: { "$code": "function() {}", "$scope": {} }
			, "CodeWithScope"		: { "$code": "function() {}", "$scope": { "x": 1 } }
			, "Subdocument"			: { "foo": "bar", "baz": { "$numberInt": "42" } }
			, "Array"				: [{"$numberInt": "1"}, {"$numberLong": "2"}, {"$numberDouble": "3"}, 4, 5.0]
			, "Timestamp"			: { "$timestamp": { "t": 42, "i": 1 } }
			, "RegularExpression"	: { "$regularExpression": { "pattern": "foo*", "options": "ix" } }
			, "DatetimeEpoch"		: { "$date": { "$numberLong": "0" } }
			, "DatetimePositive"	: { "$date": { "$numberLong": "9223372036854775807" } }
			, "DatetimeNegative"	: { "$date": { "$numberLong": "-9223372036854775808" } }
			, "True"				: true
			, "False"				: false
			, "DBPointer"			: { "$dbPointer": { "$ref": "db.collection", "$id": { "$oid": "57e193d7a9cc81b4027498b1" } } }
			, "DBRef"				: { "$ref": "collection", "$id": { "$oid": "57fd71e96e32ab4225b723fb" }, "$db": "database" }
			, "DBRefNoDB"			: { "$ref": "collection", "$id": { "$oid": "57fd71e96e32ab4225b723fb" } }
			, "MinKey"				: { "$minKey": 1 }
			, "MaxKey"				: { "$maxKey": 1 }
			, "Null"				: null
			, "Undefined"			: { "$undefined": true }
			}`

	ejp := newExtJSONParser(strings.NewReader(in), true)

	cases := []ejpTestCase{
		{
			f: expectSingleValue, p: ejp,
			k: "_id", t: bson.TypeObjectID, v: &extJSONValue{t: bson.TypeString, v: "57e193d7a9cc81b4027498b5"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Symbol", t: bson.TypeSymbol, v: &extJSONValue{t: bson.TypeString, v: "symbol"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "String", t: bson.TypeString, v: &extJSONValue{t: bson.TypeString, v: "string"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Int32", t: bson.TypeInt32, v: &extJSONValue{t: bson.TypeString, v: "42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Int64", t: bson.TypeInt64, v: &extJSONValue{t: bson.TypeString, v: "42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Double", t: bson.TypeDouble, v: &extJSONValue{t: bson.TypeString, v: "42.42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "SpecialFloat", t: bson.TypeDouble, v: &extJSONValue{t: bson.TypeString, v: "NaN"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Decimal", t: bson.TypeDecimal128, v: &extJSONValue{t: bson.TypeString, v: "1234"},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "Binary", t: bson.TypeBinary,
			v: &extJSONObject{
				keys: []string{"base64", "subType"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "o0w498Or7cijeBSpkquNtg=="},
					{t: bson.TypeString, v: "03"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "BinaryUserDefined", t: bson.TypeBinary,
			v: &extJSONObject{
				keys: []string{"base64", "subType"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "AQIDBAU="},
					{t: bson.TypeString, v: "80"},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Code", t: bson.TypeJavaScript, v: &extJSONValue{t: bson.TypeString, v: "function() {}"},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "CodeWithEmptyScope", t: bson.TypeCodeWithScope,
			v: ejpSubDocumentTestValue{
				code: "function() {}",
				ktvs: []ejpKeyTypValTriple{},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "CodeWithScope", t: bson.TypeCodeWithScope,
			v: ejpSubDocumentTestValue{
				code: "function() {}",
				ktvs: []ejpKeyTypValTriple{
					{"x", bson.TypeInt32, &extJSONValue{t: bson.TypeInt32, v: int32(1)}},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "Subdocument", t: bson.TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"foo", bson.TypeString, &extJSONValue{t: bson.TypeString, v: "bar"}},
					{"baz", bson.TypeInt32, &extJSONValue{t: bson.TypeString, v: "42"}},
				},
			},
		},
		{
			f: expectArray, p: ejp,
			k: "Array", t: bson.TypeArray,
			v: []ejpKeyTypValTriple{
				{typ: bson.TypeInt32, val: &extJSONValue{t: bson.TypeString, v: "1"}},
				{typ: bson.TypeInt64, val: &extJSONValue{t: bson.TypeString, v: "2"}},
				{typ: bson.TypeDouble, val: &extJSONValue{t: bson.TypeString, v: "3"}},
				{typ: bson.TypeInt32, val: &extJSONValue{t: bson.TypeInt32, v: int32(4)}},
				{typ: bson.TypeDouble, val: &extJSONValue{t: bson.TypeDouble, v: 5.0}},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "Timestamp", t: bson.TypeTimestamp,
			v: &extJSONObject{
				keys: []string{"t", "i"},
				values: []*extJSONValue{
					{t: bson.TypeInt32, v: int32(42)},
					{t: bson.TypeInt32, v: int32(1)},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "RegularExpression", t: bson.TypeRegex,
			v: &extJSONObject{
				keys: []string{"pattern", "options"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "foo*"},
					{t: bson.TypeString, v: "ix"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimeEpoch", t: bson.TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "0"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimePositive", t: bson.TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "9223372036854775807"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimeNegative", t: bson.TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "-9223372036854775808"},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "True", t: bson.TypeBoolean, v: &extJSONValue{t: bson.TypeBoolean, v: true},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "False", t: bson.TypeBoolean, v: &extJSONValue{t: bson.TypeBoolean, v: false},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DBPointer", t: bson.TypeDBPointer,
			v: &extJSONObject{
				keys: []string{"$ref", "$id"},
				values: []*extJSONValue{
					{t: bson.TypeString, v: "db.collection"},
					{t: bson.TypeString, v: "57e193d7a9cc81b4027498b1"},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "DBRef", t: bson.TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"$ref", bson.TypeString, &extJSONValue{t: bson.TypeString, v: "collection"}},
					{"$id", bson.TypeObjectID, &extJSONValue{t: bson.TypeString, v: "57fd71e96e32ab4225b723fb"}},
					{"$db", bson.TypeString, &extJSONValue{t: bson.TypeString, v: "database"}},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "DBRefNoDB", t: bson.TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"$ref", bson.TypeString, &extJSONValue{t: bson.TypeString, v: "collection"}},
					{"$id", bson.TypeObjectID, &extJSONValue{t: bson.TypeString, v: "57fd71e96e32ab4225b723fb"}},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "MinKey", t: bson.TypeMinKey, v: &extJSONValue{t: bson.TypeInt32, v: int32(1)},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "MaxKey", t: bson.TypeMaxKey, v: &extJSONValue{t: bson.TypeInt32, v: int32(1)},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Null", t: bson.TypeNull, v: &extJSONValue{t: bson.TypeNull, v: nil},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Undefined", t: bson.TypeUndefined, v: &extJSONValue{t: bson.TypeBoolean, v: true},
		},
	}

	// run the test cases
	for _, tc := range cases {
		tc.f(t, tc.p, tc.k, tc.t, tc.v)
	}

	// expect end of whole document: read final }
	k, typ, err := ejp.readKey()
	readKeyDiff(t, "", k, bson.Type(0), typ, err, expectErrEOD, "")

	// expect end of whole document: read EOF
	k, typ, err = ejp.readKey()
	readKeyDiff(t, "", k, bson.Type(0), typ, err, expectErrEOD, "")
	if diff := cmp.Diff(jpsDoneState, ejp.s); diff != "" {
		t.Errorf("expected parser to be in done state but instead is in %v\n", ejp.s)
		t.FailNow()
	}
}
