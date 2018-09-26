// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtJSONParserPeekType(t *testing.T) {
	cases := [][]interface{}{
		{`{"$oid": "57e193d7a9cc81b4027498b5"}`, TypeObjectID, "Object ID"},
		{`{"$symbol": "symbol"}`, TypeSymbol, "Symbol"},
		{`"string"`, TypeString, "String"},
		{`{"$numberInt": "42"}`, TypeInt32, "Int32"},
		{`{"$numberLong": "42"}`, TypeInt64, "Int64"},
		{`{"$numberDouble": "42.42"}`, TypeDouble, "Double"},
		{`{"$numberDouble": "NaN"}`, TypeDouble, "Double--NaN"},
		{`{"$numberDecimal": "1234"}`, TypeDecimal128, "Decimal"},
		{`{"$binary": {"base64": "o0w498Or7cijeBSpkquNtg==", "subType": "03"}}`, TypeBinary, "Binary"},
		{`{"$binary": {"base64": "AQIDBAU=", "subType": "80"}}`, TypeBinary, "Binary"},
		{`{"$code": "function() {}"}`, TypeJavaScript, "Code no scope"},
		{`{"$code": "function() {}", "$scope": {}}`, TypeCodeWithScope, "Code With Scope"},
		{`{"foo": "bar"}`, TypeEmbeddedDocument, "Toplevel document"},
		{`[{"$numberInt": "1"},{"$numberInt": "2"},{"$numberInt": "3"},{"$numberInt": "4"},{"$numberInt": "5"}]`, TypeArray, "Array"},
		{`{"$timestamp": {"t": 42, "i": 1}}`, TypeTimestamp, "Timestamp"},
		{`{"$regularExpression": {"pattern": "foo*", "options": "ix"}}`, TypeRegex, "Regular expression"},
		{`{"$date": {"$numberLong": "0"}}`, TypeDateTime, "Datetime"},
		{`true`, TypeBoolean, "Boolean--true"},
		{`false`, TypeBoolean, "Boolean--false"},
		{`{"$dbPointer": {"$ref": "db.collection", "$id": {"$oid": "57e193d7a9cc81b4027498b1"}}}`, TypeDBPointer, "DBPointer"},
		{`{"$ref": "collection", "$id": {"$oid": "57fd71e96e32ab4225b723fb"}, "$db": "database"}`, TypeEmbeddedDocument, "DBRef"},
		{`{"$minKey": 1}`, TypeMinKey, "MinKey"},
		{`{"$maxKey": 1}`, TypeMaxKey, "MaxKey"},
		{`null`, TypeNull, "Null"},
		{`{"$undefined": true}`, TypeUndefined, "Undefined"},
	}

	for _, in := range cases {
		ejp := newExtJSONParser(strings.NewReader(in[0].(string)))

		typ, err := ejp.peekType()
		assert.Equal(t, in[1].(Type), typ, in[2].(string))
		noerr(t, err)
	}
}

func TestExtJSONParserNestedObjects(t *testing.T) {
	in := `{"k1": 1, "k2": { "k3": { "k4": 4 } }, "k5": 5}`

	ejp := newExtJSONParser(strings.NewReader(in))

	k, typ, err := ejp.readKey()
	assert.Equal(t, "k1", k)
	assert.Equal(t, TypeInt32, typ)
	assert.Equal(t, int32(1), ejp.v.v.(int32))
	noerr(t, err)

	k, typ, err = ejp.readKey()
	assert.Equal(t, "k2", k)
	assert.Equal(t, TypeEmbeddedDocument, typ)
	noerr(t, err)

	k, typ, err = ejp.readKey()
	assert.Equal(t, "k3", k)
	assert.Equal(t, TypeEmbeddedDocument, typ)
	noerr(t, err)

	k, typ, err = ejp.readKey()
	assert.Equal(t, "k4", k)
	assert.Equal(t, TypeInt32, typ)
	assert.Equal(t, int32(4), ejp.v.v.(int32))
	noerr(t, err)

	for i := 0; i < 2; i++ {
		k, typ, err = ejp.readKey()
		assert.Equal(t, "", k)
		assert.Zero(t, typ)
		assert.Equal(t, ErrEOD, err)
	}

	k, typ, err = ejp.readKey()
	assert.Equal(t, "k5", k)
	assert.Equal(t, TypeInt32, typ)
	assert.Equal(t, int32(5), ejp.v.v.(int32))
	noerr(t, err)

	k, typ, err = ejp.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)
}

func TestExtJSONParserSpacing(t *testing.T) {
	cases := []string{
		`{
			"_id": { "$oid": "57e193d7a9cc81b4027498b5" },
			"Symbol": { "$symbol": "symbol" },
			"String": "string",
			"Int32": { "$numberInt": "42" },
			"Int64": { "$numberLong": "42" },
			"Int": 42,
			"MinKey": { "$minKey": 1 }
		}`,
		`{ "_id": { "$oid": "57e193d7a9cc81b4027498b5" }
		 , "Symbol": { "$symbol": "symbol" }
		 , "String": "string"
		 , "Int32": { "$numberInt": "42" }
		 , "Int64": { "$numberLong": "42" }
		 , "Int": 42
		 , "MinKey": { "$minKey": 1 }
		 }`,
		`{
			"_id":    { "$oid"       : "57e193d7a9cc81b4027498b5" },
			"Symbol": { "$symbol"    : "symbol" },
			"String": "string",
			"Int32":  { "$numberInt" : "42" },
			"Int64":  { "$numberLong": "42" },
			"Int":    42,
			"MinKey": { "$minKey": 1 }
		}`,
		`{"_id":{"$oid":"57e193d7a9cc81b4027498b5"},"Symbol":{"$symbol":"symbol"},"String":"string","Int32":{"$numberInt":"42"},"Int64":{"$numberLong":"42"},"Int":42,"MinKey":{"$minKey":1}}`,
		`	{
			"_id"		: { "$oid": "57e193d7a9cc81b4027498b5" },
			"Symbol"	: { "$symbol": "symbol" },
			"String"	: "string",
			"Int32"		: { "$numberInt": "42" },
			"Int64"		: { "$numberLong": "42" },
			"Int"		: 42,
			"MinKey"	: { "$minKey": 1 }
			}	`,
	}

	for _, in := range cases {
		ejp := newExtJSONParser(strings.NewReader(in))

		typ, err := ejp.peekType()

		assert.Equal(t, TypeEmbeddedDocument, typ, "Top level document is recognized as \"Embedded Doc\"")
		noerr(t, err)

		k, typ, err := ejp.readKey()
		assert.Equal(t, "_id", k)
		assert.Equal(t, TypeObjectID, typ)
		noerr(t, err)

		v, err := ejp.readValue(TypeObjectID)
		assert.Equal(t, TypeString, v.t)
		assert.Equal(t, "57e193d7a9cc81b4027498b5", v.v.(string))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "Symbol", k)
		assert.Equal(t, TypeSymbol, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeSymbol)
		assert.Equal(t, TypeString, v.t)
		assert.Equal(t, "symbol", v.v.(string))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "String", k)
		assert.Equal(t, TypeString, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeString)
		assert.Equal(t, TypeString, v.t)
		assert.Equal(t, "string", v.v.(string))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "Int32", k)
		assert.Equal(t, TypeInt32, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeInt32)
		assert.Equal(t, TypeString, v.t)
		assert.Equal(t, "42", v.v.(string))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "Int64", k)
		assert.Equal(t, TypeInt64, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeInt64)
		assert.Equal(t, TypeString, v.t)
		assert.Equal(t, "42", v.v.(string))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "Int", k)
		assert.Equal(t, TypeInt32, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeInt32)
		assert.Equal(t, TypeInt32, v.t)
		assert.Equal(t, int32(42), v.v.(int32))
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "MinKey", k)
		assert.Equal(t, TypeMinKey, typ)
		noerr(t, err)

		v, err = ejp.readValue(TypeMinKey)
		assert.Equal(t, TypeInt32, v.t)
		assert.Equal(t, int32(1), v.v.(int32))
		noerr(t, err)
	}
}

func TestExtJSONParserInvalidObjects(t *testing.T) {
	cases := [][]interface{}{
		{`{`, 1, "missing key--EOF"},
		{`{:`, 1, "missing key--colon first"},
		{`{"a":`, 1, "missing value"},
		{`{"a" 1`, 1, "missing colon"},
		{`{"a"::`, 1, "extra colon"},
		{`{"a": {"$numberInt": "1", "x"`, 2, "invalid values for extended type"},
		{`{"a": 1`, 3, "missing }"},
		{`{"a": 1 "b"`, 3, "missing comma"},
		{`{"a": 1,, "b"`, 3, "extra comma--after value"},
		{`{"a": 1,}`, 3, "extra comma--at end"},
	}

	for _, tc := range cases {
		in := tc[0].(string)
		n := tc[1].(int)
		msg := tc[2].(string)

		ejp := newExtJSONParser(strings.NewReader(in))

		k, typ, err := ejp.readKey()
		if n == 1 {
			assert.Equal(t, "", k, msg)
			assert.Zero(t, typ, msg)
			assert.Error(t, err, msg)
			continue
		}

		assert.Equal(t, "a", k, msg)
		assert.NotZero(t, typ, msg)
		noerr(t, err)

		v, err := ejp.readValue(TypeInt32)
		if n == 2 {
			assert.Nil(t, v, msg)
			assert.Error(t, err, msg)
			continue
		}

		assert.Equal(t, int32(1), v.v.(int32), msg)
		noerr(t, err)

		k, typ, err = ejp.readKey()
		assert.Equal(t, "", k, msg)
		assert.Zero(t, typ, msg)
		assert.Error(t, err, msg)
	}
}

func TestExtJSONParserInvalidArrays(t *testing.T) {
	cases := [][]interface{}{
		{`[,`, 1, "leading comma"},
		{`[1`, 2, "missing ]"},
		{`["a":`, 2, "colon in array"},
		{`[1,,`, 2, "extra comma"},
		{`[1,]`, 2, "trailing comma"},
	}

	for _, tc := range cases {
		in := tc[0].(string)
		n := tc[1].(int)
		msg := tc[2].(string)

		ejp := newExtJSONParser(strings.NewReader(in))

		for i := 0; i < n; i++ {
			_, err := ejp.peekType()
			noerr(t, err)
		}

		_, err := ejp.peekType()
		assert.Error(t, err, msg)
	}
}

type ejpExpectationTest func(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T)

type ejpTestCase struct {
	f ejpExpectationTest
	p *extJSONParser
	k string
	t Type
	v interface{}
}

func expectStringValue(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, expectedType, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeString, v.t)
	assert.Equal(t, expectedValue.(string), v.v.(string))
	noerr(t, err)
}

func expectObjectValue(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, expectedType, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeEmbeddedDocument, v.t)
	noerr(t, err)

	actObj := v.v.(*extJSONObject)
	expObj := expectedValue.(*extJSONObject)

	for i, actKey := range actObj.keys {
		expKey := expObj.keys[i]
		actVal := actObj.values[i]
		expVal := expObj.values[i]

		assert.Equal(t, expKey, actKey)
		assert.Equal(t, expVal.t, actVal.t)
		assert.Equal(t, expVal.v, actVal.v)
	}

	// expect object to end
	k, typ, err = p.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)
}

type ejpTestCodeWithScope struct {
	code  string
	scope [][]interface{} // list of (key string, type Type, val *extJSONValue) triples
}

// expectCodeWithScope takes the expectedKey, ignores the expectedType, and uses the expectedValue
// as an ejpTestCodeWithScope
func expectCodeWithScope(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	cws := expectedValue.(ejpTestCodeWithScope)

	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, TypeCodeWithScope, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeString, v.t)
	assert.Equal(t, cws.code, v.v.(string))
	noerr(t, err)

	for _, kv := range cws.scope {
		eKey := kv[0].(string)
		eTyp := kv[1].(Type)
		eVal := kv[2].(*extJSONValue)

		k, typ, err = p.readKey()
		assert.Equal(t, eKey, k)
		assert.Equal(t, eTyp, typ)
		noerr(t, err)

		v, err = p.readValue(typ)
		assert.Equal(t, eVal.t, v.t)
		assert.Equal(t, eVal.v, v.v)
		noerr(t, err)
	}

	// expect scope doc to close
	k, typ, err = p.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)

	// expect entire codeWithScope doc to close
	k, typ, err = p.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)
}

// expectSubDocument takes the expected key, ignores the expectedType, and uses the expectedValue
// as a slice of (key string, type Type, value *extJSONValue) triples
func expectSubDocument(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	ktvs := expectedValue.([][]interface{})

	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, TypeEmbeddedDocument, typ)
	noerr(t, err)

	for _, kv := range ktvs {
		eKey := kv[0].(string)
		eTyp := kv[1].(Type)
		eVal := kv[2].(*extJSONValue)

		k, typ, err = p.readKey()
		assert.Equal(t, eKey, k)
		assert.Equal(t, eTyp, typ)
		noerr(t, err)

		v, err := p.readValue(typ)
		assert.Equal(t, eVal.t, v.t)
		assert.Equal(t, eVal.v, v.v)
		noerr(t, err)
	}

	// expect subdoc to close
	k, typ, err = p.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)
}

// expectArray takes the expectedKey, ignores the expectedType, and uses the expectedValue
// as a slice of (type Type, value *extJSONValue) pairs
func expectArray(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	tvs := expectedValue.([][]interface{})

	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, TypeArray, typ)
	noerr(t, err)

	for _, tv := range tvs {
		eTyp := tv[0].(Type)
		eVal := tv[1].(*extJSONValue)

		typ, err = p.peekType()
		assert.Equal(t, eTyp, typ)
		noerr(t, err)

		v, err := p.readValue(typ)
		assert.Equal(t, eVal.t, v.t)
		assert.Equal(t, eVal.v, v.v)
		noerr(t, err)
	}

	// expect array to end
	typ, err = p.peekType()
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOA, err)
}

func expectBoolValue(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, expectedType, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeBoolean, v.t)
	assert.Equal(t, expectedValue.(bool), v.v.(bool))
	noerr(t, err)
}

func expectInt32Value(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, expectedType, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeInt32, v.t)
	assert.Equal(t, expectedValue.(int32), v.v.(int32))
	noerr(t, err)
}

func expectNullValue(p *extJSONParser, expectedKey string, expectedType Type, expectedValue interface{}, t *testing.T) {
	k, typ, err := p.readKey()
	assert.Equal(t, expectedKey, k)
	assert.Equal(t, expectedType, typ)
	noerr(t, err)

	v, err := p.readValue(typ)
	assert.Equal(t, TypeNull, v.t)
	assert.Nil(t, v.v)
	noerr(t, err)
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

	ejp := newExtJSONParser(strings.NewReader(in))

	cases := []ejpTestCase{
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "_id", t: TypeObjectID, v: "57e193d7a9cc81b4027498b5",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Symbol", t: TypeSymbol, v: "symbol",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "String", t: TypeString, v: "string",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Int32", t: TypeInt32, v: "42",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Int64", t: TypeInt64, v: "42",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Double", t: TypeDouble, v: "42.42",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "SpecialFloat", t: TypeDouble, v: "NaN",
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Decimal", t: TypeDecimal128, v: "1234",
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "Binary", t: TypeBinary,
			v: &extJSONObject{
				keys:   []string{"base64", "subType"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "o0w498Or7cijeBSpkquNtg=="},
					&extJSONValue{t: TypeString, v: "03"},
				},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "BinaryUserDefined", t: TypeBinary,
			v: &extJSONObject{
				keys:   []string{"base64", "subType"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "AQIDBAU="},
					&extJSONValue{t: TypeString, v: "80"},
				},
			},
		},
		ejpTestCase{
			f: expectStringValue, p: ejp,
			k: "Code", t: TypeJavaScript, v: "function() {}",
		},
		ejpTestCase{
			f: expectCodeWithScope, p: ejp,
			k: "CodeWithEmptyScope", t: TypeCodeWithScope,
			v: ejpTestCodeWithScope{
				code:  "function() {}",
				scope: [][]interface{}{},
			},
		},
		ejpTestCase{
			f: expectCodeWithScope, p: ejp,
			k: "CodeWithScope", t: TypeCodeWithScope,
			v: ejpTestCodeWithScope{
				code:  "function() {}",
				scope: [][]interface{}{
					{"x", TypeInt32, &extJSONValue{t: TypeInt32, v: int32(1)}},
				},
			},
		},
		ejpTestCase{
			f: expectSubDocument, p: ejp,
			k: "Subdocument", t: TypeEmbeddedDocument,
			v: [][]interface{}{
				{"foo", TypeString, &extJSONValue{t: TypeString, v: "bar"}},
				{"baz", TypeInt32, &extJSONValue{t: TypeString, v: "42"}},
			},
		},
		ejpTestCase{
			f: expectArray, p: ejp,
			k: "Array", t: TypeArray,
			v: [][]interface{}{
				{TypeInt32, &extJSONValue{t: TypeString, v: "1"}},
				{TypeInt64, &extJSONValue{t: TypeString, v: "2"}},
				{TypeDouble, &extJSONValue{t: TypeString, v: "3"}},
				{TypeInt32, &extJSONValue{t: TypeInt32, v: int32(4)}},
				{TypeDouble, &extJSONValue{t: TypeDouble, v: 5.0}},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "Timestamp", t: TypeTimestamp,
			v: &extJSONObject{
				keys:   []string{"t", "i"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeInt32, v: int32(42)},
					&extJSONValue{t: TypeInt32, v: int32(1)},
				},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "RegularExpression", t: TypeRegex,
			v: &extJSONObject{
				keys:   []string{"pattern", "options"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "foo*"},
					&extJSONValue{t: TypeString, v: "ix"},
				},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "DatetimeEpoch", t: TypeDateTime,
			v: &extJSONObject{
				keys:   []string{"$numberLong"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "0"},
				},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "DatetimePositive", t: TypeDateTime,
			v: &extJSONObject{
				keys:   []string{"$numberLong"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "9223372036854775807"},
				},
			},
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "DatetimeNegative", t: TypeDateTime,
			v: &extJSONObject{
				keys:   []string{"$numberLong"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "-9223372036854775808"},
				},
			},
		},
		ejpTestCase{
			f: expectBoolValue, p: ejp,
			k: "True", t: TypeBoolean, v: true,
		},
		ejpTestCase{
			f: expectBoolValue, p: ejp,
			k: "False", t: TypeBoolean, v: false,
		},
		ejpTestCase{
			f: expectObjectValue, p: ejp,
			k: "DBPointer", t: TypeDBPointer,
			v: &extJSONObject{
				keys:   []string{"$ref", "$id"},
				values: []*extJSONValue{
					&extJSONValue{t: TypeString, v: "db.collection"},
					&extJSONValue{t: TypeString, v: "57e193d7a9cc81b4027498b1"},
				},
			},
		},
		ejpTestCase{
			f: expectSubDocument, p: ejp,
			k: "DBRef", t: TypeEmbeddedDocument,
			v: [][]interface{}{
				{"$ref", TypeString, &extJSONValue{t: TypeString, v: "collection"}},
				{"$id", TypeObjectID, &extJSONValue{t: TypeString, v: "57fd71e96e32ab4225b723fb"}},
				{"$db", TypeString, &extJSONValue{t: TypeString, v: "database"}},
			},
		},
		ejpTestCase{
			f: expectSubDocument, p: ejp,
			k: "DBRefNoDB", t: TypeEmbeddedDocument,
			v: [][]interface{}{
				{"$ref", TypeString, &extJSONValue{t: TypeString, v: "collection"}},
				{"$id", TypeObjectID, &extJSONValue{t: TypeString, v: "57fd71e96e32ab4225b723fb"}},
			},
		},
		ejpTestCase{
			f: expectInt32Value, p: ejp,
			k: "MinKey", t: TypeMinKey, v: int32(1),
		},
		ejpTestCase{
			f: expectInt32Value, p: ejp,
			k: "MaxKey", t: TypeMaxKey, v: int32(1),
		},
		ejpTestCase{
			f: expectNullValue, p: ejp,
			k: "Null", t: TypeNull, v: nil,
		},
		ejpTestCase{
			f: expectBoolValue, p: ejp,
			k: "Undefined", t: TypeUndefined, v: true,
		},
	}

	// run the test cases
	for _, tc := range cases {
		tc.f(tc.p, tc.k, tc.t, tc.v, t)
	}

	// expect end of whole document: read final }
	k, typ, err := ejp.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)

	// expect end of whole document: read EOF
	k, typ, err = ejp.readKey()
	assert.Equal(t, "", k)
	assert.Zero(t, typ)
	assert.Equal(t, ErrEOD, err)
	assert.Equal(t, jpDoneState, ejp.s)
}
