// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"io"
)

const maxNestingDepth = 200

type jsonParseState byte

const (
	jpStartState = jsonParseState(iota)
	sawBeginObject
	sawEndObject
	sawBeginArray
	sawEndArray
	sawColon
	sawComma
	sawKey
	sawValue
	jpDoneState
	jpInvalidState
)

type jsonParseMode byte

const (
	jpObjectMode = jsonParseMode(iota)
	jpArrayMode
	jpInvalidMode
)

type extJSONValue struct {
	t Type
	v interface{}
}

type extJSONObject struct {
	keys   []string
	values []*extJSONValue
}

type extJSONParser struct {
	js *jsonScanner
	s  jsonParseState
	m  []jsonParseMode
	k  string
	v  *extJSONValue

	err      error
	depth    int
	maxDepth int
}

// newExtJSONParser returns a new extended JSON parser, ready to to begin
// parsing from the first character of the argued json input. It will not
// perform any read-ahead and will therefore not report any errors about
// malformed JSON at this point.
func newExtJSONParser(r io.Reader) *extJSONParser {
	return &extJSONParser{
		js:       &jsonScanner{r: r},
		s:        jpStartState,
		m:        []jsonParseMode{},
		maxDepth: maxNestingDepth,
	}
}

// peekType examines the next value and returns its BSON Type
func (ejp *extJSONParser) peekType() (Type, error) {
	var t Type
	var err error

	ejp.advanceState()
	switch ejp.s {
	case sawValue:
		t = ejp.v.t
	case sawBeginArray:
		t = TypeArray
	case jpInvalidState:
		err = ejp.err
	case sawComma:
		// in array mode, seeing a comma means we need to progress again to actually observe a type
		if ejp.peekMode() == jpArrayMode {
			return ejp.peekType()
		}
	case sawEndArray:
		// this would only be a valid state if we were in array mode, so return end-of-array error
		err = ErrEOA
	case sawBeginObject:
		// peek key to determine type
		ejp.advanceState()
		switch ejp.s {
		case sawEndObject: // empty embedded document
			t = TypeEmbeddedDocument
		case jpInvalidState:
			err = ejp.err
		case sawKey:
			t = wrapperKeyBSONType(ejp.k)

			if t == TypeJavaScript {
				// just saw $code, need to check for $scope at same level
				_, err := ejp.readValue(TypeJavaScript)

				if err != nil {
					break
				}

				switch ejp.s {
				case sawEndObject: // type is TypeJavaScript
				case sawComma:
					ejp.advanceState()
					if ejp.s == sawKey && ejp.k == "$scope" {
						t = TypeCodeWithScope
					} else {
						err = fmt.Errorf("invalid extended JSON: unexpected key %s in code object", ejp.k)
					}
				case jpInvalidState:
					err = ejp.err
				default:
					err = errors.New("invalid JSON")
				}
			}
		}
	}

	return t, err
}

// readKey parses the next key and its type and returns them
func (ejp *extJSONParser) readKey() (string, Type, error) {
	// advance to key (or return with error)
	switch ejp.s {
	case jpStartState:
		ejp.advanceState()
		if ejp.s == sawBeginObject {
			ejp.advanceState()
		}
	case sawBeginObject:
		ejp.advanceState()
	case sawValue, sawEndObject, sawEndArray:
		ejp.advanceState()
		switch ejp.s {
		case sawComma:
			ejp.advanceState()
		case sawEndObject, jpDoneState:
			return "", 0, ErrEOD
		case jpInvalidState:
			return "", 0, ejp.err
		default:
			return "", 0, errors.New("invalid JSON input")
		}
	case sawKey: // do nothing (key was peeked before)
	default:
		return "", 0, errors.New("invalid state to request key")
	}

	// read key
	var key string

	switch ejp.s {
	case sawKey:
		key = ejp.k
	case sawEndObject:
		return "", 0, ErrEOD
	case jpInvalidState:
		return "", 0, ejp.err
	default:
		return "", 0, errors.New("invalid state to request key")
	}

	// check for colon
	ejp.advanceState()
	if ejp.s != sawColon {
		return "", 0, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", key)
	}

	// peek at the value to determine type
	t, err := ejp.peekType()
	if err != nil {
		return "", 0, err
	}

	return key, t, nil
}

// readValue returns the value corresponding to the Type returned by peekType
func (ejp *extJSONParser) readValue(t Type) (*extJSONValue, error) {
	if ejp.s == jpInvalidState {
		return nil, ejp.err
	}

	var v *extJSONValue

	switch t {
	case TypeNull, TypeBoolean, TypeString:
		if ejp.s != sawValue {
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
		v = ejp.v
	case TypeInt32, TypeInt64, TypeDouble:
		// relaxed version allows these to be literal number values
		if ejp.s == sawValue {
			v = ejp.v
			break
		}
		fallthrough
	case TypeDecimal128, TypeSymbol, TypeObjectID, TypeMinKey, TypeMaxKey, TypeUndefined:
		switch ejp.s {
		case sawKey:
			// read colon
			ejp.advanceState()
			if ejp.s != sawColon {
				return nil, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", ejp.k)
			}

			// read value
			ejp.advanceState()
			if ejp.s != sawValue {
				return nil, fmt.Errorf("expected value for type %s", t.String())
			}
			v = ejp.v

			// read end object
			ejp.advanceState()
			if ejp.s != sawEndObject {
				return nil, fmt.Errorf("invalid JSON input: expected closing } after %s value for key \"%s\"", t, ejp.k)
			}
		default:
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
	case TypeBinary, TypeRegex, TypeTimestamp, TypeDBPointer:
		if ejp.s != sawKey {
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
		// read colon
		ejp.advanceState()
		if ejp.s != sawColon {
			return nil, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", ejp.k)
		}

		// read KV pairs
		keys, vals, err := ejp.readObject(2)
		if err != nil {
			return nil, err
		}

		v = &extJSONValue{t: TypeEmbeddedDocument, v: &extJSONObject{keys: keys, values: vals}}
	case TypeDateTime:
		switch ejp.s {
		case sawValue:
			v = ejp.v
		case sawKey:
			// read colon
			ejp.advanceState()
			if ejp.s != sawColon {
				return nil, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", ejp.k)
			}

			// read KV pairs
			keys, vals, err := ejp.readObject(1)
			if err != nil {
				return nil, err
			}

			v = &extJSONValue{t: TypeEmbeddedDocument, v: &extJSONObject{keys: keys, values: vals}}
		default:
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
	case TypeJavaScript:
		switch ejp.s {
		case sawKey:
			// read colon
			ejp.advanceState()
			if ejp.s != sawColon {
				return nil, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", ejp.k)
			}

			// read value
			ejp.advanceState()
			if ejp.s != sawValue {
				return nil, fmt.Errorf("expected value for type %s", t.String())
			}
			v = ejp.v

			// read end object or comma and just return
			ejp.advanceState()
		case sawEndObject:
			v = ejp.v
		default:
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
	case TypeCodeWithScope:
		if ejp.s == sawKey && ejp.k == "$scope" {
			v = ejp.v // this is the $code string from earlier

			// read colon
			ejp.advanceState()
			if ejp.s != sawColon {
				return nil, fmt.Errorf("invalid JSON input: missing colon after key \"%s\"", ejp.k)
			}

			// read {
			ejp.advanceState()
			if ejp.s != sawBeginObject {
				return nil, errors.New("invalid JSON input: $scope is not an embedded document")
			}
		} else {
			return nil, fmt.Errorf("invalid request to read type %s", t)
		}
	case TypeEmbeddedDocument, TypeArray:
		return nil, fmt.Errorf("invalid request to read full %s", t)
	}

	return v, nil
}

// readObject is a utility method for reading full objects of known (or expected) size
// it is useful for extended JSON types such as binary, datetime, regex, and timestamp
func (ejp *extJSONParser) readObject(numKeys int) ([]string, []*extJSONValue, error) {
	keys := make([]string, numKeys)
	vals := make([]*extJSONValue, numKeys)

	ejp.advanceState()
	if ejp.s != sawBeginObject {
		return nil, nil, errors.New("expected {")
	}

	for i := 0; i < numKeys; i++ {
		key, t, err := ejp.readKey()
		if err != nil {
			return nil, nil, err
		}

		if ejp.s == sawKey {
			v, err := ejp.readValue(t)
			if err != nil {
				return nil, nil, err
			}

			keys[i] = key
			vals[i] = v
			continue
		} else if ejp.s != sawValue {
			return nil, nil, errors.New("expected value")
		}

		keys[i] = key
		vals[i] = ejp.v
	}

	ejp.advanceState()
	if ejp.s != sawEndObject {
		return nil, nil, errors.New("expected }")
	}

	return keys, vals, nil
}

// advanceState reads the next JSON token from the scanner and transitions
// from the current state based on that token's type
func (ejp *extJSONParser) advanceState() {
	if ejp.s == jpDoneState || ejp.s == jpInvalidState {
		return
	}

	jt, err := ejp.js.nextToken()

	if err != nil {
		ejp.err = err
		ejp.s = jpInvalidState
		return
	}

	switch ejp.s {
	case jpStartState:
		ejp.transitionFromStart(jt)
	case sawBeginObject:
		ejp.transitionFromBeginObject(jt)
	case sawBeginArray:
		ejp.transitionFromBeginArray(jt)
	case sawEndObject, sawEndArray, sawValue:
		ejp.transitionFromEndToken(jt)
	case sawColon:
		ejp.transitionFromColon(jt)
	case sawComma:
		m := ejp.peekMode()
		if m == jpObjectMode {
			ejp.transitionFromCommaInObject(jt)
		} else if m == jpArrayMode {
			ejp.transitionFromCommaInArray(jt)
		} else {
			ejp.err = unexpectedEndTokenError(',', jt.p)
		}
	case sawKey:
		ejp.transitionFromKey(jt)
	}
}

// transitionFromStart examines the first JSON token's type and steps the
// parser to the next state
// valid tokens are: {, [, EOF, or a JSON value (string, number, JSON literal)
func (ejp *extJSONParser) transitionFromStart(jt *jsonToken) {
	switch jt.t {
	case beginObjectTokenType:
		ejp.s = sawBeginObject
		ejp.pushMode(jpObjectMode)
		ejp.depth = 1
	case beginArrayTokenType:
		ejp.s = sawBeginArray
		ejp.pushMode(jpArrayMode)
	case colonTokenType, commaTokenType, endObjectTokenType, endArrayTokenType:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	case eofTokenType:
		ejp.s = jpDoneState
	default:
		ejp.s = sawValue
		ejp.v = extendJSONToken(jt)
	}
}

// transitionFromBeginObject examines the type of the next JSON token after a
// '{' and steps the parser to the next state
// valid tokens are: } or a string (a key)
func (ejp *extJSONParser) transitionFromBeginObject(jt *jsonToken) {
	switch jt.t {
	case endObjectTokenType:
		ejp.s = sawEndObject
		ejp.depth--
		ejp.popMode() // this case only happens when we encounter {}
	case stringTokenType:
		ejp.s = sawKey
		ejp.k = jt.v.(string)
	default:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	}
}

// transitionFromEndToken examines the type of the next JSON token after a
// value or one of '}' or ']' and steps the parser to the next state
// valid tokens are: }, ], <comma>, or EOF
func (ejp *extJSONParser) transitionFromEndToken(jt *jsonToken) {
	switch jt.t {
	case endObjectTokenType:
		ejp.s = sawEndObject
		ejp.depth--

		m := ejp.popMode()
		if m != jpObjectMode {
			ejp.s = jpInvalidState
			ejp.err = unexpectedEndTokenError('}', jt.p)
		}
	case endArrayTokenType:
		ejp.s = sawEndArray

		m := ejp.popMode()
		if m != jpArrayMode {
			ejp.s = jpInvalidState
			ejp.err = unexpectedEndTokenError(']', jt.p)
		}
	case commaTokenType:
		ejp.s = sawComma
	case eofTokenType:
		ejp.s = jpDoneState
		if len(ejp.m) != 0 {
			ejp.err = errors.New("invalid JSON input; unexpected end of input")
			ejp.s = jpInvalidState
		}
	default:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	}
}

// transitionFromBeginArray examines the type of the next JSON token after a
// '[' and steps the parser to the next state
// valid tokens are: {, [, ], or a JSON value (string, number, JSON literal)
func (ejp *extJSONParser) transitionFromBeginArray(jt *jsonToken) {
	switch jt.t {
	case beginObjectTokenType:
		ejp.s = sawBeginObject
		ejp.pushMode(jpObjectMode)
		ejp.depth++
		if ejp.depth > ejp.maxDepth {
			ejp.err = fmt.Errorf("invalid JSON input. Position: %d, nesting too deep (%d levels)", jt.p, ejp.depth)
			ejp.s = jpInvalidState
		}
	case beginArrayTokenType:
		ejp.s = sawBeginArray
		ejp.pushMode(jpArrayMode)
	case endArrayTokenType:
		ejp.s = sawEndArray
		ejp.popMode() // this case only happens when we encounter []
	case eofTokenType, endObjectTokenType, colonTokenType, commaTokenType:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	default:
		ejp.s = sawValue
		ejp.v = extendJSONToken(jt)
	}
}

// transitionFromColon examines the type of the next JSON token after a ':'
// and steps the parser to the next state
// valid tokens are: {, [, or a JSON value (string, number, JSON literal)
func (ejp *extJSONParser) transitionFromColon(jt *jsonToken) {
	switch jt.t {
	case beginObjectTokenType:
		ejp.s = sawBeginObject
		ejp.pushMode(jpObjectMode)
		ejp.depth++
		if ejp.depth > ejp.maxDepth {
			ejp.err = fmt.Errorf("invalid JSON input. Position: %d, nesting too deep (%d levels)", jt.p, ejp.depth)
			ejp.s = jpInvalidState
		}
	case beginArrayTokenType:
		ejp.s = sawBeginArray
		ejp.pushMode(jpArrayMode)
	case endObjectTokenType, endArrayTokenType, colonTokenType, commaTokenType, eofTokenType:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	default:
		ejp.s = sawValue
		ejp.v = extendJSONToken(jt)
	}
}

// transitionFromCommaInObject examines the type of the next JSON token after
// a ',' in an object and steps the parser to the next state
// valid tokens are: a string (a key)
// (trailing commas not allowed according to RFC-8259)
func (ejp *extJSONParser) transitionFromCommaInObject(jt *jsonToken) {
	if jt.t == stringTokenType {
		ejp.s = sawKey
		ejp.k = jt.v.(string)
	} else {
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	}
}

// transitionFromCommaInArray examines the type of the next JSON token after
// a ',' in an array and steps the parser to the next state
// valid tokens are: {, [, or a JSON value (string, number, JSON literal)
// (trailing commas not allowed according to RFC-8259)
func (ejp *extJSONParser) transitionFromCommaInArray(jt *jsonToken) {
	switch jt.t {
	case beginObjectTokenType:
		ejp.s = sawBeginObject
		ejp.pushMode(jpObjectMode)
		ejp.depth++
		if ejp.depth > ejp.maxDepth {
			ejp.err = fmt.Errorf("invalid JSON input. Position: %d, nesting too deep (%d levels)", jt.p, ejp.depth)
			ejp.s = jpInvalidState
		}
	case beginArrayTokenType:
		ejp.s = sawBeginArray
		ejp.pushMode(jpArrayMode)
	case endObjectTokenType, endArrayTokenType, colonTokenType, commaTokenType, eofTokenType:
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	default:
		// a normal value in the array
		ejp.s = sawValue
		ejp.v = extendJSONToken(jt)
	}
}

// transitionFromKey examines the type of the next JSON token after a key and
// steps the parser to the next state
// valid tokens are: :
func (ejp *extJSONParser) transitionFromKey(jt *jsonToken) {
	if jt.t == colonTokenType {
		ejp.s = sawColon
	} else {
		ejp.err = fmt.Errorf("invalid JSON input. Position: %d", jt.p)
		ejp.s = jpInvalidState
	}
}

func (ejp *extJSONParser) pushMode(m jsonParseMode) {
	ejp.m = append(ejp.m, m)
}

func (ejp *extJSONParser) popMode() jsonParseMode {
	l := len(ejp.m)
	if l == 0 {
		return jpInvalidMode
	}

	m := ejp.m[l-1]
	ejp.m = ejp.m[:l-1]

	return m
}

func (ejp *extJSONParser) peekMode() jsonParseMode {
	l := len(ejp.m)
	if l == 0 {
		return jpInvalidMode
	}

	return ejp.m[l-1]
}

func unexpectedEndTokenError(c byte, p int) error {
	return fmt.Errorf("invalid JSON input; unexpected %c at %d", c, p)
}

func extendJSONToken(jt *jsonToken) *extJSONValue {
	var t Type

	switch jt.t {
	case int32TokenType:
		t = TypeInt32
	case int64TokenType:
		t = TypeInt64
	case doubleTokenType:
		t = TypeDouble
	case stringTokenType:
		t = TypeString
	case boolTokenType:
		t = TypeBoolean
	case nullTokenType:
		t = TypeNull
	default:
		return nil
	}

	return &extJSONValue{t: t, v: jt.v}
}
