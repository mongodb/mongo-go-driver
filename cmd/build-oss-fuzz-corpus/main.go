// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Entry point for the MongoDB Go Driver integration into the Google "oss-fuzz" project
// (https://github.com/google/oss-fuzz).
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson"
)

const dataDir = "testdata/bson-corpus/"

type validityTestCase struct {
	Description       string  `json:"description"`
	CanonicalExtJSON  string  `json:"canonical_extjson"`
	RelaxedExtJSON    *string `json:"relaxed_extjson"`
	DegenerateExtJSON *string `json:"degenerate_extjson"`
	ConvertedExtJSON  *string `json:"converted_extjson"`
}

func findJSONFilesInDir(dir string) ([]string, error) {
	files := make([]string, 0)

	entries, err := ioutil.ReadDir(dir)
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

	// Use a SHA256 hash of the BSON bytes for the filename. This isn't an oss-fuzz requirement, it
	// just simplifies file naming.
	zipFile := fmt.Sprintf("%x", sha256.Sum256(jbytes))

	f, err := zw.Create(zipFile)
	if err != nil {
		log.Fatalf("error creating zip file: %v", err)
	}

	_, err = f.Write(jbytes)
	if err != nil {
		log.Fatalf("failed to write file: %s into zip file: %v", zipFile, err)
	}
}

// seedTestCase will add the byte representation for each "extJSON" string of each valid test case to the fuzzer's
// corpus.
func seedTestCase(zw *zip.Writer, tcase []*validityTestCase) {
	for _, vtc := range tcase {
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
	fileNames, err := findJSONFilesInDir(dataDir)
	if err != nil {
		log.Fatalf("failed to find JSON files in directory %q: %v", dataDir, err)
	}

	for _, fileName := range fileNames {
		filePath := path.Join(dataDir, fileName)

		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("failed to open file %q: %v", filePath, err)
		}

		tc := struct {
			Valid []*validityTestCase `json:"valid"`
		}{}

		if err := json.NewDecoder(file).Decode(&tc); err != nil {
			log.Fatalf("failed to decode file %q: %v", filePath, err)
		}

		seedTestCase(zw, tc.Valid)
	}
}

// main packs the local corpus as <fuzzer_name>_seed_corpus.zip, which is used by OSS-Fuzz to seed remote fuzzing
// of the MongoDB Go Driver. See here for more details: https://google.github.io/oss-fuzz/architecture/
func main() {
	seedCorpus := os.Args[1]
	if filepath.Ext(seedCorpus) != ".zip" {
		log.Fatalf("expected zip file <corpus>.zip, got %s", seedCorpus)
	}

	zipFile, err := os.Create(seedCorpus)
	if err != nil {
		log.Fatalf("failed creating zip file: %v", err)
	}

	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Fatalf("failed to close zip file: %v", err)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	seedBSONCorpus(zipWriter)

	if err := zipWriter.Close(); err != nil {
		log.Fatalf("failed to close zip writer: %v", err)
	}
}
