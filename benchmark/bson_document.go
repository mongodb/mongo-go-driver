// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

func bsonDocumentEncoding(tm TimerManager, iters int, source string) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, source)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.Marshal(doc)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("marshaling error")
		}
	}

	return nil
}

func bsonDocumentDecodingLazy(tm TimerManager, iters int, source string) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, source)
	if err != nil {
		return err
	}

	raw, err := bson.Marshal(doc)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		var out bson.D
		err := bson.Unmarshal(raw, &out)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("marshaling error")
		}
	}
	return nil
}

func bsonDocumentDecoding(tm TimerManager, iters, numKeys int, source string) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, source)
	if err != nil {
		return err
	}

	raw, err := bson.Marshal(doc)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		var out bson.D
		err := bson.Unmarshal(raw, &out)
		if err != nil {
			return err
		}

		if len(out) != numKeys {
			return errors.New("document parsing error")
		}
	}
	return nil

}

func BSONFlatDocumentEncoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentEncoding(tm, iters, flatBSONData)
}

func BSONFlatDocumentDecodingLazy(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecodingLazy(tm, iters, flatBSONData)
}

func BSONFlatDocumentDecoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecoding(tm, iters, 145, flatBSONData)
}

func BSONDeepDocumentEncoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentEncoding(tm, iters, deepBSONData)
}

func BSONDeepDocumentDecodingLazy(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecodingLazy(tm, iters, deepBSONData)
}

func BSONDeepDocumentDecoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecoding(tm, iters, 126, deepBSONData)
}

func BSONFullDocumentEncoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentEncoding(tm, iters, fullBSONData)
}

func BSONFullDocumentDecodingLazy(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecodingLazy(tm, iters, fullBSONData)
}

func BSONFullDocumentDecoding(_ context.Context, tm TimerManager, iters int) error {
	return bsonDocumentDecoding(tm, iters, 145, fullBSONData)
}
