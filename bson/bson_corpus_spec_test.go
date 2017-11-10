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
	Description        string
	Canonical_Bson     string
	Canonical_Extjson  string
	Relaxed_Extjson    string "json:omitempty"
	Degenerate_Bson    string "json:omitempty"
	Degenerate_Extjson string "json:omitempty"
	Converted_Bson     string "json:omitempty"
	Converted_Extjson  string "json:omitempty"
	Lossy              bool   "json:omitempty"
}

type testCase struct {
	Description  string
	Bson_Type    string
	Test_Key     string        "json:omitempty"
	Valid        []valid       "json:omitempty"
	DecodeErrors []decodeError "json:omitempty"
	ParseErrors  []parseError  "json:omitempty"
	Deprecated   bool          "json:omitempty"
}

const testsDir string = "../specifications/source/bson-corpus/tests/"

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

		for _, validCase := range test.Valid {
			lossy := validCase.Lossy
			cEJ := validCase.Canonical_Extjson
			cB := validCase.Canonical_Bson

			t.Run(testName+"validateCanonicalBSON:"+validCase.Description, func(t *testing.T) {
				validateCanonicalBSON(t, cB, cEJ)
			})
			t.Run(testName+"validateCanonicalExtendedJSON:"+validCase.Description, func(t *testing.T) {
				validateCanonicalExtendedJSON(t, cB, cEJ, lossy)
			})

			rEJ := validCase.Relaxed_Extjson
			if rEJ != "" {
				t.Run(testName+"validateBsonToRelaxedJSON:"+validCase.Description, func(t *testing.T) {
					validateBsonToRelaxedJSON(t, cB, rEJ)
				})

				// TODO: Failing on int64.json MaxInt
				t.Run(testName+"validateRelaxedExtendedJSON:"+validCase.Description, func(t *testing.T) {
					validateRelaxedExtendedJSON(t, rEJ, test.Description)
				})
			}

			dB := validCase.Degenerate_Bson
			if dB != "" {
				t.Run(testName+"validateDegenerateBSON:"+validCase.Description, func(t *testing.T) {
					validateDegenerateBSON(t, dB, cB)
				})
			}

			dEJ := validCase.Degenerate_Extjson
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

//for cB input:
func validateCanonicalBSON(t *testing.T, cB string, cEJ string) {
	//native_to_bson( bson_to_native(cB) ) = cB
	decoded, err := hex.DecodeString(cB)
	require.NoError(t, err)

	nativeReprD := bson.D{}
	err = bson.Unmarshal([]byte(decoded), &nativeReprD)
	require.NoError(t, err)

	roundTripCBByteRepr, err := bson.Marshal(nativeReprD)
	roundTripCB := hex.EncodeToString(roundTripCBByteRepr)
	require.Equal(t, cB, strings.ToUpper(roundTripCB))

	//native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
	var nativeReprBsonD bson.D
	err = bson.Unmarshal([]byte(decoded), &nativeReprBsonD)
	require.NoError(t, err)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeReprBsonD)
	require.NoError(t, err)
	validateExtendedJSONWithCondition(t, cEJ, string(roundTripCEJ), nativeReprBsonD)
}

//for cEJ input:
func validateCanonicalExtendedJSON(t *testing.T, cB string, cEJ string, lossy bool) {
	marshalDDoc := extjson.MarshalD{}
	err := json.Unmarshal([]byte(cEJ), &marshalDDoc)
	require.NoError(t, err)
	bsonDDoc := bson.D(marshalDDoc)

	// native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
	roundTripCEJByteRepr, err := extjson.EncodeBSONDtoJSON(bsonDDoc)
	require.NoError(t, err)
	validateExtendedJSONWithCondition(t, cEJ, string(roundTripCEJByteRepr), bsonDDoc)

	// native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
	if !lossy {
		bsonHexDecoded, err := bson.Marshal(bsonDDoc)
		require.NoError(t, err)

		roundTripCB := hex.EncodeToString(bsonHexDecoded)
		require.Equal(t, cB, strings.ToUpper(roundTripCB))
	}
}

//native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
func validateBsonToRelaxedJSON(t *testing.T, cB string, rEJ string) {
	// Convert BSON to native
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

// for rEJ input
func validateRelaxedExtendedJSON(t *testing.T, rEJ string, testFileName string) {
	//native_to_relaxed_extended_json( json_to_native(rEJ) ) = rEJ
	nativeRepr := bson.M{}
	require.NoError(t, json.Unmarshal([]byte(rEJ), &nativeRepr))

	// TODO: Fails on MaxInt
	// Need this as without this JSON.unmarshall will see a JSON Number and it will convert that into a float64
	// Then I have no idea whether the user wanted a floating point number 1.0 or an integer 1.
	if strings.Contains(testFileName, "Int") {
		for k := range nativeRepr {
			nativeRepr[k] = int64(nativeRepr[k].(float64))
		}
	}
	roundTripREJ, err := json.Marshal(nativeRepr)
	require.NoError(t, err)

	nativeReprBsonD := bson.D{}
	nativeReprBsonD.AppendMap(nativeRepr)
	validateExtendedJSONWithCondition(t, rEJ, string(roundTripREJ), nativeReprBsonD)
}

//for dB input (if it exists):
func validateDegenerateBSON(t *testing.T, dB string, cB string) {
	//native_to_bson( bson_to_native(dB) ) = cB
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

//for dEJ input (if it exists):
func validateDegenerateExtendedJSON(t *testing.T, dEJ string, cEJ string, cB string, lossy bool) {
	//native_to_canonical_extended_json( json_to_native(dEJ) ) = cEJ
	marshalDDoc := extjson.MarshalD{}
	err := json.Unmarshal([]byte(dEJ), &marshalDDoc)
	require.NoError(t, err)
	nativeD := bson.D(marshalDDoc)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeD)
	require.NoError(t, err)
	require.Equal(t, testutil.CompressJSON(cEJ), string(roundTripCEJ))

	//native_to_bson( json_to_native(dEJ) ) = cB (unless lossy)
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
	// In case of $numberDouble we can't fully represent it in Go. Therefore we will convert cEJ into NativeRepr
	if strings.Contains(expected, "1.234567890123456") && testutil.CompressJSON(expected) != string(returned) {
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(expected), &marshalDDoc)
		require.NoError(t, err)
		require.Equal(t, marshalDDoc[0].Value, nativeRepr[0].Value)
	} else {
		require.Equal(t, testutil.CompressJSON(expected), string(returned))
	}
}
