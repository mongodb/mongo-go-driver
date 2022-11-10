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
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func createBucketFindCursor(ctx context.Context, operation *operation) (*cursorResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var filter bson.Raw
	opts := options.GridFSFind()

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "maxTimeMS":
			opts.SetMaxTime(time.Duration(val.Int32()) * time.Millisecond)
		case "filter":
			filter = val.Document()
		default:
			return nil, fmt.Errorf("unrecognized bucket find option %q", key)
		}
	}
	if filter == nil {
		return nil, newMissingArgumentError("filter")
	}

	cursor, err := bucket.FindContext(ctx, filter, opts)
	res := &cursorResult{
		cursor: cursor,
		err:    err,
	}
	return res, nil
}

func executeBucketDelete(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var id *bson.RawValue

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
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

	return newErrorResult(bucket.DeleteContext(ctx, *id)), nil
}

func executeBucketDownload(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var id *bson.RawValue
	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
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

func executeBucketDownloadByName(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}

	var filename string
	opts := options.GridFSName()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "filename":
			filename = val.StringValue()
		case "revision":
			opts.SetRevision(val.AsInt32())
		default:
			return nil, fmt.Errorf("unrecognized bucket download option %q", key)
		}
	}
	if filename == "" {
		return nil, newMissingArgumentError("filename")
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStreamByName(filename, &buf, opts)
	if err != nil {
		return newErrorResult(err), nil
	}

	return newValueResult(bsontype.Binary, bsoncore.AppendBinary(nil, 0, buf.Bytes()), nil), nil
}

func executeBucketDrop(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	return newErrorResult(bucket.DropContext(ctx)), nil
}

func executeBucketRename(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var id *bson.RawValue
	var newFilename string
	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "id":
			id = &val
		case "newFilename":
			newFilename = val.StringValue()
		default:
			return nil, fmt.Errorf("unrecognized bucket rename option %q", key)
		}
	}
	if id == nil {
		return nil, newMissingArgumentError("id")
	}

	return newErrorResult(bucket.RenameContext(ctx, id, newFilename)), nil
}

func executeBucketUpload(ctx context.Context, operation *operation) (*operationResult, error) {
	bucket, err := entities(ctx).gridFSBucket(operation.Object)
	if err != nil {
		return nil, err
	}

	var filename string
	var fileBytes []byte
	opts := options.GridFSUpload()

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
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
		case "contentType":
			return nil, newSkipTestError("the deprecated contentType file option is not supported")
		case "disableMD5":
			return nil, newSkipTestError("the deprecated disableMD5 file option is not supported")
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
