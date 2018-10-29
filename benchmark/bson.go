// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"errors"
	"io/ioutil"
	"path/filepath"

	"github.com/mongodb/mongo-go-driver/bson"
)

const (
	perfDataDir  = "perf"
	bsonDataDir  = "extended_bson"
	flatBSONData = "flat_bson.json"
	deepBSONData = "deep_bson.json"
	fullBSONData = "full_bson.json"
)

// utility functions for the bson benchmarks

func loadSourceDocument(pathParts ...string) (*bson.Document, error) {
	data, err := ioutil.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		return nil, err
	}
	doc := bson.NewDocument()
	err = bson.UnmarshalExtJSON(data, true, &doc)
	if err != nil {
		return nil, err
	}

	if doc.Len() == 0 {
		return nil, errors.New("empty bson document")
	}

	return doc, nil
}

func loadSourceReader(pathParts ...string) (bson.Reader, error) {
	doc, err := loadSourceDocument(pathParts...)
	if err != nil {
		return nil, err
	}
	raw, err := doc.MarshalBSON()
	if err != nil {
		return nil, err
	}

	return bson.Reader(raw), nil
}
