// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"strings"
)

// structTagParser returns the struct tags for a given reflect.StructField.
type structTagParser interface {
	parseStructTags(reflect.StructField) (*structTags, error)
	parseJSONStructTags(reflect.StructField) (*structTags, error)
}

// DefaultStructTagParser is the StructTagParser used by the StructCodec by default.
var DefaultStructTagParser = &StructTagParser{
	LookupEncoderOnMinSize:  retrieverOnMinSize{},
	LookupEncoderOnTruncate: retrieverOnTruncate{},
}

type retrieverOnMinSize struct{}

func (retrieverOnMinSize) LookupEncoder(reg EncoderRegistry, t reflect.Type) (ValueEncoder, error) {
	enc, err := reg.LookupEncoder(t)
	if err != nil {
		return enc, err
	}
	switch t.Kind() {
	case reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
		if codec, ok := enc.(*numCodec); ok {
			c := *codec
			c.minSize = true
			return &c, nil
		}
	}
	return enc, nil
}

type retrieverOnTruncate struct{}

func (retrieverOnTruncate) LookupEncoder(reg EncoderRegistry, t reflect.Type) (ValueEncoder, error) {
	enc, err := reg.LookupEncoder(t)
	if err != nil {
		return enc, err
	}
	switch t.Kind() {
	case reflect.Float32,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		if codec, ok := enc.(*numCodec); ok {
			c := *codec
			c.truncate = true
			return &c, nil
		}
	}
	return enc, nil
}

// structTags represents the struct tag fields that the StructCodec uses during
// the encoding and decoding process.
//
// In the case of a struct, the lowercased field name is used as the key for each exported
// field but this behavior may be changed using a struct tag. The tag may also contain flags to
// adjust the marshalling behavior for the field.
//
// The properties are defined below:
//
//	inline     Inline the field, which must be a struct or a map, causing all of its fields
//	           or keys to be processed as if they were part of the outer struct. For maps,
//	           keys must not conflict with the bson keys of other struct fields.
//
//	omitEmpty  Only include the field if it's not set to the zero value for the type or to
//	           empty slices or maps.
//
//	skip       This struct field should be skipped. This is usually denoted by parsing a "-"
//	           for the name.
type structTags struct {
	Name      string
	Inline    bool
	OmitEmpty bool
	Skip      bool

	LookupEncoderOnMinSize  EncoderRetriever
	LookupEncoderOnTruncate EncoderRetriever
}

// EncoderRetriever is used to look up ValueEncoder with given EncoderRegistry and reflect.Type.
type EncoderRetriever interface {
	LookupEncoder(EncoderRegistry, reflect.Type) (ValueEncoder, error)
}

// StructTagParser defines the encoder lookup bahavior when minSize and truncate tags are set.
type StructTagParser struct {
	LookupEncoderOnMinSize  EncoderRetriever
	LookupEncoderOnTruncate EncoderRetriever
}

// parseStructTags handles the bson struct tag. See the documentation for StructTags to see
// what each of the returned fields means.
//
// If there is no name in the struct tag fields, the struct field name is lowercased.
// The tag formats accepted are:
//
//	"[<key>][,<flag1>[,<flag2>]]"
//
//	`(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// An example:
//
//	type T struct {
//	    A bool
//	    B int    "myb"
//	    C string "myc,omitempty"
//	    D string `bson:",omitempty" json:"jsonkey"`
//	    E int64  ",minsize"
//	    F int64  "myf,omitempty,minsize"
//	}
//
// A struct tag either consisting entirely of '-' or with a bson key with a
// value consisting entirely of '-' will return a StructTags with Skip true and
// the remaining fields will be their default values.
func (p *StructTagParser) parseStructTags(sf reflect.StructField) (*structTags, error) {
	key := strings.ToLower(sf.Name)
	tag, ok := sf.Tag.Lookup("bson")
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}
	return p.parseTags(key, tag)
}

func (p *StructTagParser) parseTags(key string, tag string) (*structTags, error) {
	var st structTags
	if tag == "-" {
		st.Skip = true
		return &st, nil
	}

	for idx, str := range strings.Split(tag, ",") {
		if idx == 0 && str != "" {
			key = str
		}
		switch str {
		case "inline":
			st.Inline = true
		case "omitempty":
			st.OmitEmpty = true
		case "minsize":
			st.LookupEncoderOnMinSize = p.LookupEncoderOnMinSize
		case "truncate":
			st.LookupEncoderOnTruncate = p.LookupEncoderOnTruncate
		}
	}

	st.Name = key

	return &st, nil
}

// parseJSONStructTags parses the json tag instead on a field where the
// bson tag isn't available.
func (p *StructTagParser) parseJSONStructTags(sf reflect.StructField) (*structTags, error) {
	key := strings.ToLower(sf.Name)
	tag, ok := sf.Tag.Lookup("bson")
	if !ok {
		tag, ok = sf.Tag.Lookup("json")
	}
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}

	return p.parseTags(key, tag)
}
