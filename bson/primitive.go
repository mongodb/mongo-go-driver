// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on gopkg.in/mgo.v2/bson by Gustavo Niemeyer
// See THIRD-PARTY-NOTICES for original license terms.

package bson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Zeroer allows custom struct types to implement a report of zero
// state. All struct types that don't implement Zeroer or where IsZero
// returns false are considered to be not zero.
type Zeroer interface {
	IsZero() bool
}

// The following primitive types are similar to Go primitives for BSON types that
// do not have direct Go primitive representations.

// Binary represents a BSON binary value.
type Binary struct {
	Subtype byte
	Data    []byte
}

// Equal compares bp to bp2 and returns true if they are equal.
func (bp Binary) Equal(bp2 Binary) bool {
	if bp.Subtype != bp2.Subtype {
		return false
	}
	return bytes.Equal(bp.Data, bp2.Data)
}

// IsZero returns if bp is the empty Binary.
func (bp Binary) IsZero() bool {
	return bp.Subtype == 0 && len(bp.Data) == 0
}

// Undefined represents the BSON undefined value type.
type Undefined struct{}

// DateTime represents the BSON datetime value.
type DateTime int64

var (
	_ json.Marshaler   = DateTime(0)
	_ json.Unmarshaler = (*DateTime)(nil)
)

// MarshalJSON marshal to time type.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return d.Time().UTC().MarshalJSON()
}

// UnmarshalJSON creates a bson.DateTime from a JSON string.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	// Ignore "null" so that we can distinguish between a "null" value and
	// valid value that is the zero time (as reported by time.Time.IsZero).
	if string(data) == "null" {
		return nil
	}
	var t time.Time
	if err := t.UnmarshalJSON(data); err != nil {
		return err
	}
	*d = NewDateTimeFromTime(t)
	return nil
}

// Time returns the date as a time type.
func (d DateTime) Time() time.Time {
	return time.Unix(int64(d)/1000, int64(d)%1000*1000000)
}

// NewDateTimeFromTime creates a new DateTime from a Time.
func NewDateTimeFromTime(t time.Time) DateTime {
	return DateTime(t.Unix()*1e3 + int64(t.Nanosecond())/1e6)
}

// Null represents the BSON null value.
type Null struct{}

// Regex represents a BSON regex value.
type Regex struct {
	Pattern string
	Options string
}

func (rp Regex) String() string {
	return fmt.Sprintf(`{"pattern": "%s", "options": "%s"}`, rp.Pattern, rp.Options)
}

// Equal compares rp to rp2 and returns true if they are equal.
func (rp Regex) Equal(rp2 Regex) bool {
	return rp.Pattern == rp2.Pattern && rp.Options == rp2.Options
}

// IsZero returns if rp is the empty Regex.
func (rp Regex) IsZero() bool {
	return rp.Pattern == "" && rp.Options == ""
}

// DBPointer represents a BSON dbpointer value.
type DBPointer struct {
	DB      string
	Pointer ObjectID
}

func (d DBPointer) String() string {
	return fmt.Sprintf(`{"db": "%s", "pointer": "%s"}`, d.DB, d.Pointer)
}

// Equal compares d to d2 and returns true if they are equal.
func (d DBPointer) Equal(d2 DBPointer) bool {
	return d == d2
}

// IsZero returns if d is the empty DBPointer.
func (d DBPointer) IsZero() bool {
	return d.DB == "" && d.Pointer.IsZero()
}

// JavaScript represents a BSON JavaScript code value.
type JavaScript string

// Symbol represents a BSON symbol value.
type Symbol string

// CodeWithScope represents a BSON JavaScript code with scope value.
type CodeWithScope struct {
	Code  JavaScript
	Scope any
}

func (cws CodeWithScope) String() string {
	return fmt.Sprintf(`{"code": "%s", "scope": %v}`, cws.Code, cws.Scope)
}

// Timestamp represents a BSON timestamp value.
type Timestamp struct {
	T uint32
	I uint32
}

// After reports whether the time instant tp is after tp2.
func (tp Timestamp) After(tp2 Timestamp) bool {
	return tp.T > tp2.T || (tp.T == tp2.T && tp.I > tp2.I)
}

// Before reports whether the time instant tp is before tp2.
func (tp Timestamp) Before(tp2 Timestamp) bool {
	return tp.T < tp2.T || (tp.T == tp2.T && tp.I < tp2.I)
}

// Equal compares tp to tp2 and returns true if they are equal.
func (tp Timestamp) Equal(tp2 Timestamp) bool {
	return tp.T == tp2.T && tp.I == tp2.I
}

// IsZero returns if tp is the zero Timestamp.
func (tp Timestamp) IsZero() bool {
	return tp.T == 0 && tp.I == 0
}

// Compare compares the time instant tp with tp2. If tp is before tp2, it returns -1; if tp is after
// tp2, it returns +1; if they're the same, it returns 0.
func (tp Timestamp) Compare(tp2 Timestamp) int {
	switch {
	case tp.Equal(tp2):
		return 0
	case tp.Before(tp2):
		return -1
	default:
		return +1
	}
}

// MinKey represents the BSON minkey value.
type MinKey struct{}

// MaxKey represents the BSON maxkey value.
type MaxKey struct{}

// D is an ordered representation of a BSON document. This type should be used when the order of the elements matters,
// such as MongoDB command documents. If the order of the elements does not matter, an M should be used instead.
//
// Example usage:
//
//	bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
type D []E

func (d D) String() string {
	b, err := MarshalExtJSON(d, true, false)
	if err != nil {
		return ""
	}
	return string(b)
}

// MarshalJSON encodes D into JSON.
func (d D) MarshalJSON() ([]byte, error) {
	if d == nil {
		return json.Marshal(nil)
	}
	var err error
	var buf bytes.Buffer
	buf.Write([]byte("{"))
	enc := json.NewEncoder(&buf)
	for i, e := range d {
		err = enc.Encode(e.Key)
		if err != nil {
			return nil, err
		}
		buf.Write([]byte(":"))
		err = enc.Encode(e.Value)
		if err != nil {
			return nil, err
		}
		if i < len(d)-1 {
			buf.Write([]byte(","))
		}
	}
	buf.Write([]byte("}"))
	return json.RawMessage(buf.Bytes()).MarshalJSON()
}

// UnmarshalJSON decodes D from JSON.
func (d *D) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t == nil {
		*d = nil
		return nil
	}
	if v, ok := t.(json.Delim); !ok || v != '{' {
		return &json.UnmarshalTypeError{
			Value:  tokenString(t),
			Type:   reflect.TypeOf(D(nil)),
			Offset: dec.InputOffset(),
		}
	}
	*d, err = jsonDecodeD(dec)
	return err
}

// E represents a BSON element for a D. It is usually used inside a D.
type E struct {
	Key   string
	Value any
}

// M is an unordered representation of a BSON document. This type should be used when the order of the elements does not
// matter. This type is handled as a regular map[string]any when encoding and decoding. Elements will be
// serialized in an undefined, random order. If the order of the elements matters, a D should be used instead.
//
// Example usage:
//
//	bson.M{"foo": "bar", "hello": "world", "pi": 3.14159}
type M map[string]any

func (m M) String() string {
	b, err := MarshalExtJSON(m, true, false)
	if err != nil {
		return ""
	}
	return string(b)
}

// An A is an ordered representation of a BSON array.
//
// Example usage:
//
//	bson.A{"bar", "world", 3.14159, bson.D{{"qux", 12345}}}
type A []any

func jsonDecodeD(dec *json.Decoder) (D, error) {
	res := D{}
	for {
		var e E

		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := t.(string)
		if !ok {
			break
		}
		e.Key = key

		t, err = dec.Token()
		if err != nil {
			return nil, err
		}
		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '[':
				e.Value, err = jsonDecodeSlice(dec)
				if err != nil {
					return nil, err
				}
			case '{':
				e.Value, err = jsonDecodeD(dec)
				if err != nil {
					return nil, err
				}
			}
		default:
			e.Value = t
		}

		res = append(res, e)
	}
	return res, nil
}

func jsonDecodeSlice(dec *json.Decoder) ([]any, error) {
	var res []any
	done := false
	for !done {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '[':
				a, err := jsonDecodeSlice(dec)
				if err != nil {
					return nil, err
				}
				res = append(res, a)
			case '{':
				d, err := jsonDecodeD(dec)
				if err != nil {
					return nil, err
				}
				res = append(res, d)
			default:
				done = true
			}
		default:
			res = append(res, t)
		}
	}
	return res, nil
}

func tokenString(t json.Token) string {
	switch v := t.(type) {
	case json.Delim:
		switch v {
		case '{':
			return "object"
		case '[':
			return "array"
		}
	case bool:
		return "bool"
	case float64:
		return "number"
	case json.Number, string:
		return "string"
	}
	return "unknown"
}
