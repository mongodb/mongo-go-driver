// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// Pool of buffers for marshalling BSON.
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Pool of bson.Encoder.
var encPool = sync.Pool{
	New: func() interface{} {
		return new(bson.Encoder)
	},
}

func marshalValueWithRegistry(r *bson.Registry, val interface{}) (bsoncore.Value, error) {
	sw := bufPool.Get().(*bytes.Buffer)
	defer func() {
		// Proper usage of a sync.Pool requires each entry to have approximately
		// the same memory cost. To obtain this property when the stored type
		// contains a variably-sized buffer, we add a hard limit on the maximum
		// buffer to place back in the pool. We limit the size to 16MiB because
		// that's the maximum wire message size supported by any current MongoDB
		// server.
		//
		// Comment based on
		// https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/fmt/print.go;l=147
		//
		// Recycle byte slices that are smaller than 16MiB and at least half
		// occupied.
		if sw.Cap() < 16*1024*1024 && sw.Cap()/2 < sw.Len() {
			bufPool.Put(sw)
		}
	}()
	sw.Reset()
	vwFlusher := bson.NewValueWriter(sw).(interface {
		// The returned instance should be a *bson.valueWriter.
		WriteDocumentElement(string) (bson.ValueWriter, error)
		Flush() error
	})
	vw, err := vwFlusher.WriteDocumentElement("")
	if err != nil {
		return bsoncore.Value{}, err
	}

	enc := encPool.Get().(*bson.Encoder)
	defer encPool.Put(enc)
	enc.Reset(vw)
	enc.SetRegistry(r)
	if err := enc.Encode(val); err != nil {
		return bsoncore.Value{}, err
	}

	// flush the bytes written because we cannot guarantee that a full document has been written
	// after the flush, *sw will be in the format
	// [value type, 0 (null byte to indicate end of empty element name), value bytes..]
	if err := vwFlusher.Flush(); err != nil {
		return bsoncore.Value{}, err
	}
	typ := sw.Next(2)
	return bsoncore.Value{Type: bsoncore.Type(typ[0]), Data: sw.Bytes()}, nil
}
