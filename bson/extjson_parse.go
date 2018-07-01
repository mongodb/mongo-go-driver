// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/mongodb/mongo-go-driver/bson/builder"
)

type parseState struct {
	wtype         wrapperType
	firstKey      bool
	currentValue  []byte
	docBuilder    *builder.DocumentBuilder
	subdocBuilder *builder.DocumentBuilder
	containingKey *string
	code          *string
	scope         *builder.DocumentBuilder
	refFound      bool
	idFound       bool
	dbFound       bool
}

func newParseState(b *builder.DocumentBuilder, containingKey *string) *parseState {
	return &parseState{
		firstKey:      true,
		docBuilder:    b,
		subdocBuilder: builder.NewDocumentBuilder(),
		containingKey: containingKey,
	}
}

func (s *parseState) parseElement(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
	wtype := wrapperKeyType(key)

	// dbRef can have regular elements after $id and $db appear
	if s.wtype == dbRef && s.idFound && s.dbFound && wtype == none {
		return parseDocElement(s.subdocBuilder, true)(key, value, dataType, offset)
	}

	if s.wtype != wtype && !s.firstKey {
		return fmt.Errorf(
			"previous key in the object were %s, but the %s is %s",
			s.wtype.String(),
			string(key),
			wtype.String(),
		)
	}

	s.wtype = wtype

	// The only wrapper types that allow more than one top-level key are code/CodeWithScope and dbRef
	if s.wtype != none && s.wtype != code && s.wtype != dbRef && !s.firstKey {
		return errors.New("%s wrapper object cannot have more than one key")
	}

	s.firstKey = true

	if s.wtype == none {
		return parseDocElement(s.subdocBuilder, true)(key, value, dataType, offset)
	}

	if s.containingKey == nil && s.wtype != dbRef {
		return errors.New("cannot parse wrapper type at top-level")
	}

	switch s.wtype {
	case objectID:
		oid, err := parseObjectID(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.ObjectID(*s.containingKey, oid))
	case symbol:
		str, err := parseSymbol(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Symbol(*s.containingKey, str))
	case int32Type:
		i, err := parseInt32(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Int32(*s.containingKey, i))
	case int64Type:
		i, err := parseInt64(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Int64(*s.containingKey, i))
	case double:
		f, err := parseDouble(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Double(*s.containingKey, f))
	case decimalType:
		d, err := parseDecimal(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Decimal(*s.containingKey, d))
	case binaryData:
		b, t, err := parseBinary(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.BinaryWithSubtype(*s.containingKey, b, t))
	case code:
		switch string(key) {
		case "$code":
			if s.code != nil {
				return errors.New("duplicate $code key in object")
			}

			code, err := parseCode(value, dataType)
			if err != nil {
				return err
			}

			s.code = &code
		case "$scope":
			if s.scope != nil {
				return errors.New("duplicate $scope key in object")
			}

			b, err := parseScope(value, dataType)

			if err != nil {
				return err
			}

			s.scope = b
		}
	case timestamp:
		t, i, err := parseTimestamp(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Timestamp(*s.containingKey, t, i))
	case regex:
		p, o, err := parseRegex(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Regex(*s.containingKey, p, o))
	case dbPointer:
		ns, oid, err := parseDBPointer(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.DBPointer(*s.containingKey, ns, oid))
	case dateTime:
		d, err := parseDatetime(value, dataType)
		if err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.DateTime(*s.containingKey, d))
	case dbRef:
		switch string(key) {
		case "$ref":
			if s.refFound {
				return errors.New("duplicate $ref key in object")
			}

			ref, err := parseRef(value, dataType)
			if err != nil {
				return err
			}

			s.subdocBuilder.Append(builder.C.String("$ref", ref))
			s.refFound = true
		case "$id":
			if s.idFound {
				return errors.New("duplicate $id field in object")
			}

			err := parseDocElement(s.subdocBuilder, true)([]byte("$id"), value, dataType, 0)
			if err != nil {
				return err
			}

			s.idFound = true
		case "$db":
			if s.dbFound {
				return errors.New("duplicate $db key in object")
			}

			db, err := parseDB(value, dataType)
			if err != nil {
				return err
			}

			s.subdocBuilder.Append(builder.C.String("$db", db))
			s.dbFound = true
		}
	case minKey:
		if err := parseMinKey(value, dataType); err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.MinKey(*s.containingKey))
	case maxKey:
		if err := parseMaxKey(value, dataType); err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.MaxKey(*s.containingKey))
	case undefined:
		if err := parseUndefined(value, dataType); err != nil {
			return err
		}

		s.docBuilder.Append(builder.C.Undefined(*s.containingKey))
	}

	return nil
}
