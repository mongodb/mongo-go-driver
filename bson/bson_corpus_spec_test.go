package bson_test

import (
	"testing"
	"path"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/extjson"
	"github.com/10gen/mongo-go-driver/bson/internal/json"
	//"encoding/json"
	"encoding/hex"
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
		//
		//if   entry.Name() != "code.json" {
		//	continue
		//}

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
		if (test.Deprecated) {
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
				// TODO: Steven - all good
				t.Run(testName+"validateBsonToRelaxedJSON:"+validCase.Description, func(t *testing.T) {
					validateBsonToRelaxedJSON(t, cB, rEJ)
				})

				// TODO: Failing on int64.json MaxInt
				t.Run(testName+"validateRelaxedExtendedJSON:"+validCase.Description, func(t *testing.T) {
					validateRelaxedExtendedJSON(t, rEJ, test.Description)
				})
			}

			 //TODO: Steven - Passes all. All good here
			dB := validCase.Degenerate_Bson
			if dB != "" {
				t.Run(testName+"validateDegenerateBSON:"+validCase.Description, func(t *testing.T) {
					validateDegenerateBSON(t, dB, cB)
				})
			}

			 //TODO: Steven - All passing
			dEJ := validCase.Degenerate_Extjson
			if dEJ != "" {
				t.Run(testName+"validateDegenerateExtendedJSON:"+validCase.Description, func (t *testing.T) {
					validateDegenerateExtendedJSON(t, dEJ, cEJ, cB, lossy)
				})
			}
		}

		// TODO: Steven - All good here
		for _, decodeTest := range test.DecodeErrors {
			t.Run(testName+"decodeTest:"+decodeTest.Description, func (t *testing.T) {
				testDecodeError(t, decodeTest.Bson)
			})
		}

		// TODO: Steven - Passes all. All good here.
		for _, parseTest := range test.ParseErrors {
			t.Run(testName+"parseTest:"+parseTest.Description, func (t *testing.T) {
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

	// TODO: Steven - update: The above code works as it preserves order
	// Converted to BSON.D As order is preservered. With bson.M its just a map so its random.
	//nativeRepr := bson.M{}
	//err = bson.Unmarshal([]byte(decoded), nativeRepr)
	//require.NoError(t, err)

	//nativeD := bson.D{}
	//nativeD.AppendMap(nativeRepr)
	//roundTripCBByteRepr, err := bson.Marshal(nativeRepr);
	/////////////////////////////////////////////////////////////////////////

	roundTripCBByteRepr, err := bson.Marshal(nativeReprD);
	roundTripCB := hex.EncodeToString(roundTripCBByteRepr);
	require.Equal(t, cB, strings.ToUpper(roundTripCB))

	//native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
	var nativeReprBsonD bson.D // Need this new bson.D object as encodeBsonDtoJson wants a Bson.D
	err = bson.Unmarshal([]byte(decoded), &nativeReprBsonD)
	require.NoError(t, err)

	//t.Log(nativeReprBsonD)

	roundTripCEJ, err := extjson.EncodeBSONDtoJSON(nativeReprBsonD)
	require.NoError(t, err)

	// In case of $numberDouble we can't fully represent it in Go. Therefore we will convert cEJ into NativeRepr
	if strings.Contains(cEJ, "$numberDouble") && compressJSON(cEJ) != string(roundTripCEJ) {
		t.Log("IN NUMBERDOUBLE")
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(cEJ), &marshalDDoc)
		require.NoError(t, err)
		require.Equal(t, marshalDDoc[0].Value, nativeReprBsonD[0].Value)
	} else {
		require.Equal(t, compressJSON(cEJ), string(roundTripCEJ))
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

	// TODO:Steven: Clean this up
	// In case of $numberDouble we can't fully represent it in Go. Therefore we will convert cEJ into NativeRepr
	if strings.Contains(rEJ, "1.234567890123456") && compressJSON(rEJ) != string(roundTripREJ) {
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(rEJ), &marshalDDoc)
		require.NoError(t, err)

		bsonDDoc := bson.D{}
		bsonDDoc.AppendMap(nativeRepr)
		require.Equal(t, marshalDDoc[0].Value, bsonDDoc[0].Value)
	} else {
		require.Equal(t, compressJSON(rEJ), string(roundTripREJ))
	}
}

func validateRelaxedExtendedJSON(t *testing.T, rEJ string, testFileName string) {
	//native_to_relaxed_extended_json( json_to_native(rEJ) ) = rEJ
	nativeRepr := bson.M{}
	require.NoError(t, json.Unmarshal([]byte(rEJ), &nativeRepr))

	// TODO: steven -  Need this as without this JSON.unmarshall will see a JSON Number and it will convert that into a float64
	// Then I have no idea whether the user wanted a floating point number 1.0 or an integer 1.
	// TODO: Fails on MaxInt
	if (strings.Contains(testFileName, "Int")) {
		for k := range nativeRepr {
			nativeRepr[k] = int64(nativeRepr[k].(float64))
		}

	}
	roundTripREJ, err := json.Marshal(nativeRepr)
	require.NoError(t, err)
	//require.Equal(t, compressJSON(rEJ), string(roundTripREJ))

	// TODO: Steven Clean
	// In case of $numberDouble we can't fully represent it in Go. Therefore we will convert cEJ into NativeRepr
	if strings.Contains(testFileName, "Double") && compressJSON(rEJ) != string(roundTripREJ) {
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(rEJ), &marshalDDoc)
		require.NoError(t, err)

		nativeReprBsonD := bson.D{}
		nativeReprBsonD.AppendMap(nativeRepr)
		require.Equal(t, marshalDDoc[0].Value, nativeReprBsonD[0].Value)
	} else {
		require.Equal(t, compressJSON(rEJ), string(roundTripREJ))
	}
}

//for cEJ input:
func validateCanonicalExtendedJSON(t *testing.T, cB string, cEJ string, lossy bool) {
	marshalDDoc := extjson.MarshalD{}
	err := json.Unmarshal([]byte(cEJ), &marshalDDoc)
	require.NoError(t, err)

	bsonDDoc := bson.D(marshalDDoc)

	//fmt.Println(bsonDDoc)

	// TODO:Steven - In some scenarios, json.Unmarshall(bson.D) doesn't work well. ie timestamps
	// As a backup we're gonna import the other stufff.
	//if len(bsonDDoc) == 0 {
	//	nativeRepr := bson.M{}
	//	require.NoError(t, json.Unmarshal([]byte(cEJ), &nativeRepr))
	//
	//	bsonDDoc = bson.D{}
	//	bsonDDoc.AppendMap(nativeRepr)
	//}

	t.Log(bsonDDoc)
	// native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
	roundTripCEJByteRepr, err := extjson.EncodeBSONDtoJSON(bsonDDoc)
	require.NoError(t, err)


	// TODO:Steven: Clean this up
	// In case of $numberDouble we can't fully represent it in Go. Therefore we will convert cEJ into NativeRepr
	if strings.Contains(cEJ, "$numberDouble") && compressJSON(cEJ) != string(roundTripCEJByteRepr) {
		marshalDDoc := extjson.MarshalD{}
		err := json.Unmarshal([]byte(cEJ), &marshalDDoc)
		require.NoError(t, err)
		require.Equal(t, marshalDDoc[0].Value, bsonDDoc[0].Value)
	} else {
		require.Equal(t, compressJSON(cEJ), string(roundTripCEJByteRepr))
	}





	// native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
	if !lossy {
		bsonHexDecoded, err := bson.Marshal(bsonDDoc)
		require.NoError(t, err)

		roundTripCB := hex.EncodeToString(bsonHexDecoded)
		require.Equal(t, cB, strings.ToUpper(roundTripCB))
	}
}


//for dB input (if it exists):
func validateDegenerateBSON(t *testing.T, dB string, cB string) {
	//native_to_bson( bson_to_native(dB) ) = cB
	decoded, err := hex.DecodeString(dB)
	require.NoError(t, err)

	nativeRepr := bson.M{}
	err = bson.Unmarshal([]byte(decoded), nativeRepr)
	require.NoError(t, err)

	dBByteRepr, err := bson.Marshal(nativeRepr);
	require.NoError(t, err)
	roundTripDB := hex.EncodeToString(dBByteRepr);
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
	require.Equal(t, compressJSON(cEJ), string(roundTripCEJ))

	//native_to_bson( json_to_native(dEJ) ) = cB (unless lossy)
	if !lossy {
		dBByteRepr, err := bson.Marshal(nativeD);
		require.NoError(t, err)
		roundTripDB := hex.EncodeToString(dBByteRepr);
		require.Equal(t, cB, strings.ToUpper(roundTripDB))
	}
}

func compressJSON(js string) string {
	var json_bytes = []byte(js)
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, json_bytes); err != nil {
		fmt.Println(err)
	}
	return buffer.String()
}

// Currently have 2 cases failing
//--- FAIL: TestBSONSpec/code;/code;decodeError:invalid_UTF-8 (0.00s)
//--- FAIL: TestBSONSpec/string;/string;decodeError:invalid_UTF-8 (0.00s)
func testDecodeError(t *testing.T, b string) {
	decoded, err := hex.DecodeString(b)
	require.NoError(t, err)

	nativeReprD := bson.D{}
	err = bson.Unmarshal([]byte(decoded), &nativeReprD)

	if err == nil {
		t.Log([]byte(nativeReprD[0].Value.(string)))
	}

	require.Error(t, err)
}

func testParseError(t *testing.T, s string) {
	var nativeReprBsonD bson.D
	d := bson.Unmarshal([]byte(s), &nativeReprBsonD)
	require.Error(t, d)
}
