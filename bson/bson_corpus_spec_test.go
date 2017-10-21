package bson_test

import (
	"encoding/hex"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/extjson"
	"github.com/10gen/mongo-go-driver/bson/internal/json"
	"github.com/10gen/mongo-go-driver/bson/internal/testutil"
	"github.com/stretchr/testify/require"
)

type parseError struct {
	Description string
	String      string
}

type decodeError struct {
	Description string
	Bson        string
}

type valid struct {
	Description       string
	CanonicalBson     string `json:"canonical_bson"`
	CanonicalExtjson  string `json:"canonical_extjson"`
	RelaxedExtjson    string `json:"relaxed_extjson", omitempty`
	DegenerateBson    string `json:"degenerate_bson", omitempty`
	DegenerateExtjson string `json:"degenerate_extjson", omitempty`
	ConvertedBson     string `json:"converted_bson", omitempty`
	ConvertedExtjson  string `json:"converted_extjson", omitempty`
	Lossy             bool   `json:omitempty`
}

type testCase struct {
	Description  string
	BsonType     string        `json:"bson_type"`
	TestKey      string        `json:"test_key, omitempty"`
	Valid        []valid       `json:omitempty`
	DecodeErrors []decodeError `json:omitempty`
	ParseErrors  []parseError  `json:omitempty`
	Deprecated   bool          `json:omitempty`
}

const testsDir string = "../data/bson-corpus/"

func TestBSONSpec(t *testing.T) {
	for _, file := range testutil.FindJSONFilesInDir(t, testsDir) {
		runTest(t, file)
	}
}

func runTest(t *testing.T, filename string) {
	filepath := path.Join(testsDir, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	// remove .json extention
	filename = filename[:len(filename)-5]
	testName := filename + ";"

	t.Run(testName, func(t *testing.T) {
		var test testCase
		require.NoError(t, json.Unmarshal(content, &test))
		if test.Deprecated {
			return
		}

		bsonType := test.BsonType
		for _, validCase := range test.Valid {
			lossy := validCase.Lossy
			cEJ := validCase.CanonicalExtjson
			cB := validCase.CanonicalBson

			t.Run(testName+"validateCanonicalBSON:"+validCase.Description, func(t *testing.T) {
				validateCanonicalBSON(t, cB, cEJ)
			})
			t.Run(testName+"validateCanonicalExtendedJSON:"+validCase.Description, func(t *testing.T) {
				validateCanonicalExtendedJSON(t, cB, cEJ, lossy)
			})

			rEJ := validCase.RelaxedExtjson
			if rEJ != "" {
				t.Run(testName+"validateBsonToRelaxedJSON:"+validCase.Description, func(t *testing.T) {
					validateBsonToRelaxedJSON(t, cB, rEJ)
				})
				t.Run(testName+"validateRelaxedExtendedJSON:"+validCase.Description, func(t *testing.T) {
					validateRelaxedExtendedJSON(t, rEJ, bsonType)
				})
			}

			dB := validCase.DegenerateBson
			if dB != "" {
				t.Run(testName+"validateDegenerateBSON:"+validCase.Description, func(t *testing.T) {
					validateDegenerateBSON(t, dB, cB)
				})
			}

			dEJ := validCase.DegenerateExtjson
			if dEJ != "" {
				t.Run(testName+"validateDegenerateExtendedJSON:"+validCase.Description, func(t *testing.T) {
					validateDegenerateExtendedJSON(t, dEJ, cEJ, cB, lossy)
				})
			}
		}
		for _, decodeTest := range test.DecodeErrors {
			t.Run(testName+"testDecodeError:"+decodeTest.Description, func(t *testing.T) {
				testDecodeError(t, decodeTest.Bson)
			})
		}
		for _, parseTest := range test.ParseErrors {
			t.Run(testName+"testParseError:"+parseTest.Description, func(t *testing.T) {
				testParseError(t, parseTest.String)
			})
		}
	})
}

// This method validates round trip accuracy for canonical BSON and conversion from canonical BSON to canonical
// Extended JSON.
func validateCanonicalBSON(t *testing.T, cB string, cEJ string) {
	// 1. native_to_bson( bson_to_native(cB) ) = cB
	decoded, err := hex.DecodeString(cB)
	require.NoError(t, err)

	nativeReprD := bson.D{}
	err = bson.Unmarshal([]byte(decoded), &nativeReprD)
	require.NoError(t, err)

	roundTripCBByteRepr, err := bson.Marshal(nativeReprD)
	roundTripCB := hex.EncodeToString(roundTripCBByteRepr)
	require.Equal(t, cB, strings.ToUpper(roundTripCB))

	// 2. native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
	var nativeReprBsonD bson.D
	err = bson.Unmarshal([]byte(decoded), &nativeReprBsonD)
	require.NoError(t, err)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeReprBsonD)
	require.NoError(t, err)
	validateExtendedJSONWithCondition(t, cEJ, string(roundTripCEJ), nativeReprBsonD)
}

// This method validates round trip accuracy for canonical Extended JSON and conversion from canonical Extended JSON
// to canonical BSON.
func validateCanonicalExtendedJSON(t *testing.T, cB string, cEJ string, lossy bool) {
	marshalDDoc := extjson.MarshalD{}
	err := json.Unmarshal([]byte(cEJ), &marshalDDoc)
	require.NoError(t, err)
	bsonDDoc := bson.D(marshalDDoc)

	// 1. native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
	roundTripCEJByteRepr, err := extjson.EncodeBSONDtoJSON(bsonDDoc)
	require.NoError(t, err)
	validateExtendedJSONWithCondition(t, cEJ, string(roundTripCEJByteRepr), bsonDDoc)

	// 2. native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
	if !lossy {
		bsonHexDecoded, err := bson.Marshal(bsonDDoc)
		require.NoError(t, err)

		roundTripCB := hex.EncodeToString(bsonHexDecoded)
		require.Equal(t, cB, strings.ToUpper(roundTripCB))
	}
}

// This method validates conversion from canonical BSON into relaxed Extended JSON
func validateBsonToRelaxedJSON(t *testing.T, cB string, rEJ string) {
	// 1. native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
	decoded, err := hex.DecodeString(cB)
	require.NoError(t, err)

	nativeRepr := bson.M{}
	error := bson.Unmarshal([]byte(decoded), nativeRepr)
	require.NoError(t, error)

	roundTripREJ, err := json.Marshal(nativeRepr)
	require.NoError(t, err)

	bsonDDoc := bson.D{}
	bsonDDoc.AppendMap(nativeRepr)
	validateExtendedJSONWithCondition(t, rEJ, string(roundTripREJ), bsonDDoc)
}

// This method validates round trip accuracy for relaxed Extended JSON.
func validateRelaxedExtendedJSON(t *testing.T, rEJ string, bsonType string) {
	// 1. native_to_relaxed_extended_json( json_to_native(rEJ) ) = rEJ
	nativeRepr := bson.M{}
	require.NoError(t, json.Unmarshal([]byte(rEJ), &nativeRepr))

	roundTripREJ, err := json.Marshal(nativeRepr)
	require.NoError(t, err)
	nativeReprBsonD := bson.D{}
	nativeReprBsonD.AppendMap(nativeRepr)
	validateExtendedJSONWithCondition(t, rEJ, string(roundTripREJ), nativeReprBsonD)
}

// This method validates conversion from degenerate BSON into canonical BSON.
func validateDegenerateBSON(t *testing.T, dB string, cB string) {
	// 1. native_to_bson( bson_to_native(dB) ) = cB
	decoded, err := hex.DecodeString(dB)
	require.NoError(t, err)

	nativeRepr := bson.M{}
	err = bson.Unmarshal([]byte(decoded), nativeRepr)
	require.NoError(t, err)

	dBByteRepr, err := bson.Marshal(nativeRepr)
	require.NoError(t, err)
	roundTripDB := hex.EncodeToString(dBByteRepr)
	require.Equal(t, cB, strings.ToUpper(roundTripDB))
}

// This method validates conversion from degenerate Extended JSON into canonical Extended JSON, as well as conversion
// from degenerate Extended JSON into canonical Extended JSON.
func validateDegenerateExtendedJSON(t *testing.T, dEJ string, cEJ string, cB string, lossy bool) {
	// 1. native_to_canonical_extended_json( json_to_native(dEJ) ) = cEJ
	marshalDDoc := extjson.MarshalD{}
	err := json.Unmarshal([]byte(dEJ), &marshalDDoc)
	require.NoError(t, err)
	nativeD := bson.D(marshalDDoc)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeD)
	require.NoError(t, err)
	require.Equal(t, testutil.CompressJSON(cEJ), string(roundTripCEJ))

	// 1. native_to_bson( json_to_native(dEJ) ) = cB (unless lossy)
	if !lossy {
		dBByteRepr, err := bson.Marshal(nativeD)
		require.NoError(t, err)
		roundTripDB := hex.EncodeToString(dBByteRepr)
		require.Equal(t, cB, strings.ToUpper(roundTripDB))
	}
}

func testDecodeError(t *testing.T, b string) {
	decoded, err := hex.DecodeString(b)
	require.NoError(t, err)
	nativeReprD := bson.D{}
	err = bson.Unmarshal([]byte(decoded), &nativeReprD)
	require.Error(t, err)
}

func testParseError(t *testing.T, s string) {
	var nativeReprBsonD bson.D
	d := bson.Unmarshal([]byte(s), &nativeReprBsonD)
	require.Error(t, d)
}

// This method is required as due to the double values in double.json cannot be represented in 64bit floats in Go.
// Therefore we will check for the condition and if the condition holds, we'll convert the expected string into Go's
// native representation and compare those values.
func validateExtendedJSONWithCondition(t *testing.T, expected string, returned string, nativeRepr bson.D) {
	if testutil.CompressJSON(expected) != string(returned) {
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(expected), &marshalDDoc)
		require.NoError(t, err)
		require.Equal(t, marshalDDoc[0].Value, nativeRepr[0].Value)
	} else {
		require.Equal(t, testutil.CompressJSON(expected), string(returned))
	}
}
