// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/pretty"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

type testCase struct {
	Description  string                `json:"description"`
	BsonType     string                `json:"bson_type"`
	TestKey      *string               `json:"test_key"`
	Valid        []validityTestCase    `json:"valid"`
	DecodeErrors []decodeErrorTestCase `json:"decodeErrors"`
	ParseErrors  []parseErrorTestCase  `json:"parseErrors"`
	Deprecated   *bool                 `json:"deprecated"`
}

type validityTestCase struct {
	Description       string  `json:"description"`
	CanonicalBson     string  `json:"canonical_bson"`
	CanonicalExtJSON  string  `json:"canonical_extjson"`
	RelaxedExtJSON    *string `json:"relaxed_extjson"`
	DegenerateBSON    *string `json:"degenerate_bson"`
	DegenerateExtJSON *string `json:"degenerate_extjson"`
	ConvertedBSON     *string `json:"converted_bson"`
	ConvertedExtJSON  *string `json:"converted_extjson"`
	Lossy             *bool   `json:"lossy"`
}

type decodeErrorTestCase struct {
	Description string `json:"description"`
	Bson        string `json:"bson"`
}

type parseErrorTestCase struct {
	Description string `json:"description"`
	String      string `json:"string"`
}

const dataDir = "../../data"

var dvd DefaultValueDecoders
var dve DefaultValueEncoders

var dc = DecodeContext{Registry: NewRegistryBuilder().Build()}
var ec = EncodeContext{Registry: NewRegistryBuilder().Build()}

func findJSONFilesInDir(t *testing.T, dir string) []string {
	files := make([]string, 0)

	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}

func needsEscapedUnicode(bsonType string) bool {
	return bsonType == "0x02" || bsonType == "0x0D" || bsonType == "0x0E" || bsonType == "0x0F"
}

func escape(s string, escapeUnicode bool) string {
	newS := ""

	for _, r := range s {
		switch r {
		case 0x8:
			newS += `\b`
		case 0x9:
			newS += `\t`
		case 0xA:
			newS += `\n`
		case 0xC:
			newS += `\f`
		case 0xD:
			newS += `\r`
		default:
			if r < ' ' || (r > unicode.MaxASCII && escapeUnicode) {
				code := fmt.Sprintf("%04x", r)
				newS += `\u` + code
			} else {
				newS += string(r)
			}
		}
	}

	return newS
}

func normalizeCanonicalDouble(t *testing.T, key string, cEJ string) string {
	// Unmarshal string into map
	cEJMap := make(map[string]map[string]string)
	err := json.Unmarshal([]byte(cEJ), &cEJMap)
	require.NoError(t, err)

	// Parse the float contained by the map.
	expectedString := cEJMap[key]["$numberDouble"]
	expectedFloat, err := strconv.ParseFloat(expectedString, 64)

	// Normalize the string
	return fmt.Sprintf(`{"%s":{"$numberDouble":"%s"}}`, key, formatDouble(expectedFloat))
}

func normalizeRelaxedDouble(t *testing.T, key string, rEJ string) string {
	// Unmarshal string into map
	rEJMap := make(map[string]float64)
	err := json.Unmarshal([]byte(rEJ), &rEJMap)
	if err != nil {
		return normalizeCanonicalDouble(t, key, rEJ)
	}

	// Parse the float contained by the map.
	expectedFloat := rEJMap[key]

	// Normalize the string
	return fmt.Sprintf(`{"%s":%s}`, key, formatDouble(expectedFloat))
}

// nativeToBSON encodes the native bson.Document (doc) into canonical BSON and compares it to the expected
// canonical BSON (cB)
func nativeToBSON(t *testing.T, cB []byte, doc *bson.Document, testDesc, bType, docSrcDesc string) {
	actualB := new(bytes.Buffer)
	vw, err := NewBSONValueWriter(actualB)
	expectNoError(t, err, fmt.Sprintf("%s: creating ValueWriter", testDesc))
	err = dve.DocumentEncodeValue(ec, vw, doc)
	expectNoError(t, err, fmt.Sprintf("%s: encoding %s BSON", testDesc, bType))

	if diff := cmp.Diff(cB, actualB.Bytes()); diff != "" {
		t.Errorf("%s: 'native_to_bson(%s) = cB' failed (-want, +got):\n-%v\n+%v\n",
			testDesc, docSrcDesc, cB, actualB.Bytes())
		t.FailNow()
	}
}

// nativeToJSON encodes the native bson.Document (doc) into
func nativeToJSON(t *testing.T, ej string, doc *bson.Document, testDesc, ejType, ejShortName, docSrcDesc string) {
	actualEJ := new(strings.Builder)
	ejvw, err := NewExtJSONValueWriter(actualEJ, ejType != "relaxed")
	expectNoError(t, err, fmt.Sprintf("%s: creating %s ExtJSONValueWriter", testDesc, ejType))
	err = dve.DocumentEncodeValue(ec, ejvw, doc)
	expectNoError(t, err, fmt.Sprintf("%s: encoding %s extended JSON", testDesc, ejType))

	if diff := cmp.Diff(ej, actualEJ.String()); diff != "" {
		t.Errorf("%s: 'native_to_%s_extended_json(%s) = %s' failed (-want, +got):\n%s\n",
			testDesc, ejType, docSrcDesc, ejShortName, diff)
		t.FailNow()
	}
}

func runTest(t *testing.T, file string) {
	filepath := path.Join(dataDir, file)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	// Remove ".json" from filename.
	file = file[:len(file)-5]
	testName := "bson_corpus--" + file

	t.Run(testName, func(t *testing.T) {
		var test testCase
		require.NoError(t, json.Unmarshal(content, &test))

		for _, v := range test.Valid {
			// get canonical BSON
			cB, err := hex.DecodeString(v.CanonicalBson)
			expectNoError(t, err, fmt.Sprintf("%s: reading canonical BSON", v.Description))

			// get canonical extended JSON
			cEJ := escape(string(pretty.Ugly([]byte(v.CanonicalExtJSON))), needsEscapedUnicode(test.BsonType))
			if test.BsonType == "0x01" {
				cEJ = normalizeCanonicalDouble(t, *test.TestKey, cEJ)
			}

			/*** canonical BSON round-trip tests ***/
			doc := bson.NewDocument()
			err = dvd.DocumentDecodeValue(dc, NewValueReader(cB), &doc)
			expectNoError(t, err, fmt.Sprintf("%s: decoding canonical BSON", v.Description))

			// native_to_bson(bson_to_native(cB)) = cB
			nativeToBSON(t, cB, doc, v.Description, "canonical", "bson_to_native(cB)")

			// native_to_canonical_extended_json(bson_to_native(cB)) = cEJ
			nativeToJSON(t, cEJ, doc, v.Description, "canonical", "cEJ", "bson_to_native(cB)")

			// native_to_relaxed_extended_json(bson_to_native(cB)) = rEJ (if rEJ exists)
			if v.RelaxedExtJSON != nil {
				rEJ := escape(string(pretty.Ugly([]byte(*v.RelaxedExtJSON))), needsEscapedUnicode(test.BsonType))

				if test.BsonType == "0x01" {
					rEJ = normalizeRelaxedDouble(t, *test.TestKey, rEJ)
				}

				nativeToJSON(t, rEJ, doc, v.Description, "relaxed", "rEJ", "bson_to_native(cB)")
			}

			/*** canonical extended JSON round-trip tests ***/
			doc = bson.NewDocument()
			err = dvd.DocumentDecodeValue(dc, NewExtJSONValueReader(strings.NewReader(cEJ), true), &doc)
			expectNoError(t, err, fmt.Sprintf("%s: decoding canonical extended JSON", v.Description))

			// native_to_canonical_extended_json(json_to_native(cEJ)) = cEJ
			nativeToJSON(t, cEJ, doc, v.Description, "canonical", "cEJ", "json_to_native(cEJ)")

			// native_to_bson(json_to_native(cEJ)) = cb (unless lossy)
			if v.Lossy == nil || !*v.Lossy {
				nativeToBSON(t, cB, doc, v.Description, "canonical", "json_to_native(cEJ)")
			}

			/*** degenerate BSON round-trip tests (if exists) ***/
			if v.DegenerateBSON != nil {
				dB, err := hex.DecodeString(*v.DegenerateBSON)
				expectNoError(t, err, fmt.Sprintf("%s: reading degenerate BSON", v.Description))

				doc = bson.NewDocument()
				err = dvd.DocumentDecodeValue(dc, NewValueReader(dB), &doc)
				expectNoError(t, err, fmt.Sprintf("%s: decoding degenerate BSON", v.Description))

				nativeToBSON(t, cB, doc, v.Description, "degenerate", "bson_to_native(dB)")
			}

			/*** degenerate JSON round-trip tests (if exists) ***/
			if v.DegenerateExtJSON != nil {
				dEJ := escape(string(pretty.Ugly([]byte(*v.DegenerateExtJSON))), needsEscapedUnicode(test.BsonType))
				if test.BsonType == "0x01" {
					dEJ = normalizeCanonicalDouble(t, *test.TestKey, dEJ)
				}

				doc = bson.NewDocument()
				err = dvd.DocumentDecodeValue(dc, NewExtJSONValueReader(strings.NewReader(dEJ), true), &doc)
				expectNoError(t, err, fmt.Sprintf("%s: decoding degenerate canonical extended JSON", v.Description))

				// native_to_canonical_extended_json(json_to_native(dEJ)) = cEJ
				nativeToJSON(t, cEJ, doc, v.Description, "degenerate canonical", "cEJ", "json_to_native(dEJ)")

				// native_to_bson(json_to_native(dEJ)) = cB (unless lossy)
				if v.Lossy == nil || !*v.Lossy {
					nativeToBSON(t, cB, doc, v.Description, "canonical", "json_to_native(dEJ)")
				}
			}

			/*** relaxed extended JSON round-trip tests (if exists) ***/
			if v.RelaxedExtJSON != nil {
				rEJ := escape(string(pretty.Ugly([]byte(*v.RelaxedExtJSON))), needsEscapedUnicode(test.BsonType))

				if test.BsonType == "0x01" {
					rEJ = normalizeRelaxedDouble(t, *test.TestKey, rEJ)
				}

				doc = bson.NewDocument()
				err = dvd.DocumentDecodeValue(dc, NewExtJSONValueReader(strings.NewReader(rEJ), false), &doc)
				expectNoError(t, err, fmt.Sprintf("%s: decoding relaxed extended JSON", v.Description))

				// native_to_relaxed_extended_json(json_to_native(rEJ)) = rEJ
				nativeToJSON(t, rEJ, doc, v.Description, "relaxed", "eJR", "json_to_native(rEJ)")
			}
		}
	})
}

func Test_BsonCorpus(t *testing.T) {
	for _, file := range findJSONFilesInDir(t, dataDir) {
		runTest(t, file)
	}
}
