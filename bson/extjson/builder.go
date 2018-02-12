// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package extjson

import (
	"fmt"

	"errors"

	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/builder"
	"github.com/mongodb/mongo-go-driver/bson/internal/jsonparser"
)

type docElementParser func([]byte, []byte, jsonparser.ValueType, int) error
type arrayElementParser func(int, []byte, jsonparser.ValueType, int) error

// ParseObjectToBuilder parses a JSON object string into a *builder.DocumentBuilder.
func ParseObjectToBuilder(s string) (*builder.DocumentBuilder, error) {
	b := builder.NewDocumentBuilder()
	err := parseObjectToBuilder(b, s, nil, true)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// ParseArrayToBuilder parses a JSON array string into a *builder.ArrayBuilder.
func ParseArrayToBuilder(s string) (*builder.ArrayBuilder, error) {
	return parseArrayToBuilder(s, true)
}

func getDocElementParser(b *builder.DocumentBuilder, containingKey *string, ext bool) (docElementParser, *parseState) {
	var p docElementParser
	var s *parseState

	if ext {
		s = newParseState(b, containingKey)
		p = s.parseElement
	} else {
		p = parseDocElement(b, false)
	}

	return p, s
}

func parseArrayToBuilder(s string, ext bool) (*builder.ArrayBuilder, error) {
	var b builder.ArrayBuilder

	_, err := jsonparser.ArrayEach([]byte(s), parseArrayElement(&b, ext))

	return &b, err
}

func parseObjectToBuilder(b *builder.DocumentBuilder, s string, containingKey *string, ext bool) error {
	p, st := getDocElementParser(b, containingKey, ext)
	err := jsonparser.ObjectEach([]byte(s), p)
	if err != nil {
		return err
	}

	if st != nil {
		switch st.wtype {
		case code:
			if st.code == nil {
				return errors.New("extjson object with $scope must also have $code")
			}

			if st.scope == nil {
				b.Append(builder.C.JavaScriptCode(*containingKey, *st.code))
			} else {
				scope := make([]byte, st.scope.RequiredBytes())
				_, err := st.scope.WriteDocument(scope)
				if err != nil {
					return fmt.Errorf("unable to write $scope document to bytes: %s", err)
				}

				b.Append(builder.C.CodeWithScope(*containingKey, *st.code, scope))
			}
		case dbRef:
			if !st.refFound || !st.idFound {
				return errors.New("extjson dbRef must have both $ref and $i")
			}

			fallthrough
		case none:
			if containingKey == nil {
				*b = *st.subdocBuilder
			} else {
				b.Append(builder.C.SubDocument(*containingKey, st.subdocBuilder))
			}
		}
	}

	return nil
}

func parseDocElement(b *builder.DocumentBuilder, ext bool) docElementParser {
	return func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		name := string(key)

		switch dataType {
		case jsonparser.String:
			unescaped, err := jsonparser.Unescape(value, nil)
			if err != nil {
				return fmt.Errorf(`unable to unescape string "%s": %s`, string(value), err)
			}

			b.Append(builder.C.String(name, string(unescaped)))

		case jsonparser.Number:
			i, err := jsonparser.ParseInt(value)
			if err == nil {
				b.Append(builder.C.Int64(name, i))
				break
			}

			f, err := jsonparser.ParseFloat(value)
			if err != nil {
				return fmt.Errorf("invalid JSON number: %s", string(value))
			}

			b.Append(builder.C.Double(name, f))

		case jsonparser.Object:
			err := parseObjectToBuilder(b, string(value), &name, ext)
			if err != nil {
				return fmt.Errorf("%s: %s", err, string(value))
			}

		case jsonparser.Array:
			array, err := ParseArrayToBuilder(string(value))
			if err != nil {
				return fmt.Errorf("invalid JSON array: %s", string(value))
			}

			b.Append(builder.C.Array(name, array))

		case jsonparser.Boolean:
			boolean, err := jsonparser.ParseBoolean(value)
			if err != nil {
				return fmt.Errorf("invalid JSON boolean: %s", string(value))
			}

			b.Append(builder.C.Boolean(name, boolean))

		case jsonparser.Null:
			b.Append(builder.C.Null(name))
		}

		return nil
	}
}

func parseArrayElement(b *builder.ArrayBuilder, ext bool) arrayElementParser {
	return func(index int, value []byte, dataType jsonparser.ValueType, offset int) error {
		name := strconv.FormatInt(int64(index), 10)

		return parseDocElement(&b.DocumentBuilder, ext)([]byte(name), value, dataType, offset)
	}
}
