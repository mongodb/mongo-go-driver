package bson_test

import (
	"testing"
	"path"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"encoding/json"
	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/extjson"
	"encoding/hex"
	"github.com/10gen/stitch/utils/xjson"
	"bytes"
	"fmt"
	"strings"
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

const DEBUG = true;

const testsDir string = "../specifications/source/bson-corpus/tests/"

func TestBSONSpec(t *testing.T) {
	for _, file := range FindJSONFilesInDir(t, testsDir) {
		runTest(t, file)
	}
}

func FindJSONFilesInDir(t *testing.T, dir string) []string {
	files := make([]string, 0)
	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}


		if (entry.Name() != "decimal128-1.json") {
			continue
		}


		files = append(files, entry.Name())
	}
	return files
}

func runTest(t *testing.T, filename string) {
	filepath := path.Join(testsDir, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	// remove .json extention
	filename = filename[:len(filename) - 5]
	testName := filename + ";"

	t.Run(testName, func(t *testing.T) {
		var test testCase
		require.NoError(t, json.Unmarshal(content, &test))
		//t.Log("TestDescription", test.Description)

		if (test.Deprecated) {
			return
		}

		//printTestCaseData(t, test)



		for _, validCase := range test.Valid {
			//t.Log("\n\nDescription", validCase.Description)

			cEJ := validCase.Canonical_Extjson
			cB := validCase.Canonical_Bson

			t.Run(testName+"validateCanonicalBSON", func(t *testing.T) {
				validateCanonicalBSON(t, cB, cEJ)
			})


			rEJ := validCase.Relaxed_Extjson
			if (validCase.Relaxed_Extjson != "") {
				t.Run(testName+"validateBsonToRelaxedJson", func(t *testing.T) {
					validateBsonToRelaxedJson(t, cB, rEJ)
				})
				t.Run(testName+"validateREJ", func(t *testing.T) {
					validateREJ(t, rEJ)
				})
			}
			t.Run(testName+"validateCanonicalExtendedJSON", func(t *testing.T) {
				validateCanonicalExtendedJSON(t, cB, cEJ)
			})
		}


	})
}


//for cB input:
//native_to_bson( bson_to_native(cB) ) = cB
//native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
func validateCanonicalBSON(t *testing.T, cB string, cEJ string) {
	// Convert BSON to native then back to bson
	decoded, err := hex.DecodeString(cB)
	require.NoError(t, err)

	nativeRepr := bson.M{}
	err = bson.Unmarshal([]byte(decoded), nativeRepr)
	require.NoError(t, err)

	roundTripCBByteRepr, err := bson.Marshal(nativeRepr);
	roundTripCB := hex.EncodeToString(roundTripCBByteRepr);
	require.Equal(t, cB, strings.ToUpper(roundTripCB))

	// Convert native to extended JSON
	var nativeReprBsonD bson.D // Need this new bson.D object as encodeBsonDtoJson wants a Bson.D
	err = bson.Unmarshal([]byte(decoded), &nativeReprBsonD)
	require.NoError(t, err)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeReprBsonD)
	require.NoError(t, err)
	require.Equal(t, compressJSON(cEJ), string(roundTripCEJ))
}

//native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
func validateBsonToRelaxedJson(t *testing.T, cB string, rEJ string) {
	// Convert BSON to native
	decoded, err := hex.DecodeString(cB)
	require.NoError(t, err)

	nativeRepr := bson.M{}
	error := bson.Unmarshal([]byte(decoded), nativeRepr)
	require.NoError(t, error)

	roundTripREJ, err := json.Marshal(xjson.NewValueOf(nativeRepr))
	require.NoError(t, err)
	require.Equal(t, compressJSON(rEJ), string(roundTripREJ))
}

//for cEJ input:
//1. native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
//2. native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
func validateCanonicalExtendedJSON(t *testing.T, cB string, cEJ string) {
	marshalDDoc := extjson.MarshalD{}
	json.Unmarshal([]byte(cEJ), &marshalDDoc)

	bsonDDoc := bson.D(marshalDDoc)
	bsonHexDecoded, err := bson.Marshal(bsonDDoc);
	require.NoError(t, err)

	roundTripCB := hex.EncodeToString(bsonHexDecoded)
	require.Equal(t, cB, strings.ToUpper(roundTripCB))

	// Next do cEJ => native => cEJ
	roundTripCEJByteRepr, err := extjson.EncodeBSONDtoJSON(bsonDDoc)
	require.NoError(t, err)
	require.Equal(t, compressJSON(cEJ), string(roundTripCEJByteRepr))
}

//for rEJ input (if it exists):
//native_to_relaxed_extended_json( json_to_native(rEJ) ) = rEJ
func validateREJ(t *testing.T, rEJ string) {
	debuglog(t, "rEJ", rEJ)

	nativeRepr := bson.M{}
	require.NoError(t, json.Unmarshal([]byte(rEJ), &nativeRepr))
	debuglog(t, "nativeRepr", nativeRepr)
	roundTripREJ, err := json.Marshal(nativeRepr)
	debuglog(t, "roundTripREJ", roundTripREJ)
	debuglog(t, "roundTripREJ", string(roundTripREJ))
	require.NoError(t, err)
	require.Equal(t, compressJSON(rEJ), string(roundTripREJ))
}

func compressJSON(js string) string {
	var json_bytes = []byte(js)
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, json_bytes); err != nil {
		fmt.Println(err)
	}
	return buffer.String()
}



func debuglog(t *testing.T, desc string, ob interface{}) {
	if (DEBUG) {
		t.Log(desc, ob)
	}
}

func printTestCaseData(t *testing.T, test testCase) {
	t.Log(test.Description)
	t.Log(test.Bson_Type)
	t.Log(test.Test_Key)
	t.Log(test.Valid)
	t.Log(test.DecodeErrors)
	t.Log(test.ParseErrors)
	t.Log(test.Deprecated)
}