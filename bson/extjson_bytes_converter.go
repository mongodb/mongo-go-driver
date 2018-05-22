// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/parser"
	"github.com/mongodb/mongo-go-driver/bson/parser/ast"
)

type extJSONWriter struct {
	*bytes.Buffer
	canonical bool
}

// ToExtJSON converts a BSON byte slice into an extended JSON string. If canonical is true, it
// will output canonical extended JSON. Otherwise, it will output relaxed extended JSON.
func ToExtJSON(canonical bool, bson []byte) (string, error) {
	p, err := parser.NewBSONParser(bytes.NewReader(bson))
	if err != nil {
		return "", err
	}

	doc, err := p.ParseDocument()
	if err != nil {
		return "", err
	}

	w := &extJSONWriter{bytes.NewBuffer([]byte{}), canonical}
	err = w.writeDocument(doc)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

func (w *extJSONWriter) writeStringLiteral(s string) error {
	s = `"` + s + `"`
	_, err := w.Write([]byte(s))

	return err
}

func (w *extJSONWriter) writeNonExtDocument(d *ast.Document) error {
	canonical := w.canonical
	w.canonical = false

	err := w.writeDocument(d)
	w.canonical = canonical

	return err
}

func (w *extJSONWriter) writeDocument(d *ast.Document) error {
	_, err := w.WriteRune('{')
	if err != nil {
		return err
	}

	for i, element := range d.EList {
		if i != 0 {
			_, err = w.WriteRune(',')
			if err != nil {
				return err
			}

		}

		switch e := element.(type) {
		case *ast.FloatElement:
			err = w.writeFloatElement(e)
		case *ast.StringElement:
			err = w.writeStringElement(e)
		case *ast.DocumentElement:
			err = w.writeDocumentElement(e)
		case *ast.ArrayElement:
			err = w.writeArrayElement(e)
		case *ast.BinaryElement:
			err = w.writeBinaryElement(e)
		case *ast.UndefinedElement:
			err = w.writeUndefinedElement(e)
		case *ast.ObjectIDElement:
			err = w.writeObjectIDElement(e)
		case *ast.BoolElement:
			err = w.writeBoolElement(e)
		case *ast.DateTimeElement:
			err = w.writeDatetimeElement(e)
		case *ast.NullElement:
			err = w.writeNullElement(e)
		case *ast.RegexElement:
			err = w.writeRegexElement(e)
		case *ast.DBPointerElement:
			err = w.writeDBPointerElement(e)
		case *ast.JavaScriptElement:
			err = w.writeJavaScriptElement(e)
		case *ast.SymbolElement:
			err = w.writeSymbolElement(e)
		case *ast.CodeWithScopeElement:
			err = w.writeCodeWithScopeElement(e)
		case *ast.Int32Element:
			err = w.writeInt32Element(e)
		case *ast.TimestampElement:
			err = w.writeTimestampElement(e)
		case *ast.Int64Element:
			err = w.writeInt64Element(e)
		case *ast.DecimalElement:
			err = w.writeDecimalElement(e)
		case *ast.MinKeyElement:
			err = w.writeMinKeyElement(e)
		case *ast.MaxKeyElement:
			err = w.writeMaxKeyElement(e)
		default:
			err = errors.New("unknown element type")
		}

		if err != nil {
			return err
		}
	}

	_, err = w.WriteRune('}')
	return err
}

func (w *extJSONWriter) writeArray(d *ast.Document) error {
	_, err := w.WriteRune('[')
	if err != nil {
		return err
	}

	for i, element := range d.EList {
		if i != 0 {
			_, err = w.WriteRune(',')
			if err != nil {
				return err
			}

		}

		switch e := element.(type) {
		case *ast.FloatElement:
			err = w.writeFloatValue(e.Double)
		case *ast.StringElement:
			err = w.writeStringValue(e.String)
		case *ast.DocumentElement:
			err = w.writeDocument(e.Document)
		case *ast.ArrayElement:
			err = w.writeArray(e.Array)
		case *ast.BinaryElement:
			err = w.writeBinaryValue(e.Binary.Data, e.Binary.Subtype)
		case *ast.UndefinedElement:
			err = w.writeUndefinedValue()
		case *ast.ObjectIDElement:
			err = w.writeObjectIDValue(e.ID)
		case *ast.BoolElement:
			err = w.writeBoolValue(e.Bool)
		case *ast.DateTimeElement:
			err = w.writeDatetimeValue(e.DateTime)
		case *ast.NullElement:
			err = w.writeNullValue()
		case *ast.RegexElement:
			err = w.writeRegexValue(e.RegexPattern.String, e.RegexOptions.String)
		case *ast.DBPointerElement:
			err = w.writeDBPointerValue(e.String, e.Pointer)
		case *ast.JavaScriptElement:
			err = w.writeJavaScriptValue(e.String)
		case *ast.SymbolElement:
			err = w.writeSymbolValue(e.String)
		case *ast.CodeWithScopeElement:
			err = w.writeCodeWithScopeValue(e.CodeWithScope.String, e.CodeWithScope.Document)
		case *ast.Int32Element:
			err = w.writeInt32Value(e.Int32)
		case *ast.TimestampElement:
			err = w.writeTimestampValue(e.Timestamp)
		case *ast.Int64Element:
			err = w.writeInt64Value(e.Int64)
		case *ast.DecimalElement:
			err = w.writeDecimalValue(e.Decimal128)
		case *ast.MinKeyElement:
			err = w.writeMinKeyValue()
		case *ast.MaxKeyElement:
			err = w.writeMaxKeyValue()
		default:
			err = errors.New("unknown element type")
		}

		if err != nil {
			return err
		}
	}

	_, err = w.WriteRune(']')
	return err
}

func (w *extJSONWriter) writeKey(s string) error {
	err := w.writeStringLiteral(s)
	if err != nil {
		return err
	}

	_, err = w.WriteRune(':')
	return err
}

func (w *extJSONWriter) writeFloatValue(f float64) error {
	s := formatDouble(f)

	var err error

	if w.canonical {
		d := newDoc(newStringElement("$numberDouble", s))
		err = w.writeDocument(d)
	} else {
		_, err = w.WriteString(s)
	}

	return err
}

func formatDouble(f float64) string {
	var s string
	if math.IsInf(f, 1) {
		s = "Infinity"
	} else if math.IsInf(f, -1) {
		s = "-Infinity"
	} else if math.IsNaN(f) {
		s = "NaN"
	} else {
		// Print exactly one decimalType place for integers; otherwise, print as many are necessary to
		// perfectly represent it.
		s = strconv.FormatFloat(f, 'G', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
	}

	return s
}

func (w *extJSONWriter) writeStringValue(s string) error {
	return w.writeStringLiteral(s)
}

func (w *extJSONWriter) writeBinaryValue(b []byte, t ast.BinarySubtype) error {
	b64 := base64.StdEncoding.EncodeToString(b)
	subType := fmt.Sprintf("%02x", byte(t))

	d := newDoc(
		newDocElement("$binary",
			newStringElement("base64", b64),
			newStringElement("subType", subType),
		),
	)

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeUndefinedValue() error {
	return w.writeDocument(newDoc(newBoolElement("$undefined", true)))
}

func (w *extJSONWriter) writeObjectIDValue(oid [12]byte) error {
	s := hex.EncodeToString(oid[:])
	d := newDoc(newStringElement("$oid", s))

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeBoolValue(b bool) error {
	_, err := w.WriteString(fmt.Sprintf("%v", b))
	return err
}

func (w *extJSONWriter) writeDatetimeValue(d int64) error {
	if w.canonical {
		return w.writeDocument(newDateDoc(d))
	}

	t := time.Unix(d/1e3, d%1e3*1e6)

	if t.Year() < 1970 || t.Year() > 9999 {
		return w.writeDocument(newDateDoc(d))
	}

	doc := newDoc(newStringElement("$date", t.Format(rfc3339Milli)))

	return w.writeDocument(doc)
}

func (w *extJSONWriter) writeNullValue() error {
	_, err := w.WriteString("null")
	return err
}

func (w *extJSONWriter) writeRegexValue(pattern string, options string) error {
	d := newDoc(
		newDocElement("$regularExpression",
			newStringElement("pattern", pattern),
			newStringElement("options", options),
		),
	)

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeDBPointerValue(ns string, oid [12]byte) error {
	d := newDoc(
		newDocElement("$dbPointer",
			newStringElement("$ref", ns),
			newObjectIDElement("$id", oid),
		),
	)

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeJavaScriptValue(code string) error {
	d := newDoc(newStringElement("$code", code))

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeSymbolValue(symbol string) error {
	d := newDoc(newStringElement("$symbol", symbol))

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeCodeWithScopeValue(code string, scope *ast.Document) error {
	d := newDoc(
		newStringElement("$code", code),
		newDocElement("$scope", scope.EList...),
	)

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeInt32Value(i int32) error {
	var err error
	numberString := strconv.FormatInt(int64(i), 10)

	if w.canonical {
		d := newDoc(newStringElement("$numberInt", numberString))
		err = w.writeDocument(d)
	} else {
		_, err = w.WriteString(numberString)
	}

	return err
}

func (w *extJSONWriter) writeTimestampValue(ts uint64) error {
	t := ts >> 32
	i := ts & 0xFFFFFFFF

	d := newDoc(
		newDocElement("$timestamp",
			newInt64Element("t", int64(t)),
			newInt64Element("i", int64(i)),
		),
	)

	return w.writeNonExtDocument(d)
}

func (w *extJSONWriter) writeInt64Value(i int64) error {
	var err error
	numberString := strconv.FormatInt(i, 10)

	if w.canonical {
		d := newDoc(newStringElement("$numberLong", numberString))
		err = w.writeDocument(d)
	} else {
		_, err = w.WriteString(numberString)
	}

	return err
}

func (w *extJSONWriter) writeMinKeyValue() error {
	d := newDoc(newInt32Element("$minKey", 1))

	return w.writeNonExtDocument(d)
}

func (w *extJSONWriter) writeMaxKeyValue() error {
	d := newDoc(newInt32Element("$maxKey", 1))

	return w.writeNonExtDocument(d)
}

func (w *extJSONWriter) writeDecimalValue(dec decimal.Decimal128) error {
	d := newDoc(newStringElement("$numberDecimal", dec.String()))

	return w.writeDocument(d)
}

func (w *extJSONWriter) writeFloatElement(e *ast.FloatElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeFloatValue(e.Double)
}

func (w *extJSONWriter) writeStringElement(e *ast.StringElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeStringLiteral(e.String)
}

func (w *extJSONWriter) writeDocumentElement(e *ast.DocumentElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeDocument(e.Document)
}

func (w *extJSONWriter) writeArrayElement(e *ast.ArrayElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeArray(e.Array)
}

func (w *extJSONWriter) writeBinaryElement(e *ast.BinaryElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeBinaryValue(e.Binary.Data, e.Binary.Subtype)
}

func (w *extJSONWriter) writeUndefinedElement(e *ast.UndefinedElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeUndefinedValue()
}

func (w *extJSONWriter) writeObjectIDElement(e *ast.ObjectIDElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeObjectIDValue(e.ID)
}

func (w *extJSONWriter) writeBoolElement(e *ast.BoolElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeBoolValue(e.Bool)
}

func (w *extJSONWriter) writeDatetimeElement(e *ast.DateTimeElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeDatetimeValue(e.DateTime)
}

func (w *extJSONWriter) writeNullElement(e *ast.NullElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeNullValue()
}

func (w *extJSONWriter) writeRegexElement(e *ast.RegexElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeRegexValue(e.RegexPattern.String, e.RegexOptions.String)
}

func (w *extJSONWriter) writeDBPointerElement(e *ast.DBPointerElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeDBPointerValue(e.String, e.Pointer)
}

func (w *extJSONWriter) writeJavaScriptElement(e *ast.JavaScriptElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeJavaScriptValue(e.String)
}

func (w *extJSONWriter) writeSymbolElement(e *ast.SymbolElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeSymbolValue(e.String)
}

func (w *extJSONWriter) writeCodeWithScopeElement(e *ast.CodeWithScopeElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeCodeWithScopeValue(e.CodeWithScope.String, e.CodeWithScope.Document)
}

func (w *extJSONWriter) writeInt32Element(e *ast.Int32Element) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeInt32Value(e.Int32)
}

func (w *extJSONWriter) writeTimestampElement(e *ast.TimestampElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeTimestampValue(e.Timestamp)
}

func (w *extJSONWriter) writeInt64Element(e *ast.Int64Element) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeInt64Value(e.Int64)
}

func (w *extJSONWriter) writeDecimalElement(e *ast.DecimalElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeDecimalValue(e.Decimal128)
}

func (w *extJSONWriter) writeMinKeyElement(e *ast.MinKeyElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeMinKeyValue()
}

func (w *extJSONWriter) writeMaxKeyElement(e *ast.MaxKeyElement) error {
	err := w.writeKey(e.Name.Key)
	if err != nil {
		return err
	}

	return w.writeMaxKeyValue()
}

func newDoc(elements ...ast.Element) *ast.Document {
	return &ast.Document{EList: elements}
}

func newDateDoc(d int64) *ast.Document {
	dateString := strconv.FormatInt(d, 10)

	return newDoc(newDocElement("$date",
		newStringElement("$numberLong", dateString)))
}

func newDocElement(key string, elements ...ast.Element) *ast.DocumentElement {
	return &ast.DocumentElement{Name: &ast.ElementKeyName{Key: key}, Document: newDoc(elements...)}
}

func newStringElement(key string, value string) *ast.StringElement {
	return &ast.StringElement{Name: &ast.ElementKeyName{Key: key}, String: value}
}

func newBoolElement(key string, value bool) *ast.BoolElement {
	return &ast.BoolElement{Name: &ast.ElementKeyName{Key: key}, Bool: value}
}

func newInt32Element(key string, i int32) *ast.Int32Element {
	return &ast.Int32Element{Name: &ast.ElementKeyName{Key: key}, Int32: i}
}

func newInt64Element(key string, i int64) *ast.Int64Element {
	return &ast.Int64Element{Name: &ast.ElementKeyName{Key: key}, Int64: i}
}

func newObjectIDElement(key string, oid [12]byte) *ast.ObjectIDElement {
	return &ast.ObjectIDElement{Name: &ast.ElementKeyName{Key: key}, ID: oid}
}
