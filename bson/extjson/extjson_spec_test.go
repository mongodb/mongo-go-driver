package extjson

import (
	"testing"
	"path"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"encoding/json"
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

const testsDir string = "../../specifications/source/bson-corpus/tests/"

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


		t.Log("\n\n")
		t.Log("TestDescription", test.Description)
		for _, validCase := range test.Valid {
			t.Log("Description", validCase.Description)
			t.Log("cB", validCase.Canonical_Bson)
			t.Log("cEJ", validCase.Canonical_Extjson)

			//cEJ := validCase.Canonical_Extjson
			cB := validCase.Canonical_Bson
			validateCB(t, cB)


		}
	})
}

//for cB input:
//	native_to_bson( bson_to_native(cB) ) = cB
//	native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
//	native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
func validateCB(t *testing.T, cB string) {



}



//func validateCEJ(t *testing.T, cEJ string) {
//	doc := MarshalD{} // Need this to get the data
//	//gc.So(json.Unmarshal([]byte(cEJ), &doc), gc.ShouldBeNil)
//	json.Unmarshal([]byte(cEJ), &doc)
//	bsonDoc := bson.D(doc)
//
//	dec, _ := DecodeExtended(bsonDoc)
//	//gc.So(err, gc.ShouldBeNil)
//
//	r := MarshalD(dec.(bson.D))
//	rDocStr, _ := r.MarshalJSON()
//	//gc.So(err, gc.ShouldBeNil)
//
//	rDoc := MarshalD{}
//	//gc.So(json.Unmarshal([]byte(rDocStr), &rDoc), gc.ShouldBeNil)
//	json.Unmarshal([]byte(rDocStr), &rDoc)
//	//return bson.D(rDoc), bsonDoc
//
//
//	t.Log("doc", doc)
//	t.Log("bsonDoc", bsonDoc)
//	t.Log("dec", dec)
//	t.Log("r", r)
//	t.Log("rDocStr", rDocStr)
//	t.Log("rDoc", rDoc)
//	t.Log("bson.D(rDoc)", bson.D(rDoc))
//}

//func assertBsonD(jsonString string) (bson.D, bson.D) {
//	doc := xjson.MarshalD{} // Need this to get the data
//	gc.So(json.Unmarshal([]byte(jsonString), &doc), gc.ShouldBeNil)
//	bsonDoc := bson.D(doc)
//
//	dec, err := xjson.DecodeExtended(bsonDoc)
//	gc.So(err, gc.ShouldBeNil)
//
//	r := xjson.MarshalD(dec.(bson.D))
//	rDocStr, err := r.MarshalJSON()
//	gc.So(err, gc.ShouldBeNil)
//
//	rDoc := xjson.MarshalD{}
//	gc.So(json.Unmarshal([]byte(rDocStr), &rDoc), gc.ShouldBeNil)
//	return bson.D(rDoc), bsonDoc
//}


//for cB input:
//	native_to_bson( bson_to_native(cB) ) = cB
//	native_to_canonical_extended_json( bson_to_native(cB) ) = cEJ
//	native_to_relaxed_extended_json( bson_to_native(cB) ) = rEJ (if rEJ exists)
//for cEJ input:
//	native_to_canonical_extended_json( json_to_native(cEJ) ) = cEJ
//	native_to_bson( json_to_native(cEJ) ) = cB (unless lossy)
//for dB input (if it exists):
//	native_to_bson( bson_to_native(dB) ) = cB
//for dEJ input (if it exists):
//	native_to_canonical_extended_json( json_to_native(dEJ) ) = cEJ
//	native_to_bson( json_to_native(dEJ) ) = cB (unless lossy)
//for rEJ input (if it exists):
//	native_to_relaxed_extended_json( json_to_native(rEJ) ) = rEJ

func printTestCaseData(t *testing.T, test testCase) {
	t.Log(test.Description)
	t.Log(test.Bson_Type)
	t.Log(test.Test_Key)
	t.Log(test.Valid)
	t.Log(test.DecodeErrors)
	t.Log(test.ParseErrors)
	t.Log(test.Deprecated)
}
