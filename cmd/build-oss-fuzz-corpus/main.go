// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
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

const dataDir = "testdata/bson-corpus/"

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

// jsonToNative decodes the extended JSON string (ej) into a native Document
func jsonToNative(ej, ejType, testDesc string) (bson.D, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(ej), ejType != "relaxed", &doc); err != nil {
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

	b, err := bson.Marshal(native)
	if err != nil {
		return nil, fmt.Errorf("%s: encoding %s BSON: %w", testDesc, ejType, err)
	}

	return b, nil
}

// seedExtJSON will add the byte representation of the "extJSON" string to the fuzzer's coprus.
func seedExtJSON(zw *zip.Writer, extJSON string, extJSONType string, desc string) {
	jbytes, err := jsonToBytes(extJSON, extJSONType, desc)
	if err != nil {
		log.Fatalf("failed to convert JSON to bytes: %v", err)
	}

	hash := sha1.New()
	hash.Write(jbytes)
	file_in_zip := hex.EncodeToString(hash.Sum(nil))

	f, err := zw.Create(file_in_zip)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.Write(jbytes)
	if err != nil {
		log.Fatalf("Failed to write file: %s into zip file", file_in_zip)
	}
}

// seedTestCase will add the byte representation for each "extJSON" string of each valid test case to the fuzzer's
// corpus.
func seedTestCase(zw *zip.Writer, tcase *testCase) {
	for _, vtc := range tcase.Valid {
		seedExtJSON(zw, vtc.CanonicalExtJSON, "canonical", vtc.Description)

		// Seed the relaxed extended JSON.
		if vtc.RelaxedExtJSON != nil {
			seedExtJSON(zw, *vtc.RelaxedExtJSON, "relaxed", vtc.Description)
		}

		// Seed the degenerate extended JSON.
		if vtc.DegenerateExtJSON != nil {
			seedExtJSON(zw, *vtc.DegenerateExtJSON, "degenerate", vtc.Description)
		}

		// Seed the converted extended JSON.
		if vtc.ConvertedExtJSON != nil {
			seedExtJSON(zw, *vtc.ConvertedExtJSON, "converted", vtc.Description)
		}
	}
}

// seedBSONCorpus will unmarshal the data from "testdata/bson-corpus" into a slice of "testCase" structs and then
// marshal the "*_extjson" field of each "validityTestCase" into a slice of bytes to seed the fuzz corpus.
func seedBSONCorpus(zw *zip.Writer) {
	fileNames, err := FindJSONFilesInDir(dataDir)
	if err != nil {
		log.Fatalf("failed to find JSON files in directory %q: %v", dataDir, err)
	}

	for _, fileName := range fileNames {
		filePath := path.Join(dataDir, fileName)

		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("failed to open file %q: %v", filePath, err)
		}

		var tcase testCase
		if err := json.NewDecoder(file).Decode(&tcase); err != nil {
			log.Fatal(err)
		}

		seedTestCase(zw, &tcase)
	}
}

// This cmd generates and adds slice of bytes to files with sha1 hash name, and zip them as a <fuzzer_name>_seed_corpus.zip,
// which is used later by oss-fuzz as seed corpus, This is done because as of now oss-fuzz does not support go native t.Add()
// method.
func main() {
	seed_corpus := os.Args[1]
	if !strings.HasSuffix(seed_corpus, ".zip") {
		log.Fatalln("Expected command line:", os.Args[0], "<seed_corpus>.zip")
	}

	zip_file, err := os.Create(seed_corpus)
	if err != nil {
		log.Fatalf("Failed creating file: %s", err)
	}

	defer func() {
		err := zip_file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	zip_writer := zip.NewWriter(zip_file)
	seedBSONCorpus(zip_writer)

	if err := zip_writer.Close(); err != nil {
		log.Fatal(err)
	}
}
