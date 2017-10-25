package bson_test

import (
	"testing"
	"path"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"encoding/json"
	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/bson/extjson"
	"reflect"
	"encoding/hex"
	"github.com/10gen/stitch/utils/xjson"
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


		if (entry.Name() != "int32.json") {
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


		//printTestCaseData(t, test)


		t.Log("TestDescription", test.Description)
		for _, validCase := range test.Valid {
			t.Log("Description", validCase.Description)
			t.Log("cB", validCase.Canonical_Bson)
			t.Log("cEJ", validCase.Canonical_Extjson)

			cEJ := validCase.Canonical_Extjson
			cB := validCase.Canonical_Bson
			//validateCEJ(t, cEJ)
			//validateCEJ2(t, cEJ, validCase.Canonical_Bson)

			validateCanonicalBSON(t, cB, cEJ)
		}
		t.Log("\n\n")

	})
}


type testItemTypes struct {
	obj  interface{}
	data string
}


// --------------------------------------------------------------------------
// Samples from bsonspec.org:

var items = []testItemTypes{
	{bson.M{"hello": "world"},
		"\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"},
	//{bson.M{"BSON": []interface{}{"awesome", float64(5.05), 1986}},
	//	"1\x00\x00\x00\x04BSON\x00&\x00\x00\x00\x020\x00\x08\x00\x00\x00" +
	//		"awesome\x00\x011\x00333333\x14@\x102\x00\xc2\x07\x00\x00\x00\x00"},
}

//for cB input:
//native_to_bson( bson_to_native(cB) ) = cB
//native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
//native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
func validateCanonicalBSON(t *testing.T, cB string, cEJ string) {
	t.Log("\n\n")

	// Convert BSON to native then back to bson
	decoded, err := hex.DecodeString(cB)
	t.Log("decoded", decoded)
	nativeValue := bson.M{}
	error := bson.Unmarshal([]byte(decoded), nativeValue)
	t.Log("cb", cB)
	t.Log("cEJ", cEJ)
	t.Log("[]byte(cB)", []byte(cB))
	t.Log("error", error)
	t.Log("nativeValue", nativeValue)
	asdf, err := bson.Marshal(nativeValue);
	//encode := hex.EncodeToString(asdf);
	encode2 := hex.EncodeToString(asdf);
	t.Log("encode2", encode2)
	t.Log("asdf", asdf)
	t.Log("asdf", string(asdf))
	t.Log("err", err)


	// Reference xjson_test.go:testencodeextended
	// Now convert native to extended JSON
	var doc bson.D
	err = bson.Unmarshal([]byte(decoded), &doc)
	t.Log("doc", doc)
	t.Log("err", err)

	extended, err := extjson.EncodeBSONDtoJSON(doc)
	t.Log("extended", extended)
	t.Log("extended", string(extended)) // THis is what I want!
	t.Log("err", err)

	//
	//md, err := json.Marshal(xjson.NewValueOf(nativeValue))
	//t.Log("md", string(md))
	//t.Log("err", err)
	//
	//
	//d := bson.D{{"1", 1}, {"2", 2}, {"3", 3}}
	//md2, err := json.Marshal(xjson.NewValueOf(d))
	//md3, err := extjson.EncodeBSONDtoJSON(d)
	//t.Log("md2", string(md2))
	//t.Log("md2", string(md3))
	//t.Log("err", err)
}




























//for cEJ input:
//	native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
func validateCEJ2(t *testing.T, cEJ string, cB string) {
	nativeJSON, _ := extjson.DecodeExtended(cEJ)
	t.Log("typeof", reflect.TypeOf(nativeJSON));
	t.Log("nativeJSON", nativeJSON)

	bsonData, error := bson.Marshal(nativeJSON.(bson.D))

	t.Log("BSONedDATA", bsonData)
	t.Log("cB", cB)
	t.Log("error", error)
}

//	native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
func validateCEJ(t *testing.T, cEJ string) {
	// Create BSON Document from cEJ
	doc := extjson.MarshalD{} // Need this to get the data

	//gc.So(json.Unmarshal([]byte(cEJ), &doc), gc.ShouldBeNil)
	json.Unmarshal([]byte(cEJ), &doc)
	bsonDoc := bson.D(doc)


	// Decoded eJSON
	decoded, _ := extjson.DecodeExtended(bsonDoc)
	//gc.So(err, gc.ShouldBeNil)

	r := extjson.MarshalD(decoded.(bson.D))
	rDocStr, _ := r.MarshalJSON()
	//gc.So(err, gc.ShouldBeNil)

	rDoc := extjson.MarshalD{}
	//gc.So(json.Unmarshal([]byte(rDocStr), &rDoc), gc.ShouldBeNil)
	json.Unmarshal([]byte(rDocStr), &rDoc)
	//return bson.D(rDoc), bsonDoc




	t.Log("doc", doc)
	t.Log("bsonDoc", bsonDoc)
	t.Log("decoded", decoded)
	t.Log("r", r)
	t.Log("rDocStr", rDocStr)
	t.Log("rDoc", rDoc)
	t.Log("bson.D(rDoc)", bson.D(rDoc))
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

/*


	t.Log("\n\n\n AssertBSOND")
	// AssertBSOND

	expectedDoc := extjson.MarshalD{}
	json.Unmarshal([]byte(cEJ), &expectedDoc)
	t.Log("[]byte(cEJ)", []byte(cEJ))
	t.Log("expectedDoc", expectedDoc)
	dec, err := xjson.DecodeExtended(expectedDoc)
	t.Log("dec", dec)
	t.Log("err", err)

	rDocStr, err := json.Marshal(dec)
	t.Log("rDocStr", rDocStr)
	t.Log("err", err)

	actualDoc := xjson.MarshalD{}
	json.Unmarshal([]byte(rDocStr), &actualDoc)
	t.Log("actualDoc", actualDoc)




	t.Log("\n\n\nAssertMarshallD ")
	//	ASSERT MarshalD


	unmarshalledRes := extjson.MarshalD{} // Need this to get the data
	json.Unmarshal([]byte(cEJ), &unmarshalledRes)
	bsonDoc := bson.D(doc)
	t.Log("unmarshalledRes", unmarshalledRes)
	t.Log("bsonDoc", bsonDoc)

	decodedBsonDoc, err := extjson.DecodeExtended(bsonDoc)
	t.Log("decodedBsonDoc", decodedBsonDoc)

	r := extjson.MarshalD(decodedBsonDoc.(bson.D))
	resDoc, err := r.MarshalJSON()
	t.Log("r", r)
	t.Log("resDoc", resDoc)

	rDoc := extjson.MarshalD{}
	json.Unmarshal([]byte(resDoc), &rDoc)

	t.Log("rDoc", rDoc)



 */