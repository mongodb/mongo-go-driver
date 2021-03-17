// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func executeBucketDelete(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var id *bson.RawValue
	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "id":
			id = &val
		default:
			return nil, fmt.Errorf("unrecognized bucket delete option %q", key)
		}
	}
	if id == nil {
		return nil, newMissingArgumentError("id")
	}

	return newErrorResult(bucket.Delete(*id)), nil
}

func executeBucketDownload(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var id *bson.RawValue
	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "id":
			id = &val
		default:
			return nil, fmt.Errorf("unrecognized bucket download option %q", key)
		}
	}
	if id == nil {
		return nil, newMissingArgumentError("id")
	}

	stream, err := bucket.OpenDownloadStream(*id)
	if err != nil {
		return newErrorResult(err), nil
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, stream); err != nil {
		return newErrorResult(err), nil
	}

	return newValueResult(bsontype.Binary, bsoncore.AppendBinary(nil, 0, buffer.Bytes()), nil), nil
}

func executeBucketUpload(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var filename string
	var fileBytes []byte
	opts := options.GridFSUpload()

	elems, _ := operation.Arguments.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "chunkSizeBytes":
			opts.SetChunkSizeBytes(val.Int32())
		case "filename":
			filename = val.StringValue()
		case "metadata":
			opts.SetMetadata(val.Document())
		case "source":
			fileBytes, err = hex.DecodeString(val.Document().Lookup("$$hexBytes").StringValue())
			if err != nil {
				return nil, fmt.Errorf("error converting source string to bytes: %v", err)
			}
		default:
			return nil, fmt.Errorf("unrecognized bucket upload option %q", key)
		}
	}
	if filename == "" {
		return nil, newMissingArgumentError("filename")
	}
	if fileBytes == nil {
		return nil, newMissingArgumentError("source")
	}

	fileID, err := bucket.UploadFromStream(filename, bytes.NewReader(fileBytes), opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	if operation.ResultEntityID != nil {
		fileIDValue := bson.RawValue{
			Type:  bsontype.ObjectID,
			Value: fileID[:],
		}
		if err := entities(ctx).addBSONEntity(*operation.ResultEntityID, fileIDValue); err != nil {
			return nil, fmt.Errorf("error storing result as BSON entity: %v", err)
		}
	}

	return newValueResult(bsontype.ObjectID, fileID[:], nil), nil
}
