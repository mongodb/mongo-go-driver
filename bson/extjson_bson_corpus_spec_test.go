// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/pretty"
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

const dataDir = "../data"

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
	return bsonType == "0x02" || bsonType == "0x0D" || bsonType == "0x0F"
}

func escape(s string, escapeUnicode bool) string {
	newS := ""
	lastWasBackslash := false

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
		case '\\':
			newS += `\\`
		case '"':
			if lastWasBackslash {
				newS += `\"`
			} else {
				newS += `"`
			}
		default:
			if r < ' ' || (r > unicode.MaxASCII && escapeUnicode) {
				code := fmt.Sprintf("%04x", r)
				newS += `\u` + code
			} else {
				newS += string(r)
			}
		}

		if r == '\\' {
			lastWasBackslash = true
		} else {
			lastWasBackslash = false
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
	return fmt.Sprintf(`{ "%s": { "$numberDouble": "%s" } }`, key, formatDouble(expectedFloat))
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
			cB, err := hex.DecodeString(v.CanonicalBson)
			require.NoError(t, err)

			cEJ := v.CanonicalExtJSON

			// Normalize float strings
			if test.BsonType == "0x01" {
				cEJ = normalizeCanonicalDouble(t, *test.TestKey, cEJ)
			}

			// bson_to_canonical_extended_json(cB) = cEJ
			cEJ = string(pretty.Ugly([]byte(cEJ)))
			actualExtendedJSON, err := ToExtJSON(true, cB)
			require.NoError(t, err)

			actualCompactExtendedJSON := string(pretty.Ugly([]byte(actualExtendedJSON)))
			escaped := escape(actualCompactExtendedJSON, needsEscapedUnicode(test.BsonType))
			require.Equal(t, cEJ, escaped)

			// json_to_bson(cEJ) = cB (unless lossy)
			if v.Lossy == nil || !*v.Lossy {
				doc, err := ParseExtJSONObject(v.CanonicalExtJSON)
				require.NoError(t, err)

				actualBytes, err := doc.MarshalBSON()
				require.NoError(t, err)
				require.Len(t, cB, len(actualBytes))
				require.True(t, bytes.Equal(cB, actualBytes))
			}
		}
	})
}

func Test_BsonCorpus(t *testing.T) {
	for _, file := range findJSONFilesInDir(t, dataDir) {
		runTest(t, file)
	}
}
