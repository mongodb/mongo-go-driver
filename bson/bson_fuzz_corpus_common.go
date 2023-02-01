package bson

import (
	"fmt"
	"os"
	"path"
)

type TestCase struct {
	Description  string                `json:"description"`
	BsonType     string                `json:"bson_type"`
	TestKey      *string               `json:"test_key"`
	Valid        []ValidityTestCase    `json:"valid"`
	DecodeErrors []DecodeErrorTestCase `json:"decodeErrors"`
	ParseErrors  []ParseErrorTestCase  `json:"parseErrors"`
	Deprecated   *bool                 `json:"deprecated"`
}

type ValidityTestCase struct {
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

type DecodeErrorTestCase struct {
	Description string `json:"description"`
	Bson        string `json:"bson"`
}

type ParseErrorTestCase struct {
	Description string `json:"description"`
	String      string `json:"string"`
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
func JsonToBytes(ej, ejType, testDesc string) ([]byte, error) {
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

func FindJSONFilesInDir(dir string) ([]string, error) {
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
