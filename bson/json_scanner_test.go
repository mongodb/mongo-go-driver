// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestJsonScannerEmptyInput(t *testing.T) {
	js := &jsonScanner{r: strings.NewReader("")}
	c, err := js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerEmptyObject(t *testing.T) {
	js := &jsonScanner{r: strings.NewReader("{}")}

	c, err := js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerEmptyArray(t *testing.T) {
	js := &jsonScanner{r: strings.NewReader("[]")}

	c, err := js.nextToken()
	assert.Equal(t, beginArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerValidStrings(t *testing.T) {
	cases := [][]string{
		{`""`, ""},
		{`"string"`, "string"},
		{`"\"\\\'\/\b\f\n\r\t"`, "\"\\'/\b\f\n\r\t"},
	}

	for _, tc := range cases {
		js := &jsonScanner{r: strings.NewReader(tc[0])}

		c, err := js.nextToken()
		assert.Equal(t, stringTokenType, c.t)
		assert.Equal(t, tc[1], c.v.(string))
		noerr(t, err)

		c, err = js.nextToken()
		assert.Equal(t, eofTokenType, c.t)
		noerr(t, err)
	}
}

func TestJsonScannerInvalidStrings(t *testing.T) {
	cases := []string{
		`"missing`,
		`"\invalid"`,
		`"i\nv\alid"`,
	}

	for _, tc := range cases {
		js := &jsonScanner{r: strings.NewReader(tc)}

		c, err := js.nextToken()
		assert.Nil(t, c)
		assert.Error(t, err)
	}
}

func TestJsonScannerBooleanLiterals(t *testing.T) {
	cases := [][]interface{}{
		{"true", true},
		{"false", false},
	}

	for _, tc := range cases {
		in := tc[0].(string)
		expectedV := tc[1].(bool)

		js := &jsonScanner{r: strings.NewReader(in)}

		c, err := js.nextToken()
		assert.Equal(t, boolTokenType, c.t)
		assert.Equal(t, expectedV, c.v.(bool))
		noerr(t, err)

		c, err = js.nextToken()
		assert.Equal(t, eofTokenType, c.t)
		noerr(t, err)
	}
}

func TestJsonScannerNullLiteral(t *testing.T) {
	js := &jsonScanner{r: strings.NewReader("null")}

	c, err := js.nextToken()
	assert.Equal(t, nullTokenType, c.t)
	assert.Equal(t, nil, c.v)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerInvalidLiterals(t *testing.T) {
	cases := []string{
		"trueee",
		"tire",
		"nulll",
		"fals",
		"falsee",
		"fake",
		"bad",
	}

	for _, tc := range cases {
		js := &jsonScanner{r: strings.NewReader(tc)}

		c, err := js.nextToken()
		assert.Nil(t, c)
		assert.Error(t, err)
	}
}

func TestJsonScannerValidNumbers(t *testing.T) {
	cases := [][]interface{}{
		{"0", int32TokenType, 0},
		{"-0", int32TokenType, 0},
		{"1", int32TokenType, 1},
		{"-1", int32TokenType, -1},
		{"10", int32TokenType, 10},
		{"1234", int32TokenType, 1234},
		{"-10", int32TokenType, -10},
		{"-1234", int32TokenType, -1234},
		{"2147483648", int64TokenType, 2147483648},
		{"-2147483649", int64TokenType, -2147483649},
		{"0.0", doubleTokenType, 0.0},
		{"-0.0", doubleTokenType, 0.0},
		{"0.1", doubleTokenType, 0.1},
		{"0.1234", doubleTokenType, 0.1234},
		{"1.0", doubleTokenType, 1.0},
		{"-1.0", doubleTokenType, -1.0},
		{"1.234", doubleTokenType, 1.234},
		{"1.234", doubleTokenType, 1.234},
		{"1e10", doubleTokenType, 1e+10},
		{"1E10", doubleTokenType, 1e+10},
		{"1.2e10", doubleTokenType, 1.2e+10},
		{"1.2E10", doubleTokenType, 1.2e+10},
		{"-1.2e10", doubleTokenType, -1.2e+10},
		{"-1.2E10", doubleTokenType, -1.2e+10},
		{"1.2e+10", doubleTokenType, 1.2e+10},
		{"1.2E+10", doubleTokenType, 1.2e+10},
		{"-1.2e+10", doubleTokenType, -1.2e+10},
		{"-1.2E+10", doubleTokenType, -1.2e+10},
		{"1.2e-10", doubleTokenType, 1.2e-10},
		{"1.2E-10", doubleTokenType, 1.2e-10},
		{"-1.2e-10", doubleTokenType, -1.2e-10},
		{"-1.2E-10", doubleTokenType, -1.2e-10},
	}

	for _, tc := range cases {
		in := tc[0].(string)
		expectedT := tc[1].(jsonTokenType)

		js := &jsonScanner{r: strings.NewReader(in)}

		c, err := js.nextToken()

		var expectedV interface{}
		var actualV interface{}

		if expectedT == int32TokenType {
			expectedV = int32(tc[2].(int))
			actualV = c.v.(int32)
		} else if expectedT == int64TokenType {
			expectedV = int64(tc[2].(int))
			actualV = c.v.(int64)
		} else {
			expectedV = tc[2].(float64)
			actualV = c.v.(float64)
		}

		assert.Equal(t, expectedT, c.t)
		assert.Equal(t, expectedV, actualV)
		noerr(t, err)

		c, err = js.nextToken()
		assert.Equal(t, eofTokenType, c.t)
		noerr(t, err)
	}

}

func TestJsonScannerInvalidNumbers(t *testing.T) {
	cases := []string{
		"-",
		"--0",
		"-a",
		"00",
		"01",
		"0-",
		"1-",
		"0..",
		"0.-",
		"0..0",
		"0.1.0",
		"0e",
		"0e.",
		"0e1.",
		"0e1e",
		"0e+.1",
		"0e+1.",
		"0e+1e",
	}

	for _, tc := range cases {
		js := &jsonScanner{r: strings.NewReader(tc)}

		c, err := js.nextToken()
		assert.Nil(t, c)
		assert.Error(t, err)
	}
}

func TestJsonScannerValidObject(t *testing.T) {
	in := `{"key": "string", "key2": 2,
			"key3": {}, "key4": [], "key5": false }`

	js := &jsonScanner{r: strings.NewReader(in)}

	c, err := js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "key", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "string", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "key2", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, int32TokenType, c.t)
	assert.Equal(t, int32(2), c.v.(int32))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "key3", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "key4", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "key5", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, boolTokenType, c.t)
	assert.Equal(t, false, c.v.(bool))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerValidNestedObject(t *testing.T) {
	in := `{"abc":
				{ "sub": 1,
				  "xyz": {
							"k": {"v":true}
						}
				} }`

	js := &jsonScanner{r: strings.NewReader(in)}

	c, err := js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "abc", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "sub", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, int32TokenType, c.t)
	assert.Equal(t, int32(1), c.v.(int32))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "xyz", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "k", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "v", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, boolTokenType, c.t)
	assert.Equal(t, true, c.v.(bool))
	noerr(t, err)

	for i := 0; i < 4; i++ {
		c, err = js.nextToken()
		assert.Equal(t, endObjectTokenType, c.t)
		noerr(t, err)
	}

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerValidArray(t *testing.T) {
	in := `["one", 2, -3e-20, {}, [null]]`

	js := &jsonScanner{r: strings.NewReader(in)}

	c, err := js.nextToken()
	assert.Equal(t, beginArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, "one", c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, int32TokenType, c.t)
	assert.Equal(t, int32(2), c.v.(int32))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, doubleTokenType, c.t)
	assert.Equal(t, -3e-20, c.v.(float64))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, commaTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, beginArrayTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, nullTokenType, c.t)
	noerr(t, err)

	for i := 0; i < 2; i++ {
		c, err = js.nextToken()
		assert.Equal(t, endArrayTokenType, c.t)
		noerr(t, err)
	}

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}

func TestJsonScannerLongInput(t *testing.T) {
	// the jsonScanner's buffer is 512 bytes long; this test input is longer than that so the
	// buffer update is tested

	// length = 512
	const longKey = "abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyz" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqr"

	longInput := "{\"" + longKey + "\": 1}"

	js := &jsonScanner{r: strings.NewReader(longInput)}

	c, err := js.nextToken()
	assert.Equal(t, beginObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, stringTokenType, c.t)
	assert.Equal(t, longKey, c.v.(string))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, colonTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, int32TokenType, c.t)
	assert.Equal(t, int32(1), c.v.(int32))
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, endObjectTokenType, c.t)
	noerr(t, err)

	c, err = js.nextToken()
	assert.Equal(t, eofTokenType, c.t)
	noerr(t, err)
}
