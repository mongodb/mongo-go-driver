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
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
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

const dataDir = "../testdata/bson-corpus/"

func findJSONFilesInDir(dir string) ([]string, error) {
	files := make([]string, 0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files, nil
}

// seedExtJSON will add the byte representation of the "extJSON" string to the fuzzer's coprus.
func seedExtJSON(f *testing.F, extJSON string, extJSONType string, desc string) {
	jbytes, err := jsonToBytes(extJSON, extJSONType, desc)
	if err != nil {
		f.Fatalf("failed to convert JSON to bytes: %v", err)
	}

	f.Add(jbytes)
}

// seedTestCase will add the byte representation for each "extJSON" string of each valid test case to the fuzzer's
// corpus.
func seedTestCase(f *testing.F, tcase *testCase) {
	for _, vtc := range tcase.Valid {
		seedExtJSON(f, vtc.CanonicalExtJSON, "canonical", vtc.Description)

		// Seed the relaxed extended JSON.
		if vtc.RelaxedExtJSON != nil {
			seedExtJSON(f, *vtc.RelaxedExtJSON, "relaxed", vtc.Description)
		}

		// Seed the degenerate extended JSON.
		if vtc.DegenerateExtJSON != nil {
			seedExtJSON(f, *vtc.DegenerateExtJSON, "degenerate", vtc.Description)
		}

		// Seed the converted extended JSON.
		if vtc.ConvertedExtJSON != nil {
			seedExtJSON(f, *vtc.ConvertedExtJSON, "converted", vtc.Description)
		}
	}
}

// seedBSONCorpus will unmarshal the data from "testdata/bson-corpus" into a slice of "testCase" structs and then
// marshal the "*_extjson" field of each "validityTestCase" into a slice of bytes to seed the fuzz corpus.
func seedBSONCorpus(f *testing.F) {
	fileNames, err := findJSONFilesInDir(dataDir)
	if err != nil {
		f.Fatalf("failed to find JSON files in directory %q: %v", dataDir, err)
	}

	for _, fileName := range fileNames {
		filePath := path.Join(dataDir, fileName)

		file, err := os.Open(filePath)
		if err != nil {
			f.Fatalf("failed to open file %q: %v", filePath, err)
		}

		var tcase testCase
		if err := json.NewDecoder(file).Decode(&tcase); err != nil {
			f.Fatal(err)
		}

		seedTestCase(f, &tcase)
	}
}

func needsEscapedUnicode(bsonType string) bool {
	return bsonType == "0x02" || bsonType == "0x0D" || bsonType == "0x0E" || bsonType == "0x0F"
}

func unescapeUnicode(s, bsonType string) string {
	if !needsEscapedUnicode(bsonType) {
		return s
	}

	newS := ""

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			switch s[i+1] {
			case 'u':
				us := s[i : i+6]
				u, err := strconv.Unquote(strings.Replace(strconv.Quote(us), `\\u`, `\u`, 1))
				if err != nil {
					return ""
				}
				for _, r := range u {
					if r < ' ' {
						newS += fmt.Sprintf(`\u%04x`, r)
					} else {
						newS += string(r)
					}
				}
				i += 5
			default:
				newS += string(c)
			}
		default:
			if c > unicode.MaxASCII {
				r, size := utf8.DecodeRune([]byte(s[i:]))
				newS += string(r)
				i += size - 1
			} else {
				newS += string(c)
			}
		}
	}

	return newS
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
		if !strings.ContainsRune(s, 'E') && !strings.ContainsRune(s, '.') {
			s += ".0"
		}
	}

	return s
}

func normalizeCanonicalDouble(t *testing.T, key string, cEJ string) string {
	// Unmarshal string into map
	cEJMap := make(map[string]map[string]string)
	err := json.Unmarshal([]byte(cEJ), &cEJMap)
	require.NoError(t, err)

	// Parse the float contained by the map.
	expectedString := cEJMap[key]["$numberDouble"]
	expectedFloat, err := strconv.ParseFloat(expectedString, 64)
	require.NoError(t, err)

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

// bsonToNative decodes the BSON bytes (b) into a native Document
func bsonToNative(t *testing.T, b []byte, bType, testDesc string) D {
	var doc D
	err := Unmarshal(b, &doc)
	expectNoError(t, err, fmt.Sprintf("%s: decoding %s BSON", testDesc, bType))
	return doc
}

// nativeToBSON encodes the native Document (doc) into canonical BSON and compares it to the expected
// canonical BSON (cB)
func nativeToBSON(t *testing.T, cB []byte, doc D, testDesc, bType, docSrcDesc string) {
	actual, err := Marshal(doc)
	expectNoError(t, err, fmt.Sprintf("%s: encoding %s BSON", testDesc, bType))

	if diff := cmp.Diff(cB, actual); diff != "" {
		t.Errorf("%s: 'native_to_bson(%s) = cB' failed (-want, +got):\n-%v\n+%v\n",
			testDesc, docSrcDesc, cB, actual)
		t.FailNow()
	}
}

// jsonToNative decodes the extended JSON string (ej) into a native Document
func jsonToNative(ej, ejType, testDesc string) (D, error) {
	var doc D
	if err := UnmarshalExtJSON([]byte(ej), ejType != "relaxed", &doc); err != nil {
		return nil, fmt.Errorf("%s: decoding %s extended JSON: %w", testDesc, ejType, err)
	}
	return doc, nil
}

// jsonToBytes decodes the extended JSON string (ej) into canonical BSON and then encodes it into a byte slice.
func jsonToBytes(ej, ejType, testDesc string) ([]byte, error) {
	native, err := jsonToNative(ej, ejType, testDesc)
	if err != nil {
		return nil, err
	}

	b, err := Marshal(native)
	if err != nil {
		return nil, fmt.Errorf("%s: encoding %s BSON: %w", testDesc, ejType, err)
	}

	return b, nil
}

// nativeToJSON encodes the native Document (doc) into an extended JSON string
func nativeToJSON(t *testing.T, ej string, doc D, testDesc, ejType, ejShortName, docSrcDesc string) {
	actualEJ, err := MarshalExtJSON(doc, ejType != "relaxed", true)
	expectNoError(t, err, fmt.Sprintf("%s: encoding %s extended JSON", testDesc, ejType))

	if diff := cmp.Diff(ej, string(actualEJ)); diff != "" {
		t.Errorf("%s: 'native_to_%s_extended_json(%s) = %s' failed (-want, +got):\n%s\n",
			testDesc, ejType, docSrcDesc, ejShortName, diff)
		t.FailNow()
	}
}

func runTest(t *testing.T, file string) {
	filepath := path.Join(dataDir, file)
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)

	// Remove ".json" from filename.
	file = file[:len(file)-5]
	testName := "bson_corpus--" + file

	t.Run(testName, func(t *testing.T) {
		var test testCase
		require.NoError(t, json.Unmarshal(content, &test))

		t.Run("valid", func(t *testing.T) {
			for _, v := range test.Valid {
				t.Run(v.Description, func(t *testing.T) {
					// get canonical BSON
					cB, err := hex.DecodeString(v.CanonicalBson)
					expectNoError(t, err, fmt.Sprintf("%s: reading canonical BSON", v.Description))

					// get canonical extended JSON
					var compactEJ bytes.Buffer
					require.NoError(t, json.Compact(&compactEJ, []byte(v.CanonicalExtJSON)))
					cEJ := unescapeUnicode(compactEJ.String(), test.BsonType)
					if test.BsonType == "0x01" {
						cEJ = normalizeCanonicalDouble(t, *test.TestKey, cEJ)
					}

					/*** canonical BSON round-trip tests ***/
					doc := bsonToNative(t, cB, "canonical", v.Description)

					// native_to_bson(bson_to_native(cB)) = cB
					nativeToBSON(t, cB, doc, v.Description, "canonical", "bson_to_native(cB)")

					// native_to_canonical_extended_json(bson_to_native(cB)) = cEJ
					nativeToJSON(t, cEJ, doc, v.Description, "canonical", "cEJ", "bson_to_native(cB)")

					// native_to_relaxed_extended_json(bson_to_native(cB)) = rEJ (if rEJ exists)
					if v.RelaxedExtJSON != nil {
						var compactEJ bytes.Buffer
						require.NoError(t, json.Compact(&compactEJ, []byte(*v.RelaxedExtJSON)))
						rEJ := unescapeUnicode(compactEJ.String(), test.BsonType)
						if test.BsonType == "0x01" {
							rEJ = normalizeRelaxedDouble(t, *test.TestKey, rEJ)
						}

						nativeToJSON(t, rEJ, doc, v.Description, "relaxed", "rEJ", "bson_to_native(cB)")

						/*** relaxed extended JSON round-trip tests (if exists) ***/
						doc, err = jsonToNative(rEJ, "relaxed", v.Description)
						require.NoError(t, err)

						// native_to_relaxed_extended_json(json_to_native(rEJ)) = rEJ
						nativeToJSON(t, rEJ, doc, v.Description, "relaxed", "eJR", "json_to_native(rEJ)")
					}

					/*** canonical extended JSON round-trip tests ***/
					doc, err = jsonToNative(cEJ, "canonical", v.Description)
					require.NoError(t, err)

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

						doc = bsonToNative(t, dB, "degenerate", v.Description)

						// native_to_bson(bson_to_native(dB)) = cB
						nativeToBSON(t, cB, doc, v.Description, "degenerate", "bson_to_native(dB)")
					}

					/*** degenerate JSON round-trip tests (if exists) ***/
					if v.DegenerateExtJSON != nil {
						var compactEJ bytes.Buffer
						require.NoError(t, json.Compact(&compactEJ, []byte(*v.DegenerateExtJSON)))
						dEJ := unescapeUnicode(compactEJ.String(), test.BsonType)
						if test.BsonType == "0x01" {
							dEJ = normalizeCanonicalDouble(t, *test.TestKey, dEJ)
						}

						doc, err = jsonToNative(dEJ, "degenerate canonical", v.Description)
						require.NoError(t, err)

						// native_to_canonical_extended_json(json_to_native(dEJ)) = cEJ
						nativeToJSON(t, cEJ, doc, v.Description, "degenerate canonical", "cEJ", "json_to_native(dEJ)")

						// native_to_bson(json_to_native(dEJ)) = cB (unless lossy)
						if v.Lossy == nil || !*v.Lossy {
							nativeToBSON(t, cB, doc, v.Description, "canonical", "json_to_native(dEJ)")
						}
					}
				})
			}
		})

		t.Run("decode error", func(t *testing.T) {
			for _, d := range test.DecodeErrors {
				t.Run(d.Description, func(t *testing.T) {
					b, err := hex.DecodeString(d.Bson)
					expectNoError(t, err, d.Description)

					var doc D
					err = Unmarshal(b, &doc)

					// The driver unmarshals invalid UTF-8 strings without error. Loop over the unmarshalled elements
					// and assert that there was no error if any of the string or DBPointer values contain invalid UTF-8
					// characters.
					for _, elem := range doc {
						str, ok := elem.Value.(string)
						invalidString := ok && !utf8.ValidString(str)
						dbPtr, ok := elem.Value.(primitive.DBPointer)
						invalidDBPtr := ok && !utf8.ValidString(dbPtr.DB)

						if invalidString || invalidDBPtr {
							expectNoError(t, err, d.Description)
							return
						}
					}

					expectError(t, err, fmt.Sprintf("%s: expected decode error", d.Description))
				})
			}
		})

		t.Run("parse error", func(t *testing.T) {
			for _, p := range test.ParseErrors {
				t.Run(p.Description, func(t *testing.T) {
					s := unescapeUnicode(p.String, test.BsonType)
					if test.BsonType == "0x13" {
						s = fmt.Sprintf(`{"decimal128": {"$numberDecimal": "%s"}}`, s)
					}

					switch test.BsonType {
					case "0x00", "0x05", "0x13":
						var doc D
						err := UnmarshalExtJSON([]byte(s), true, &doc)
						// Null bytes are validated when marshaling to BSON
						if strings.Contains(p.Description, "Null") {
							_, err = Marshal(doc)
						}
						expectError(t, err, fmt.Sprintf("%s: expected parse error", p.Description))
					default:
						t.Errorf("Update test to check for parse errors for type %s", test.BsonType)
						t.Fail()
					}
				})
			}
		})
	})
}

func Test_BsonCorpus(t *testing.T) {
	jsonFiles, err := findJSONFilesInDir(dataDir)
	if err != nil {
		t.Fatalf("error finding JSON files in %s: %v", dataDir, err)
	}

	for _, file := range jsonFiles {
		runTest(t, file)
	}
}

func expectNoError(t *testing.T, err error, desc string) {
	if err != nil {
		t.Helper()
		t.Errorf("%s: Unepexted error: %v", desc, err)
		t.FailNow()
	}
}

func expectError(t *testing.T, err error, desc string) {
	if err == nil {
		t.Helper()
		t.Errorf("%s: Expected error", desc)
		t.FailNow()
	}
}

func TestRelaxedUUIDValidation(t *testing.T) {
	testCases := []struct {
		description       string
		canonicalExtJSON  string
		degenerateExtJSON string
		expectedErr       string
	}{
		{
			"valid uuid",
			"{\"x\" : { \"$binary\" : {\"base64\" : \"c//SZESzTGmQ6OfR38A11A==\", \"subType\" : \"04\"}}}",
			"{\"x\" : { \"$uuid\" : \"73ffd264-44b3-4c69-90e8-e7d1dfc035d4\"}}",
			"",
		},
		{
			"invalid uuid--no hyphens",
			"",
			"{\"x\" : { \"$uuid\" : \"73ffd26444b34c6990e8e7d1dfc035d4\"}}",
			"$uuid value does not follow RFC 4122 format regarding length and hyphens",
		},
		{
			"invalid uuid--trailing hyphens",
			"",
			"{\"x\" : { \"$uuid\" : \"73ffd264-44b3-4c69-90e8-e7d1dfc035--\"}}",
			"$uuid value does not follow RFC 4122 format regarding length and hyphens",
		},
		{
			"invalid uuid--malformed hex",
			"",
			"{\"x\" : { \"$uuid\" : \"q3@fd26l-44b3-4c69-90e8-e7d1dfc035d4\"}}",
			"$uuid value does not follow RFC 4122 format regarding hex bytes: encoding/hex: invalid byte: U+0071 'q'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// get canonical extended JSON (if provided)
			cEJ := ""
			if tc.canonicalExtJSON != "" {
				var compactCEJ bytes.Buffer
				require.NoError(t, json.Compact(&compactCEJ, []byte(tc.canonicalExtJSON)))
				cEJ = unescapeUnicode(compactCEJ.String(), "0x05")
			}

			// get degenerate extended JSON
			var compactDEJ bytes.Buffer
			require.NoError(t, json.Compact(&compactDEJ, []byte(tc.degenerateExtJSON)))
			dEJ := unescapeUnicode(compactDEJ.String(), "0x05")

			// convert dEJ to native doc
			var doc D
			err := UnmarshalExtJSON([]byte(dEJ), true, &doc)

			if tc.expectedErr != "" {
				assert.Equal(t, tc.expectedErr, err.Error(), "expected error %v, got %v", tc.expectedErr, err)
			} else {
				assert.Nil(t, err, "expected no error, got error: %v", err)

				// Marshal doc into extended JSON and compare with cEJ
				nativeToJSON(t, cEJ, doc, tc.description, "degenerate canonical", "cEJ", "json_to_native(dEJ)")
			}
		})
	}

}
