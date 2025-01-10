// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	keyDiff = specificDiff("key")
	typDiff = specificDiff("type")
	valDiff = specificDiff("value")

	expectErrEOF = expectSpecificError(io.EOF)
	expectErrEOD = expectSpecificError(ErrEOD)
	expectErrEOA = expectSpecificError(ErrEOA)
)

type expectedErrorFunc func(t *testing.T, err error, desc string)

type peekTypeTestCase struct {
	desc  string
	input string
	typs  []Type
	errFs []expectedErrorFunc
}

type readKeyValueTestCase struct {
	desc  string
	input string
	keys  []string
	typs  []Type
	vals  []*extJSONValue

	keyEFs []expectedErrorFunc
	valEFs []expectedErrorFunc
}

func expectNoError(t *testing.T, err error, desc string) {
	if err != nil {
		t.Helper()
		t.Errorf("%s: Unepexted error: %v", desc, err)
		t.FailNow()
	}
}

func expectError(t *testing.T, err error, desc string) {
	if err == nil {
		t.Helper()
		t.Errorf("%s: Expected error", desc)
		t.FailNow()
	}
}

func expectSpecificError(expected error) expectedErrorFunc {
	return func(t *testing.T, err error, desc string) {
		if !errors.Is(err, expected) {
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

func readKeyDiff(t *testing.T, eKey, aKey string, eTyp, aTyp Type, err error, errF expectedErrorFunc, desc string) {
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
	makeValidPeekTypeTestCase := func(input string, typ Type, desc string) peekTypeTestCase {
		return peekTypeTestCase{
			desc: desc, input: input,
			typs:  []Type{typ},
			errFs: []expectedErrorFunc{expectNoError},
		}
	}
	makeInvalidTestCase := func(desc, input string, lastEF expectedErrorFunc) peekTypeTestCase {
		return peekTypeTestCase{
			desc: desc, input: input,
			typs:  []Type{Type(0)},
			errFs: []expectedErrorFunc{lastEF},
		}
	}

	makeInvalidPeekTypeTestCase := func(desc, input string, lastEF expectedErrorFunc) peekTypeTestCase {
		return peekTypeTestCase{
			desc: desc, input: input,
			typs:  []Type{TypeArray, TypeString, Type(0)},
			errFs: []expectedErrorFunc{expectNoError, expectNoError, lastEF},
		}
	}

	cases := []peekTypeTestCase{
		makeValidPeekTypeTestCase(`null`, TypeNull, "Null"),
		makeValidPeekTypeTestCase(`"string"`, TypeString, "String"),
		makeValidPeekTypeTestCase(`true`, TypeBoolean, "Boolean--true"),
		makeValidPeekTypeTestCase(`false`, TypeBoolean, "Boolean--false"),
		makeValidPeekTypeTestCase(`{"$minKey": 1}`, TypeMinKey, "MinKey"),
		makeValidPeekTypeTestCase(`{"$maxKey": 1}`, TypeMaxKey, "MaxKey"),
		makeValidPeekTypeTestCase(`{"$numberInt": "42"}`, TypeInt32, "Int32"),
		makeValidPeekTypeTestCase(`{"$numberLong": "42"}`, TypeInt64, "Int64"),
		makeValidPeekTypeTestCase(`{"$symbol": "symbol"}`, TypeSymbol, "Symbol"),
		makeValidPeekTypeTestCase(`{"$numberDouble": "42.42"}`, TypeDouble, "Double"),
		makeValidPeekTypeTestCase(`{"$undefined": true}`, TypeUndefined, "Undefined"),
		makeValidPeekTypeTestCase(`{"$numberDouble": "NaN"}`, TypeDouble, "Double--NaN"),
		makeValidPeekTypeTestCase(`{"$numberDecimal": "1234"}`, TypeDecimal128, "Decimal"),
		makeValidPeekTypeTestCase(`{"foo": "bar"}`, TypeEmbeddedDocument, "Toplevel document"),
		makeValidPeekTypeTestCase(`{"$date": {"$numberLong": "0"}}`, TypeDateTime, "Datetime"),
		makeValidPeekTypeTestCase(`{"$code": "function() {}"}`, TypeJavaScript, "Code no scope"),
		makeValidPeekTypeTestCase(`[{"$numberInt": "1"},{"$numberInt": "2"}]`, TypeArray, "Array"),
		makeValidPeekTypeTestCase(`{"$timestamp": {"t": 42, "i": 1}}`, TypeTimestamp, "Timestamp"),
		makeValidPeekTypeTestCase(`{"$oid": "57e193d7a9cc81b4027498b5"}`, TypeObjectID, "Object ID"),
		makeValidPeekTypeTestCase(`{"$binary": {"base64": "AQIDBAU=", "subType": "80"}}`, TypeBinary, "Binary"),
		makeValidPeekTypeTestCase(`{"$code": "function() {}", "$scope": {}}`, TypeCodeWithScope, "Code With Scope"),
		makeValidPeekTypeTestCase(`{"$binary": {"base64": "o0w498Or7cijeBSpkquNtg==", "subType": "03"}}`, TypeBinary, "Binary"),
		makeValidPeekTypeTestCase(`{"$binary": "o0w498Or7cijeBSpkquNtg==", "$type": "03"}`, TypeBinary, "Binary"),
		makeValidPeekTypeTestCase(`{"$regularExpression": {"pattern": "foo*", "options": "ix"}}`, TypeRegex, "Regular expression"),
		makeValidPeekTypeTestCase(`{"$dbPointer": {"$ref": "db.collection", "$id": {"$oid": "57e193d7a9cc81b4027498b1"}}}`, TypeDBPointer, "DBPointer"),
		makeValidPeekTypeTestCase(`{"$ref": "collection", "$id": {"$oid": "57fd71e96e32ab4225b723fb"}, "$db": "database"}`, TypeEmbeddedDocument, "DBRef"),
		makeInvalidPeekTypeTestCase("invalid array--missing ]", `["a"`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--colon in array", `["a":`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--extra comma", `["a",,`, expectError),
		makeInvalidPeekTypeTestCase("invalid array--trailing comma", `["a",]`, expectError),
		makeInvalidPeekTypeTestCase("peekType after end of array", `["a"]`, expectErrEOA),
		{
			desc:  "invalid array--leading comma",
			input: `[,`,
			typs:  []Type{TypeArray, Type(0)},
			errFs: []expectedErrorFunc{expectNoError, expectError},
		},
		makeInvalidTestCase("lone $scope", `{"$scope": {}}`, expectError),
		makeInvalidTestCase("empty code with unknown extra key", `{"$code":"", "0":""}`, expectError),
		makeInvalidTestCase("non-empty code with unknown extra key", `{"$code":"foobar", "0":""}`, expectError),
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ejp := newExtJSONParser(strings.NewReader(tc.input), true)
			// Manually set the parser's starting state to jpsSawColon so peekType will read ahead to find the extjson
			// type of the value. If not set, the parser will be in jpsStartState and advance to jpsSawKey, which will
			// cause it to return without peeking the extjson type.
			ejp.s = jpsSawColon

			for i, eTyp := range tc.typs {
				errF := tc.errFs[i]

				typ, err := ejp.peekType()
				errF(t, err, tc.desc)
				if err != nil {
					// Don't inspect the type if there was an error
					return
				}

				typDiff(t, eTyp, typ, tc.desc)
			}
		})
	}
}

func TestExtJSONParserReadKeyReadValue(t *testing.T) {
	// several test cases will use the same keys, types, and values, and only differ on input structure

	keys := []string{"_id", "Symbol", "String", "Int32", "Int64", "Int", "MinKey"}
	types := []Type{TypeObjectID, TypeSymbol, TypeString, TypeInt32, TypeInt64, TypeInt32, TypeMinKey}
	values := []*extJSONValue{
		{t: TypeString, v: "57e193d7a9cc81b4027498b5"},
		{t: TypeString, v: "symbol"},
		{t: TypeString, v: "string"},
		{t: TypeString, v: "42"},
		{t: TypeString, v: "42"},
		{t: TypeInt32, v: int32(42)},
		{t: TypeInt32, v: int32(1)},
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
			typs:   []Type{Type(0)},
			vals:   []*extJSONValue{nil},
			keyEFs: []expectedErrorFunc{expectError},
			valEFs: []expectedErrorFunc{expectErrorNOOP},
		}
	}

	secondKeyError := func(desc, input, firstKey string, firstType Type, firstValue *extJSONValue) readKeyValueTestCase {
		return readKeyValueTestCase{
			desc:   desc,
			input:  input,
			keys:   []string{firstKey, ""},
			typs:   []Type{firstType, Type(0)},
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
			typs:  []Type{TypeInt32, TypeEmbeddedDocument, TypeEmbeddedDocument, TypeInt32, Type(0), Type(0), TypeInt32, Type(0)},
			vals: []*extJSONValue{
				{t: TypeInt32, v: int32(1)}, nil, nil, {t: TypeInt32, v: int32(4)}, nil, nil, {t: TypeInt32, v: int32(5)}, nil,
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
			typs:   []Type{TypeInt32},
			vals:   []*extJSONValue{nil},
			keyEFs: []expectedErrorFunc{expectNoError},
			valEFs: []expectedErrorFunc{expectError},
		},
		firstKeyError("invalid input: missing key--EOF", "{"),
		firstKeyError("invalid input: missing key--colon first", "{:"),
		firstKeyError("invalid input: missing value", `{"a":`),
		firstKeyError("invalid input: missing colon", `{"a" 1`),
		firstKeyError("invalid input: extra colon", `{"a"::`),
		secondKeyError("invalid input: missing }", `{"a": 1`, "a", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: missing comma", `{"a": 1 "b"`, "a", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: extra comma", `{"a": 1,, "b"`, "a", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}),
		secondKeyError("invalid input: trailing comma in object", `{"a": 1,}`, "a", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}),
		{
			desc:   "invalid input: lone scope after a complete value",
			input:  `{"a": "", "b": {"$scope: ""}}`,
			keys:   []string{"a"},
			typs:   []Type{TypeString},
			vals:   []*extJSONValue{{TypeString, ""}},
			keyEFs: []expectedErrorFunc{expectNoError, expectNoError},
			valEFs: []expectedErrorFunc{expectNoError, expectError},
		},
		{
			desc:   "invalid input: lone scope nested",
			input:  `{"a":{"b":{"$scope":{`,
			keys:   []string{},
			typs:   []Type{},
			vals:   []*extJSONValue{nil},
			keyEFs: []expectedErrorFunc{expectNoError},
			valEFs: []expectedErrorFunc{expectError},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
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
		})
	}
}

type ejpExpectationTest func(t *testing.T, p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{})

type ejpTestCase struct {
	f ejpExpectationTest
	p *extJSONParser
	k string
	t Type
	v interface{}
}

// expectSingleValue is used for simple JSON types (strings, numbers, literals) and for extended JSON types that
// have single key-value pairs (i.e. { "$minKey": 1 }, { "$numberLong": "42.42" })
func expectSingleValue(t *testing.T, p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}) {
	eVal := expectedValue.(*extJSONValue)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	v, err := p.readValue(typ)
	readValueDiff(t, eVal, v, err, expectNoError, expectedKey)
}

// expectMultipleValues is used for values that are subdocuments of known size and with known keys (such as extended
// JSON types { "$timestamp": {"t": 1, "i": 1} } and { "$regularExpression": {"pattern": "", options: ""} })
func expectMultipleValues(t *testing.T, p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}) {
	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	v, err := p.readValue(typ)
	expectNoError(t, err, "")
	typDiff(t, TypeEmbeddedDocument, v.t, expectedKey)

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
	typ Type
	val *extJSONValue
}

type ejpSubDocumentTestValue struct {
	code string               // code is only used for TypeCodeWithScope (and is ignored for TypeEmbeddedDocument
	ktvs []ejpKeyTypValTriple // list of (key, type, value) triples; this is "scope" for TypeCodeWithScope
}

// expectSubDocument is used for embedded documents and code with scope types; it reads all the keys and values
// in the embedded document (or scope for codeWithScope) and compares them to the expectedValue's list of (key, type,
// value) triples
func expectSubDocument(t *testing.T, p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}) {
	subdoc := expectedValue.(ejpSubDocumentTestValue)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, expectedType, typ, err, expectNoError, expectedKey)

	if expectedType == TypeCodeWithScope {
		v, err := p.readValue(typ)
		readValueDiff(t, &extJSONValue{t: TypeString, v: subdoc.code}, v, err, expectNoError, expectedKey)
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

	if expectedType == TypeCodeWithScope {
		// expect scope doc to close
		k, typ, err = p.readKey()
		readKeyDiff(t, "", k, Type(0), typ, err, expectErrEOD, expectedKey)
	}

	// expect subdoc to close
	k, typ, err = p.readKey()
	readKeyDiff(t, "", k, Type(0), typ, err, expectErrEOD, expectedKey)
}

// expectArray takes the expectedKey, ignores the expectedType, and uses the expectedValue
// as a slice of (type Type, value *extJSONValue) pairs
func expectArray(t *testing.T, p *extJSONParser, expectedKey string, _ Type, expectedValue interface{}) {
	ktvs := expectedValue.([]ejpKeyTypValTriple)

	k, typ, err := p.readKey()
	readKeyDiff(t, expectedKey, k, TypeArray, typ, err, expectNoError, expectedKey)

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
	typDiff(t, Type(0), typ, expectedKey)
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
			, "BinaryLegacy"  : { "$binary": "o0w498Or7cijeBSpkquNtg==", "$type": "03" }
			, "BinaryUserDefined"	: { "$binary": { "base64": "AQIDBAU=", "subType": "80" } }
			, "Code"				: { "$code": "function() {}" }
			, "CodeWithEmptyScope"	: { "$code": "function() {}", "$scope": {} }
			, "CodeWithScope"		: { "$code": "function() {}", "$scope": { "x": 1 } }
			, "EmptySubdocument"    : {}
			, "Subdocument"			: { "foo": "bar", "baz": { "$numberInt": "42" } }
			, "Array"				: [{"$numberInt": "1"}, {"$numberLong": "2"}, {"$numberDouble": "3"}, 4, "string", 5.0]
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
			k: "_id", t: TypeObjectID, v: &extJSONValue{t: TypeString, v: "57e193d7a9cc81b4027498b5"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Symbol", t: TypeSymbol, v: &extJSONValue{t: TypeString, v: "symbol"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "String", t: TypeString, v: &extJSONValue{t: TypeString, v: "string"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Int32", t: TypeInt32, v: &extJSONValue{t: TypeString, v: "42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Int64", t: TypeInt64, v: &extJSONValue{t: TypeString, v: "42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Double", t: TypeDouble, v: &extJSONValue{t: TypeString, v: "42.42"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "SpecialFloat", t: TypeDouble, v: &extJSONValue{t: TypeString, v: "NaN"},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Decimal", t: TypeDecimal128, v: &extJSONValue{t: TypeString, v: "1234"},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "Binary", t: TypeBinary,
			v: &extJSONObject{
				keys: []string{"base64", "subType"},
				values: []*extJSONValue{
					{t: TypeString, v: "o0w498Or7cijeBSpkquNtg=="},
					{t: TypeString, v: "03"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "BinaryLegacy", t: TypeBinary,
			v: &extJSONObject{
				keys: []string{"base64", "subType"},
				values: []*extJSONValue{
					{t: TypeString, v: "o0w498Or7cijeBSpkquNtg=="},
					{t: TypeString, v: "03"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "BinaryUserDefined", t: TypeBinary,
			v: &extJSONObject{
				keys: []string{"base64", "subType"},
				values: []*extJSONValue{
					{t: TypeString, v: "AQIDBAU="},
					{t: TypeString, v: "80"},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Code", t: TypeJavaScript, v: &extJSONValue{t: TypeString, v: "function() {}"},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "CodeWithEmptyScope", t: TypeCodeWithScope,
			v: ejpSubDocumentTestValue{
				code: "function() {}",
				ktvs: []ejpKeyTypValTriple{},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "CodeWithScope", t: TypeCodeWithScope,
			v: ejpSubDocumentTestValue{
				code: "function() {}",
				ktvs: []ejpKeyTypValTriple{
					{"x", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "EmptySubdocument", t: TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "Subdocument", t: TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"foo", TypeString, &extJSONValue{t: TypeString, v: "bar"}},
					{"baz", TypeInt32, &extJSONValue{t: TypeString, v: "42"}},
				},
			},
		},
		{
			f: expectArray, p: ejp,
			k: "Array", t: TypeArray,
			v: []ejpKeyTypValTriple{
				{typ: TypeInt32, val: &extJSONValue{t: TypeString, v: "1"}},
				{typ: TypeInt64, val: &extJSONValue{t: TypeString, v: "2"}},
				{typ: TypeDouble, val: &extJSONValue{t: TypeString, v: "3"}},
				{typ: TypeInt32, val: &extJSONValue{t: TypeInt32, v: int32(4)}},
				{typ: TypeString, val: &extJSONValue{t: TypeString, v: "string"}},
				{typ: TypeDouble, val: &extJSONValue{t: TypeDouble, v: 5.0}},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "Timestamp", t: TypeTimestamp,
			v: &extJSONObject{
				keys: []string{"t", "i"},
				values: []*extJSONValue{
					{t: TypeInt32, v: int32(42)},
					{t: TypeInt32, v: int32(1)},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "RegularExpression", t: TypeRegex,
			v: &extJSONObject{
				keys: []string{"pattern", "options"},
				values: []*extJSONValue{
					{t: TypeString, v: "foo*"},
					{t: TypeString, v: "ix"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimeEpoch", t: TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: TypeString, v: "0"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimePositive", t: TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: TypeString, v: "9223372036854775807"},
				},
			},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DatetimeNegative", t: TypeDateTime,
			v: &extJSONObject{
				keys: []string{"$numberLong"},
				values: []*extJSONValue{
					{t: TypeString, v: "-9223372036854775808"},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "True", t: TypeBoolean, v: &extJSONValue{t: TypeBoolean, v: true},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "False", t: TypeBoolean, v: &extJSONValue{t: TypeBoolean, v: false},
		},
		{
			f: expectMultipleValues, p: ejp,
			k: "DBPointer", t: TypeDBPointer,
			v: &extJSONObject{
				keys: []string{"$ref", "$id"},
				values: []*extJSONValue{
					{t: TypeString, v: "db.collection"},
					{t: TypeString, v: "57e193d7a9cc81b4027498b1"},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "DBRef", t: TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"$ref", TypeString, &extJSONValue{t: TypeString, v: "collection"}},
					{"$id", TypeObjectID, &extJSONValue{t: TypeString, v: "57fd71e96e32ab4225b723fb"}},
					{"$db", TypeString, &extJSONValue{t: TypeString, v: "database"}},
				},
			},
		},
		{
			f: expectSubDocument, p: ejp,
			k: "DBRefNoDB", t: TypeEmbeddedDocument,
			v: ejpSubDocumentTestValue{
				ktvs: []ejpKeyTypValTriple{
					{"$ref", TypeString, &extJSONValue{t: TypeString, v: "collection"}},
					{"$id", TypeObjectID, &extJSONValue{t: TypeString, v: "57fd71e96e32ab4225b723fb"}},
				},
			},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "MinKey", t: TypeMinKey, v: &extJSONValue{t: TypeInt32, v: int32(1)},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "MaxKey", t: TypeMaxKey, v: &extJSONValue{t: TypeInt32, v: int32(1)},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Null", t: TypeNull, v: &extJSONValue{t: TypeNull, v: nil},
		},
		{
			f: expectSingleValue, p: ejp,
			k: "Undefined", t: TypeUndefined, v: &extJSONValue{t: TypeBoolean, v: true},
		},
	}

	// run the test cases
	for _, tc := range cases {
		tc.f(t, tc.p, tc.k, tc.t, tc.v)
	}

	// expect end of whole document: read final }
	k, typ, err := ejp.readKey()
	readKeyDiff(t, "", k, Type(0), typ, err, expectErrEOD, "")

	// expect end of whole document: read EOF
	k, typ, err = ejp.readKey()
	readKeyDiff(t, "", k, Type(0), typ, err, expectErrEOF, "")
	if diff := cmp.Diff(jpsDoneState, ejp.s); diff != "" {
		t.Errorf("expected parser to be in done state but instead is in %v\n", ejp.s)
		t.FailNow()
	}
}

func TestExtJSONValue(t *testing.T) {
	t.Run("Large Date", func(t *testing.T) {
		val := &extJSONValue{
			t: TypeString,
			v: "3001-01-01T00:00:00Z",
		}

		intVal, err := val.parseDateTime()
		if err != nil {
			t.Fatalf("error parsing date time: %v", err)
		}

		if intVal <= 0 {
			t.Fatalf("expected value above 0, got %v", intVal)
		}
	})
	t.Run("fallback time format", func(t *testing.T) {
		val := &extJSONValue{
			t: TypeString,
			v: "2019-06-04T14:54:31.416+0000",
		}

		_, err := val.parseDateTime()
		if err != nil {
			t.Fatalf("error parsing date time: %v", err)
		}
	})
}
