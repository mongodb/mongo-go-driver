// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/mongodb/mongo-go-driver/bson/builder"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type wrapperType byte

const (
	none     wrapperType = 0
	objectID             = iota
	symbol
	int32Type
	int64Type
	double
	decimalType
	binaryData
	code
	timestamp
	regex
	dbPointer
	dateTime
	dbRef
	minKey
	maxKey
	undefined
)

func (w wrapperType) String() string {
	switch w {
	case objectID:
		return "objectID"
	case int32Type:
		return "int32Type"
	case int64Type:
		return "int64"
	case double:
		return "double"
	case decimalType:
		return "decimalType"
	case binaryData:
		return "binary"
	case code:
		return "JavaScript code"
	case timestamp:
		return "timestamp"
	case regex:
		return "regex"
	case dbPointer:
		return "dbpointer"
	case dateTime:
		return "datetime"
	case dbRef:
		return "dbref"
	case minKey:
		return "minkey"
	case maxKey:
		return "maxkey"
	case undefined:
		return "undefined"
	}

	return "not a wrapper type key"
}

func wrapperKeyType(key []byte) wrapperType {
	switch string(key) {
	case "$numberInt":
		return int32Type
	case "$numberLong":
		return int64Type
	case "$oid":
		return objectID
	case "$symbol":
		return symbol
	case "$numberDouble":
		return double
	case "$numberDecimal":
		return decimalType
	case "$binary":
		return binaryData
	case "$code":
		fallthrough
	case "$scope":
		return code
	case "$timestamp":
		return timestamp
	case "$regularExpression":
		return regex
	case "$dbPointer":
		return dbPointer
	case "$date":
		return dateTime
	case "$ref":
		fallthrough
	case "$id":
		fallthrough
	case "$db":
		return dbRef
	case "$minKey":
		return minKey
	case "$maxKey":
		return maxKey
	case "$undefined":
		return undefined
	}

	return none
}

func wrapperKeyBSONType(key string) Type {
	switch string(key) {
	case "$numberInt":
		return TypeInt32
	case "$numberLong":
		return TypeInt64
	case "$oid":
		return TypeObjectID
	case "$symbol":
		return TypeSymbol
	case "$numberDouble":
		return TypeDouble
	case "$numberDecimal":
		return TypeDecimal128
	case "$binary":
		return TypeBinary
	case "$code":
		return TypeJavaScript
	case "$scope":
		return TypeCodeWithScope
	case "$timestamp":
		return TypeTimestamp
	case "$regularExpression":
		return TypeRegex
	case "$dbPointer":
		return TypeDBPointer
	case "$date":
		return TypeDateTime
	case "$ref":
		fallthrough
	case "$id":
		fallthrough
	case "$db":
		return TypeEmbeddedDocument // TODO: dbrefs aren't bson types
	case "$minKey":
		return TypeMinKey
	case "$maxKey":
		return TypeMaxKey
	case "$undefined":
		return TypeUndefined
	}

	return TypeEmbeddedDocument
}

func parseBinary(ejv *extJSONValue) (b []byte, subType byte, err error) {
	if ejv.t != TypeEmbeddedDocument {
		return nil, 0, fmt.Errorf("$binary value should be object, but instead is %s", ejv.t)
	}

	binObj := ejv.v.(*extJSONObject)
	bFound := false
	stFound := false

	for i, key := range binObj.keys {
		val := binObj.values[i]

		switch key {
		case "base64":
			if bFound {
				return nil, 0, errors.New("duplicate base64 key in $binary")
			}

			if val.t != TypeString {
				return nil, 0, fmt.Errorf("$binary base64 value should be string, but instead is %s", val.t)
			}

			base64Bytes, err := base64.StdEncoding.DecodeString(val.v.(string))
			if err != nil {
				return nil, 0, fmt.Errorf("invalid $binary base64 string: %s", val.v.(string))
			}

			b = base64Bytes
			bFound = true
		case "subType":
			if stFound {
				return nil, 0, errors.New("duplicate subType key in $binary")
			}

			if val.t != TypeString {
				return nil, 0, fmt.Errorf("$binary subType value should be string, but instead is %s", val.t)
			}

			i, err := strconv.ParseInt(val.v.(string), 16, 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid $binary subType string: %s", val.v.(string))
			}

			subType = byte(i)
			stFound = true
		default:
			return nil, 0, fmt.Errorf("invalid key in $binary object: %s", key)
		}
	}

	if !bFound {
		return nil, 0, errors.New("missing base64 field in $binary object")
	}

	if !stFound {
		return nil, 0, errors.New("missing subType field in $binary object")

	}

	return b, subType, nil
}

func parseDBPointer(ejv *extJSONValue) (ns string, oid objectid.ObjectID, err error) {
	if ejv.t != TypeEmbeddedDocument {
		return "", objectid.NilObjectID, fmt.Errorf("$dbPointer value should be object, but instead is %s", ejv.t)
	}

	dbpObj := ejv.v.(*extJSONObject)
	oidFound := false
	nsFound := false

	for i, key := range dbpObj.keys {
		val := dbpObj.values[i]

		switch key {
		case "$ref":
			if nsFound {
				return "", objectid.NilObjectID, errors.New("duplicate $ref key in $dbPointer")
			}

			if val.t != TypeString {
				return "", objectid.NilObjectID, fmt.Errorf("$dbPointer $ref value should be string, but instead is %s", val.t)
			}

			ns = val.v.(string)
			nsFound = true
		case "$id":
			if oidFound {
				return "", objectid.NilObjectID, errors.New("duplicate $id key in $dbPointer")
			}

			if val.t != TypeString {
				return "", objectid.NilObjectID, fmt.Errorf("$dbPointer $id value should be string, but instead is %s", val.t)
			}

			oid, err = objectid.FromHex(val.v.(string))
			if err != nil {
				return "", objectid.NilObjectID, err
			}

			oidFound = true
		default:
			return "", objectid.NilObjectID, fmt.Errorf("invalid key in $dbPointer object: %s", key)
		}
	}

	if  !nsFound {
		return "", oid, errors.New("missing $ref field in $dbPointer object")
	}

	if !oidFound {
		return "", oid, errors.New("missing $id field in $dbPointer object")
	}

	return ns, oid, nil
}

const rfc3339Milli = "2006-01-02T15:04:05.999Z07:00"

func parseDateTime(ejv *extJSONValue) (int64, error) {
	switch ejv.t {
	case TypeString:
		return parseDatetimeString(ejv.v.(string))
	case TypeEmbeddedDocument:
		return parseDatetimeObject(ejv.v.(*extJSONObject))
	default:
		return 0, fmt.Errorf("$date value should be string or object, but instead is %s", ejv.t)
	}
}

func parseDatetimeString(data string) (int64, error) {
	t, err := time.Parse(rfc3339Milli, data)
	if err != nil {
		return 0, fmt.Errorf("invalid $date value string: %s", data)
	}

	return t.Unix() * 1000, nil
}

func parseDatetimeObject(data *extJSONObject) (d int64, err error) {
	dFound := false

	for i, key := range data.keys {
		val := data.values[i]

		switch key {
		case "$numberLong":
			if dFound {
				return 0, errors.New("duplicate $numberLong key in $date")
			}

			if val.t != TypeString {
				return 0, fmt.Errorf("$date $numberLong field should be string, but instead is %s", val.t)
			}

			d, err = parseInt64(val)
			if err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("invalid key in $date object: %s", key)
		}
	}

	if !dFound {
		return 0, errors.New("missing $numberLong field in $date object")
	}

	return d, nil
}

func parseDecimal128(ejv *extJSONValue) (decimal.Decimal128, error) {
	if ejv.t != TypeString {
		return decimal.Decimal128{}, fmt.Errorf("$numberDecimal value should be string, but instead is %s", ejv.t)
	}

	d, err := decimal.ParseDecimal128(ejv.v.(string))
	if err != nil {
		return decimal.Decimal128{}, fmt.Errorf("$invalid $numberDecimal string: %s", ejv.v.(string))
	}

	return d, nil
}

func parseDouble(ejv *extJSONValue) (float64, error) {
	if ejv.t != TypeString {
		return 0, fmt.Errorf("$numberDouble value should be string, but instead is %s", ejv.t)
	}

	switch string(ejv.v.(string)) {
	case "Infinity":
		return math.Inf(1), nil
	case "-Infinity":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}

	f, err := strconv.ParseFloat(ejv.v.(string), 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

func parseInt32(ejv *extJSONValue) (int32, error) {
	if ejv.t != TypeString {
		return 0, fmt.Errorf("$numberInt value should be string, but instead is %s", ejv.t)
	}

	i, err := strconv.ParseInt(ejv.v.(string), 10, 64)
	if err != nil {
		return 0, err
	}

	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, fmt.Errorf("$numberInt value should be int32 but instead is int64: %d", i)
	}

	return int32(i), nil
}

func parseInt64(ejv *extJSONValue) (int64, error) {
	if ejv.t != TypeString {
		return 0, fmt.Errorf("$numberLong value should be string, but instead is %s", ejv.t)
	}

	i, err := strconv.ParseInt(ejv.v.(string), 10, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func parseJavascript(ejv *extJSONValue) (code string, err error) {
	if ejv.t != TypeString {
		return "", fmt.Errorf("$code value should be string, but instead is %s", ejv.t)
	}

	return ejv.v.(string), nil
}

func parseMinMaxKey(ejv *extJSONValue, minmax string) error {
	if ejv.t != TypeInt32 {
		return fmt.Errorf("$%sKey value should be int32, but instead is %s", minmax, ejv.t)
	}

	if ejv.v.(int32) != 1 {
		return fmt.Errorf("$%sKey value must be 1, but instead is %d", minmax, ejv.v.(int32))
	}

	return nil
}

func parseObjectID(ejv *extJSONValue) (objectid.ObjectID, error) {
	if ejv.t != TypeString {
		return objectid.NilObjectID, fmt.Errorf("$oid value should be string, but instead is %s", ejv.t)
	}

	return objectid.FromHex(ejv.v.(string))
}

func parseRegex(ejv *extJSONValue) (pattern, options string, err error) {
	if ejv.t != TypeEmbeddedDocument {
		return "", "", fmt.Errorf("$regularExpression value should be object, but instead is %s", ejv.t)
	}

	regexObj := ejv.v.(*extJSONObject)
	patFound := false
	optFound := false

	for i, key := range regexObj.keys {
		val := regexObj.values[i]

		switch string(key) {
		case "pattern":
			if patFound {
				return "", "", errors.New("duplicate pattern key in $regularExpression")
			}

			if val.t != TypeString {
				return "", "", fmt.Errorf("$regularExpression pattern value should be string, but instead is %s", val.t)
			}

			pattern = val.v.(string)
			patFound = true
		case "options":
			if optFound {
				return "", "", errors.New("duplicate options key in $regularExpression")
			}

			if val.t != TypeString {
				return "", "", fmt.Errorf("$regularExpression options value should be string, but instead is %s", val.t)
			}

			options = val.v.(string)
			optFound = true
		default:
			return "", "", fmt.Errorf("invalid key in $regularExpression object: %s", key)
		}
	}

	if !patFound {
		return "", "", errors.New("missing pattern field in $regularExpression object")
	}

	if !optFound {
		return "", "", errors.New("missing options field in $regularExpression object")

	}

	return pattern, options, nil
}

func parseSymbol(ejv *extJSONValue) (string, error) {
	if ejv.t != TypeString {
		return "", fmt.Errorf("$symbol value should be string, but instead is %s", ejv.t)
	}

	return ejv.v.(string), nil
}

func parseTimestamp(ejv *extJSONValue) (t, i uint32, err error) {
	if ejv.t != TypeEmbeddedDocument {
		return 0, 0, fmt.Errorf("$timestamp value should be object, but instead is %s", ejv.t)
	}

	tsObj := ejv.v.(*extJSONObject)
	tFound := false
	iFound := false

	for j, key := range tsObj.keys {
		val := tsObj.values[j]

		switch key {
		case "t":
			if tFound {
				return 0, 0, errors.New("duplicate t key in $timestamp")
			}

			switch val.t {
			case TypeInt32:
				if val.v.(int32) < 0 {
					return 0, 0, fmt.Errorf("$timestamp t number should be uint32: %s", string(val.v.(int32)))
				}

				t = uint32(val.v.(int32))
				tFound = true
			case TypeInt64:
				if val.v.(int64) < 0 || uint32(val.v.(int64)) > math.MaxUint32 {
					return 0, 0, fmt.Errorf("$timestamp t number should be uint32: %s", string(val.v.(int32)))
				}

				t = uint32(val.v.(int64))
				tFound = true
			default:
				return 0, 0 , fmt.Errorf("$timestamp t value should be uint32, but instead is %s", val.t)
			}
		case "i":
			if iFound {
				return 0, 0, errors.New("duplicate i key in $timestamp")
			}

			switch val.t {
			case TypeInt32:
				if val.v.(int32) < 0 {
					return 0, 0, fmt.Errorf("$timestamp i number should be uint32: %s", string(val.v.(int32)))
				}

				i = uint32(val.v.(int32))
				iFound = true
			case TypeInt64:
				if val.v.(int64) < 0 || uint32(val.v.(int64)) > math.MaxUint32 {
					return 0, 0, fmt.Errorf("$timestamp i number should be uint32: %s", string(val.v.(int32)))
				}

				i = uint32(val.v.(int64))
				iFound = true
			default:
				return 0, 0 , fmt.Errorf("$timestamp i value should be uint32, but instead is %s", val.t)
			}

		default:
			return 0, 0, fmt.Errorf("invalid key in $timestamp object: %s", key)
		}
	}

	if !tFound {
		return 0, 0, errors.New("missing t field in $timestamp object")
	}

	if !iFound {
		return 0, 0, errors.New("missing i field in $timestamp object")
	}

	return t, i, nil
}

func parseUndefined(ejv *extJSONValue) error {
	if ejv.t != TypeBoolean {
		return fmt.Errorf("undefined value should be boolean, but instead is %s", ejv.t)
	}

	if !ejv.v.(bool) {
		return fmt.Errorf("$undefined balue boolean should be true, but instead is %v", ejv.v.(bool))
	}

	return nil
}

func parseObjectIDOld(data []byte, dataType jsonparser.ValueType) ([12]byte, error) {
	var oid [12]byte

	if dataType != jsonparser.String {
		return oid, fmt.Errorf("$oid value should be string, but instead is %s", dataType.String())
	}

	oidBytes, err := hex.DecodeString(string(data))
	if err != nil || len(oidBytes) != 12 {
		return oid, fmt.Errorf("invalid $oid value string: %s", string(data))
	}

	copy(oid[:], oidBytes[:])

	return oid, nil
}

func parseSymbolOld(data []byte, dataType jsonparser.ValueType) (string, error) {
	if dataType != jsonparser.String {
		return "", fmt.Errorf("$symbol value should be string, but instead is %s", dataType.String())
	}

	str, err := jsonparser.ParseString(data)
	if err != nil {
		return "", fmt.Errorf("invalid escaping in symbol string: %s", string(data))
	}

	return str, nil
}

func parseInt32Old(data []byte, dataType jsonparser.ValueType) (int32, error) {
	if dataType != jsonparser.String {
		return 0, fmt.Errorf("$numberInt value should be string, but instead is %s", dataType.String())
	}

	i, err := jsonparser.ParseInt(data)
	if err != nil {
		return 0, fmt.Errorf("invalid $numberInt number value: %s", string(data))
	}

	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, fmt.Errorf("$numberInt value should be int32Type but instead is int64: %d", i)
	}

	return int32(i), nil
}

func parseInt64Old(data []byte, dataType jsonparser.ValueType) (int64, error) {
	if dataType != jsonparser.String {
		return 0, fmt.Errorf("$numberLong value should be string, but instead is %s", dataType.String())
	}

	i, err := jsonparser.ParseInt(data)
	if err != nil {
		return 0, fmt.Errorf("invalid $numberLong number value: %s", string(data))
	}

	return int64(i), nil
}

func parseDoubleOld(data []byte, dataType jsonparser.ValueType) (float64, error) {
	if dataType != jsonparser.String {
		return 0, fmt.Errorf("$numberDouble value should be string, but instead is %s", dataType.String())
	}

	switch string(data) {
	case "Infinity":
		return math.Inf(1), nil
	case "-Infinity":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}

	f, err := jsonparser.ParseFloat(data)
	if err != nil {
		return 0, fmt.Errorf("invalid $numberDouble number value: %s", string(data))
	}

	return f, nil
}

func parseDecimalOld(data []byte, dataType jsonparser.ValueType) (decimal.Decimal128, error) {
	if dataType != jsonparser.String {
		return decimal.Decimal128{}, fmt.Errorf("$numberDecimal value should be string, but instead is %s", dataType.String())
	}

	d, err := decimal.ParseDecimal128(string(data))
	if err != nil {
		return decimal.Decimal128{}, fmt.Errorf("$invalid $numberDecimal string: %s", string(data))
	}

	return d, nil
}

func parseBinaryOld(data []byte, dataType jsonparser.ValueType) ([]byte, byte, error) {
	if dataType != jsonparser.Object {
		return nil, 0, fmt.Errorf("$binary value should be object, but instead is %s", dataType.String())
	}

	var b []byte
	var subType *int64

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "base64":
			if b != nil {
				return fmt.Errorf("duplicate base64 key in $binary: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$binary base64 value should be string, but instead is %s", dataType.String())
			}

			base64Bytes, err := base64.StdEncoding.DecodeString(string(value))
			if err != nil {
				return fmt.Errorf("invalid $binary base64 string: %s", string(value))
			}

			b = base64Bytes
		case "subType":
			if subType != nil {
				return fmt.Errorf("duplicate subType key in $binary: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$binary subType value should be string, but instead is %s", dataType.String())
			}

			i, err := strconv.ParseInt(string(value), 16, 64)
			if err != nil {
				return fmt.Errorf("invalid $binary subtype string: %s", string(value))
			}

			subType = &i
		default:
			return fmt.Errorf("invalid key in $binary object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	if b == nil {
		return nil, 0, fmt.Errorf("missing base64 field in $binary object: %s", string(data))
	}

	if subType == nil {
		return nil, 0, fmt.Errorf("missing subType field in $binary object: %s", string(data))

	}

	return b, byte(*subType), nil
}

func parseCode(data []byte, dataType jsonparser.ValueType) (string, error) {
	if dataType != jsonparser.String {
		return "", fmt.Errorf("$code value should be a string, but instead is %s", dataType.String())
	}

	str, err := jsonparser.ParseString(data)
	if err != nil {
		return "", fmt.Errorf("invalid escaping in symbol string: %s", string(data))
	}

	return str, nil
}

func parseScope(data []byte, dataType jsonparser.ValueType) (*builder.DocumentBuilder, error) {
	if dataType != jsonparser.Object {
		return nil, fmt.Errorf("$scope value should be an object, but instead is %s", dataType.String())
	}

	b := builder.NewDocumentBuilder()
	err := parseObjectToBuilder(b, string(data), nil, true)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func parseTimestampOld(data []byte, dataType jsonparser.ValueType) (uint32, uint32, error) {
	if dataType != jsonparser.Object {
		return 0, 0, fmt.Errorf("$timestamp value should be object, but instead is %s", dataType.String())
	}

	var time *uint32
	var inc *uint32

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "t":
			if time != nil {
				return fmt.Errorf("duplicate t key in $timestamp: %s", string(data))
			}

			if dataType != jsonparser.Number {
				return fmt.Errorf("$timestamp t value should be string, but instead is %s", dataType.String())
			}

			i, err := jsonparser.ParseInt(value)
			if err != nil {
				return fmt.Errorf("invalid $timestamp t number: %s", string(value))
			}

			if i < 0 || i > math.MaxUint32 {
				return fmt.Errorf("$timestamp t number should be uint32: %s", string(value))
			}

			u := uint32(i)
			time = &u

		case "i":
			if inc != nil {
				return fmt.Errorf("duplicate i key in $timestamp: %s", string(data))
			}

			if dataType != jsonparser.Number {
				return fmt.Errorf("$timestamp i value should be string, but instead is %s", dataType.String())
			}

			i, err := jsonparser.ParseInt(value)
			if err != nil {
				return fmt.Errorf("invalid $timestamp i number: %s", string(value))
			}

			if i < 0 || i > math.MaxUint32 {
				return fmt.Errorf("$timestamp i number should be uint32: %s", string(value))
			}

			u := uint32(i)
			inc = &u

		default:
			return fmt.Errorf("invalid key in $timestamp object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	if time == nil {
		return 0, 0, fmt.Errorf("missing t field in $timestamp object: %s", string(data))
	}

	if inc == nil {
		return 0, 0, fmt.Errorf("missing i field in $timestamp object: %s", string(data))
	}

	return *time, *inc, nil
}

func parseRegexOld(data []byte, dataType jsonparser.ValueType) (string, string, error) {
	if dataType != jsonparser.Object {
		return "", "", fmt.Errorf("$regularExpression value should be object, but instead is %s", dataType.String())
	}

	var pat *string
	var opt *string

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "pattern":
			if pat != nil {
				return fmt.Errorf("duplicate pattern key in $regularExpression: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$regularExpression pattern value should be string, but instead is %s", dataType.String())
			}

			str := string(value)
			pat = &str
		case "options":
			if opt != nil {
				return fmt.Errorf("duplicate options key in $regularExpression: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$regularExpression options value should be string, but instead is %s", dataType.String())
			}

			str := string(value)
			opt = &str
		default:
			return fmt.Errorf("invalid key in $regularExpression object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return "", "", err
	}

	if pat == nil {
		return "", "", fmt.Errorf("missing pattern field in $regularExpression object: %s", string(data))
	}

	if opt == nil {
		return "", "", fmt.Errorf("missing subType field in $regularExpression object: %s", string(data))

	}

	unescapedRegex, err := jsonparser.Unescape([]byte(*pat), nil)
	if err != nil {
		return "", "", fmt.Errorf("invalid escaped $regularExpression pattern: %s", *pat)
	}

	return string(unescapedRegex), *opt, nil
}

func parseDBPointerOld(data []byte, dataType jsonparser.ValueType) (string, [12]byte, error) {
	var oid [12]byte
	var ns *string
	oidFound := false

	if dataType != jsonparser.Object {
		return "", oid, fmt.Errorf("$dbPointer value should be object, but instead is: %s", dataType.String())
	}

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "$ref":
			if ns != nil {
				return fmt.Errorf("duplicate $ref key in $dbPointer: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$dbPointer $ref value should be string, but instead is %s", dataType.String())
			}

			str := string(value)
			ns = &str
		case "$id":
			if oidFound {
				return fmt.Errorf("duplicate $id key in $dbPointer: %s", string(data))
			}

			if dataType != jsonparser.Object {
				return fmt.Errorf("$dbPointer $id value should be object, but instead is %s", dataType.String())
			}

			id, err := parseDBPointerObjectID(value)
			if err != nil {
				return err
			}

			oid = id
			oidFound = true
		default:
			return fmt.Errorf("invalid key in $dbPointer object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return "", oid, err
	}

	if ns == nil {
		return "", oid, fmt.Errorf("missing $ref field in $dbPointer object: %s", string(data))
	}

	if !oidFound {
		return "", oid, fmt.Errorf("missing $id field in $dbPointer object: %s", string(data))
	}

	return *ns, oid, nil
}

func parseDBPointerObjectID(data []byte) ([12]byte, error) {
	var oid [12]byte
	oidFound := false

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "$oid":
			if oidFound {
				return fmt.Errorf("duplicate $id key in $dbPointer $oid: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$dbPointer $id $oid value should be string, but instead is %s", dataType.String())
			}

			var err error
			oid, err = parseObjectIDOld(value, dataType)
			if err != nil {
				return fmt.Errorf("invalid $dbPointer $id $oid value: %s", err)
			}

			oidFound = true
		default:
			return fmt.Errorf("invalid key in $dbPointer $id object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return oid, err
	}

	if !oidFound {
		return oid, fmt.Errorf("missing $oid field in $dbPointer $id object: %s", string(data))
	}

	return oid, nil
}

func parseDatetimeOld(data []byte, dataType jsonparser.ValueType) (int64, error) {
	switch dataType {
	case jsonparser.String:
		return parseDatetimeStringOld(data)
	case jsonparser.Object:
		return parseDatetimeObjectOld(data)
	}

	return 0, fmt.Errorf("$date value should be string or object, but instead is %s", dataType.String())
}

func parseDatetimeStringOld(data []byte) (int64, error) {
	t, err := time.Parse(rfc3339Milli, string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid $date value string: %s", string(data))
	}

	return t.Unix() * 1000, nil
}

func parseDatetimeObjectOld(data []byte) (int64, error) {
	var d *int64

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch string(key) {
		case "$numberLong":
			if d != nil {
				return fmt.Errorf("duplicate $numberLong key in $dbPointer: %s", string(data))
			}

			if dataType != jsonparser.String {
				return fmt.Errorf("$date $numberLong field should be string, but instead is %s", dataType.String())
			}

			i, err := parseInt64Old(value, dataType)
			if err != nil {
				return err
			}

			d = &i

		default:
			return fmt.Errorf("invalid key in $date object: %s", string(key))
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	if d == nil {
		return 0, fmt.Errorf("missing $numberLong field in $date object: %s", string(data))
	}

	return *d, nil
}

func parseRef(data []byte, dataType jsonparser.ValueType) (string, error) {
	if dataType != jsonparser.String {
		return "", fmt.Errorf("$ref value should be string, but instead is %s", dataType.String())
	}

	str, err := jsonparser.ParseString(data)
	if err != nil {
		return "", fmt.Errorf("invalid escaping in $ref string: %s", string(data))
	}

	return str, nil
}

func parseDB(data []byte, dataType jsonparser.ValueType) (string, error) {
	if dataType != jsonparser.String {
		return "", fmt.Errorf("$db value should be string, but instead is %s", dataType.String())
	}

	str, err := jsonparser.ParseString(data)
	if err != nil {
		return "", fmt.Errorf("invalid escaping in $db string: %s", string(data))
	}

	return str, nil
}

func parseMinKeyOld(data []byte, dataType jsonparser.ValueType) error {
	if dataType != jsonparser.Number {
		return fmt.Errorf("$minKey value should be number, but instead is %s", dataType.String())
	}

	i, err := jsonparser.ParseInt(data)
	if err != nil {
		return fmt.Errorf("$minkKey value number is invalid integer: %s", string(data))
	}

	if i != 1 {
		return fmt.Errorf("$minKey value must be 1, but instead is %d", i)
	}

	return nil
}

func parseMaxKeyOld(data []byte, dataType jsonparser.ValueType) error {
	if dataType != jsonparser.Number {
		return fmt.Errorf("$maxKey value should be number, but instead is %s", dataType.String())
	}

	i, err := jsonparser.ParseInt(data)
	if err != nil {
		return fmt.Errorf("$maxkKey value number is invalid integer: %s", string(data))
	}

	if i != 1 {
		return fmt.Errorf("$maxKey value must be 1, but instead is %d", i)
	}

	return nil
}

func parseUndefinedOld(data []byte, dataType jsonparser.ValueType) error {
	if dataType != jsonparser.Boolean {
		return fmt.Errorf("undefined value should be boolean, but instead is %s", dataType.String())
	}

	b, err := jsonparser.ParseBoolean(data)
	if err != nil {
		return fmt.Errorf("$undefined value boolean is invalid: %s", string(data))
	}

	if !b {
		return fmt.Errorf("$undefined balue boolean should be true, but instead is %v", b)
	}

	return nil
}
