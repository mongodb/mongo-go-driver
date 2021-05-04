// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/internal/testutil/israce"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestGridFS(x *testing.T) {
	mt := mtest.New(x, noClientOpts)
	defer mt.Close()

	mt.Run("skipping download", func(mt *mtest.T) {
		data := []byte("abc.def.ghi")
		var chunkSize int32 = 4

		testcases := []struct {
			name string

			read              int
			skip              int64
			expectedSkipN     int64
			expectedSkipErr   error
			expectedRemaining int
		}{
			{
				"read 0, skip 0", 0, 0, 0, nil, 11,
			},
			{
				"read 0, skip to end of chunk", 0, 4, 4, nil, 7,
			},
			{
				"read 0, skip 1", 0, 1, 1, nil, 10,
			},
			{
				"read 1, skip to end of chunk", 1, 3, 3, nil, 7,
			},
			{
				"read all, skip beyond", 11, 1, 0, nil, 0,
			},
			{
				"skip all", 0, 11, 11, nil, 0,
			},
			{
				"read 1, skip to last chunk", 1, 8, 8, nil, 2,
			},
			{
				"read to last chunk, skip to end", 9, 2, 2, nil, 0,
			},
			{
				"read to last chunk, skip beyond", 9, 4, 2, nil, 0,
			},
		}

		for _, tc := range testcases {
			mt.Run(tc.name, func(mt *mtest.T) {
				bucket, err := gridfs.NewBucket(mt.DB, options.GridFSBucket().SetChunkSizeBytes(chunkSize))
				assert.Nil(mt, err, "NewBucket error: %v", err)

				ustream, err := bucket.OpenUploadStream("foo")
				assert.Nil(mt, err, "OpenUploadStream error: %v", err)

				id := ustream.FileID
				_, err = ustream.Write(data)
				assert.Nil(mt, err, "Write error: %v", err)
				err = ustream.Close()
				assert.Nil(mt, err, "Close error: %v", err)

				dstream, err := bucket.OpenDownloadStream(id)
				assert.Nil(mt, err, "OpenDownloadStream error")
				dst := make([]byte, tc.read)
				_, err = dstream.Read(dst)
				assert.Nil(mt, err, "Read error: %v", err)

				n, err := dstream.Skip(tc.skip)
				assert.Equal(mt, tc.expectedSkipErr, err, "expected error on Skip: %v, got %v", tc.expectedSkipErr, err)
				assert.Equal(mt, tc.expectedSkipN, n, "expected Skip to return: %v, got %v", tc.expectedSkipN, n)

				// Read the rest.
				dst = make([]byte, len(data))
				remaining, err := dstream.Read(dst)
				if err != nil {
					assert.Equal(mt, err, io.EOF, "unexpected Read error: %v", err)
				}
				assert.Equal(mt, tc.expectedRemaining, remaining, "expected remaining data to be: %v, got %v", tc.expectedRemaining, remaining)
			})
		}
	})

	mt.Run("index creation", func(mt *mtest.T) {
		// Unit tests showing that UploadFromStream creates indexes on the chunks and files collections.
		bucket, err := gridfs.NewBucket(mt.DB)
		assert.Nil(mt, err, "NewBucket error: %v", err)
		err = bucket.SetWriteDeadline(time.Now().Add(5 * time.Second))
		assert.Nil(mt, err, "SetWriteDeadline error: %v", err)

		byteData := []byte("Hello, world!")
		r := bytes.NewReader(byteData)

		_, err = bucket.UploadFromStream("filename", r)
		assert.Nil(mt, err, "UploadFromStream error: %v", err)

		findCtx, cancel := context.WithTimeout(mtest.Background, 5*time.Second)
		defer cancel()
		findIndex(findCtx, mt, mt.DB.Collection("fs.files"), false, "key", "filename")
		findIndex(findCtx, mt, mt.DB.Collection("fs.chunks"), true, "key", "files_id")
	})
	// should not create a new index if index is numerically the same
	mt.Run("equivalent indexes", func(mt *mtest.T) {
		tests := []struct {
			name        string
			filesIndex  bson.D
			chunksIndex bson.D
			newIndexes  bool
		}{
			{
				"numerically equal",
				bson.D{
					{"key", bson.D{{"filename", float64(1.0)}, {"uploadDate", float64(1.0)}}},
					{"name", "filename_1_uploadDate_1"},
				},
				bson.D{
					{"key", bson.D{{"files_id", float64(1.0)}, {"n", float64(1.0)}}},
					{"name", "files_id_1_n_1"},
					{"unique", true},
				},
				false,
			},
			{
				"numerically inequal",
				bson.D{
					{"key", bson.D{{"filename", float64(-1.0)}, {"uploadDate", float64(1.0)}}},
					{"name", "filename_-1_uploadDate_1"},
				},
				bson.D{
					{"key", bson.D{{"files_id", float64(1.0)}, {"n", float64(-1.0)}}},
					{"name", "files_id_1_n_-1"},
					{"unique", true},
				},
				true,
			},
		}
		for _, test := range tests {
			mt.Run(test.name, func(mt *mtest.T) {
				mt.Run("OpenUploadStream", func(mt *mtest.T) {
					// add indexes with floats to collections manually
					res := mt.DB.RunCommand(context.Background(),
						bson.D{
							{"createIndexes", "fs.files"},
							{"indexes", bson.A{
								test.filesIndex,
							}},
						},
					)
					assert.Nil(mt, res.Err(), "createIndexes error: %v", res.Err())

					res = mt.DB.RunCommand(context.Background(),
						bson.D{
							{"createIndexes", "fs.chunks"},
							{"indexes", bson.A{
								test.chunksIndex,
							}},
						},
					)
					assert.Nil(mt, res.Err(), "createIndexes error: %v", res.Err())

					mt.ClearEvents()

					bucket, err := gridfs.NewBucket(mt.DB)
					assert.Nil(mt, err, "NewBucket error: %v", err)
					defer func() {
						_ = bucket.Drop()
					}()

					_, err = bucket.OpenUploadStream("filename")
					assert.Nil(mt, err, "OpenUploadStream error: %v", err)

					mt.FilterStartedEvents(func(evt *event.CommandStartedEvent) bool {
						return evt.CommandName == "createIndexes"
					})
					evt := mt.GetStartedEvent()
					if test.newIndexes {
						if evt == nil {
							mt.Fatalf("expected createIndexes events but got none")
						}
					} else {
						if evt != nil {
							mt.Fatalf("expected no createIndexes events but got %v", evt.Command)
						}
					}
				})
				mt.Run("UploadFromStream", func(mt *mtest.T) {
					// add indexes with floats to collections manually
					res := mt.DB.RunCommand(context.Background(),
						bson.D{
							{"createIndexes", "fs.files"},
							{"indexes", bson.A{
								test.filesIndex,
							}},
						},
					)
					assert.Nil(mt, res.Err(), "createIndexes error: %v", res.Err())

					res = mt.DB.RunCommand(context.Background(),
						bson.D{
							{"createIndexes", "fs.chunks"},
							{"indexes", bson.A{
								test.chunksIndex,
							}},
						},
					)
					assert.Nil(mt, res.Err(), "createIndexes error: %v", res.Err())

					mt.ClearEvents()
					var fileContent []byte
					bucket, err := gridfs.NewBucket(mt.DB)
					assert.Nil(mt, err, "NewBucket error: %v", err)
					defer func() {
						_ = bucket.Drop()
					}()

					_, err = bucket.UploadFromStream("filename", bytes.NewBuffer(fileContent))
					assert.Nil(mt, err, "UploadFromStream error: %v", err)

					mt.FilterStartedEvents(func(evt *event.CommandStartedEvent) bool {
						return evt.CommandName == "createIndexes"
					})
					evt := mt.GetStartedEvent()
					if test.newIndexes {
						if evt == nil {
							mt.Fatalf("expected createIndexes events but got none")
						}
					} else {
						if evt != nil {
							mt.Fatalf("expected no createIndexes events but got %v", evt.Command)
						}
					}
				})
			})
		}
	})

	mt.RunOpts("download", noClientOpts, func(mt *mtest.T) {
		mt.RunOpts("get file data", noClientOpts, func(mt *mtest.T) {
			// Tests for the DownloadStream.GetFile method.

			fileName := "get-file-data-test"
			fileData := []byte{1, 2, 3, 4}
			fileMetadata := bson.D{{"k1", "v1"}, {"k2", "v2"}}
			rawMetadata, err := bson.Marshal(fileMetadata)
			assert.Nil(mt, err, "Marshal error: %v", err)
			uploadOpts := options.GridFSUpload().SetMetadata(fileMetadata)

			testCases := []struct {
				name   string
				fileID interface{}
			}{
				{"default ID", nil},
				{"custom ID type", "customID"},
			}
			for _, tc := range testCases {
				mt.Run(tc.name, func(mt *mtest.T) {
					// Create a new GridFS bucket.
					bucket, err := gridfs.NewBucket(mt.DB)
					assert.Nil(mt, err, "NewBucket error: %v", err)
					defer func() { _ = bucket.Drop() }()

					// Upload the file and store the uploaded file ID.
					uploadedFileID := tc.fileID
					dataReader := bytes.NewReader(fileData)
					if uploadedFileID == nil {
						uploadedFileID, err = bucket.UploadFromStream(fileName, dataReader, uploadOpts)
					} else {
						err = bucket.UploadFromStreamWithID(tc.fileID, fileName, dataReader, uploadOpts)
					}
					assert.Nil(mt, err, "error uploading file: %v", err)

					// The uploadDate field is calculated when the upload is complete. Manually fetch it from the
					// fs.files collection to use in assertions.
					filesColl := mt.DB.Collection("fs.files")
					uploadedFileDoc, err := filesColl.FindOne(mtest.Background, bson.D{}).DecodeBytes()
					assert.Nil(mt, err, "FindOne error: %v", err)
					uploadTime := uploadedFileDoc.Lookup("uploadDate").Time().UTC()

					expectedFile := &gridfs.File{
						ID:         uploadedFileID,
						Length:     int64(len(fileData)),
						ChunkSize:  gridfs.DefaultChunkSize,
						UploadDate: uploadTime,
						Name:       fileName,
						Metadata:   rawMetadata,
					}
					// For both methods that create a DownloadStream, open a stream and compare the file given by the
					// stream to the expected File object.
					mt.RunOpts("OpenDownloadStream", noClientOpts, func(mt *mtest.T) {
						downloadStream, err := bucket.OpenDownloadStream(uploadedFileID)
						assert.Nil(mt, err, "OpenDownloadStream error: %v", err)
						actualFile := downloadStream.GetFile()
						assert.Equal(mt, expectedFile, actualFile, "expected file %v, got %v", expectedFile, actualFile)
					})
					mt.RunOpts("OpenDownloadStreamByName", noClientOpts, func(mt *mtest.T) {
						downloadStream, err := bucket.OpenDownloadStreamByName(fileName)
						assert.Nil(mt, err, "OpenDownloadStream error: %v", err)
						actualFile := downloadStream.GetFile()
						assert.Equal(mt, expectedFile, actualFile, "expected file %v, got %v", expectedFile, actualFile)
					})
				})
			}
		})
		mt.Run("chunk size determined by files collection document", func(mt *mtest.T) {
			// Test that the chunk size for a file download is determined by the chunkSize field in the files
			// collection document, not the bucket's chunk size.

			bucket, err := gridfs.NewBucket(mt.DB)
			assert.Nil(mt, err, "NewBucket error: %v", err)
			defer func() { _ = bucket.Drop() }()

			fileData := []byte("hello world")
			uploadOpts := options.GridFSUpload().SetChunkSizeBytes(4)
			fileID, err := bucket.UploadFromStream("file", bytes.NewReader(fileData), uploadOpts)
			assert.Nil(mt, err, "UploadFromStream error: %v", err)

			// If the bucket's chunk size was used, this would error because the actual chunk size is 4 and the bucket
			// chunk size is 255 KB.
			var downloadBuffer bytes.Buffer
			_, err = bucket.DownloadToStream(fileID, &downloadBuffer)
			assert.Nil(mt, err, "DownloadToStream error: %v", err)

			downloadedBytes := downloadBuffer.Bytes()
			assert.Equal(mt, fileData, downloadedBytes, "expected bytes %s, got %s", fileData, downloadedBytes)
		})
		mt.Run("error if files collection document does not have a chunkSize field", func(mt *mtest.T) {
			// Test that opening a download returns ErrMissingChunkSize if the files collection document has no
			// chunk size field.

			oid := primitive.NewObjectID()
			filesDoc := bson.D{
				{"_id", oid},
				{"length", 10},
				{"filename", "filename"},
			}
			_, err := mt.DB.Collection("fs.files").InsertOne(mtest.Background, filesDoc)
			assert.Nil(mt, err, "InsertOne error for files collection: %v", err)

			bucket, err := gridfs.NewBucket(mt.DB)
			assert.Nil(mt, err, "NewBucket error: %v", err)
			defer func() { _ = bucket.Drop() }()

			_, err = bucket.OpenDownloadStream(oid)
			assert.Equal(mt, gridfs.ErrMissingChunkSize, err, "expected error %v, got %v", gridfs.ErrMissingChunkSize, err)
		})
		mt.Run("cursor error during read after downloading", func(mt *mtest.T) {
			// To simulate a cursor error we upload a file larger than the 16MB default batch size,
			// so the underlying cursor remains open on the server. Since the ReadDeadline is
			// set in the past, Read should cause a timeout.

			fileName := "read-error-test"
			fileData := make([]byte, 17000000)

			bucket, err := gridfs.NewBucket(mt.DB)
			assert.Nil(mt, err, "NewBucket error: %v", err)
			defer func() { _ = bucket.Drop() }()

			dataReader := bytes.NewReader(fileData)
			_, err = bucket.UploadFromStream(fileName, dataReader)
			assert.Nil(mt, err, "UploadFromStream error: %v", err)

			ds, err := bucket.OpenDownloadStreamByName(fileName)
			assert.Nil(mt, err, "OpenDownloadStreamByName error: %v", err)

			err = ds.SetReadDeadline(time.Now().Add(-1 * time.Second))
			assert.Nil(mt, err, "SetReadDeadline error: %v", err)

			p := make([]byte, len(fileData))
			_, err = ds.Read(p)
			assert.NotNil(mt, err, "expected error from Read, got nil")
			assert.True(mt, strings.Contains(err.Error(), "context deadline exceeded"),
				"expected error to contain 'context deadline exceeded', got %v", err.Error())
		})
		mt.Run("cursor error during skip after downloading", func(mt *mtest.T) {
			// To simulate a cursor error we upload a file larger than the 16MB default batch size,
			// so the underlying cursor remains open on the server. Since the ReadDeadline is
			// set in the past, Skip should cause a timeout.

			fileName := "skip-error-test"
			fileData := make([]byte, 17000000)

			bucket, err := gridfs.NewBucket(mt.DB)
			assert.Nil(mt, err, "NewBucket error: %v", err)
			defer func() { _ = bucket.Drop() }()

			dataReader := bytes.NewReader(fileData)
			_, err = bucket.UploadFromStream(fileName, dataReader)
			assert.Nil(mt, err, "UploadFromStream error: %v", err)

			ds, err := bucket.OpenDownloadStreamByName(fileName)
			assert.Nil(mt, err, "OpenDownloadStreamByName error: %v", err)

			err = ds.SetReadDeadline(time.Now().Add(-1 * time.Second))
			assert.Nil(mt, err, "SetReadDeadline error: %v", err)

			_, err = ds.Skip(int64(len(fileData)))
			assert.NotNil(mt, err, "expected error from Skip, got nil")
			assert.True(mt, strings.Contains(err.Error(), "context deadline exceeded"),
				"expected error to contain 'context deadline exceeded', got %v", err.Error())
		})
	})

	mt.RunOpts("bucket collection accessors", noClientOpts, func(mt *mtest.T) {
		// Tests for the GetFilesCollection and GetChunksCollection accessors.

		fileData := []byte{1, 2, 3, 4}
		var chunkSize int32 = 2

		testCases := []struct {
			name       string
			bucketName string // defaults to "fs"
		}{
			{"default bucket name", ""},
			{"custom bucket name", "bucket"},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				bucketOpts := options.GridFSBucket().SetChunkSizeBytes(chunkSize)
				if tc.bucketName != "" {
					bucketOpts.SetName(tc.bucketName)
				}
				bucket, err := gridfs.NewBucket(mt.DB, bucketOpts)
				assert.Nil(mt, err, "NewBucket error: %v", err)
				defer func() { _ = bucket.Drop() }()

				_, err = bucket.UploadFromStream("accessors-test-file", bytes.NewReader(fileData))
				assert.Nil(mt, err, "UploadFromStream error: %v", err)

				bucketName := tc.bucketName
				if bucketName == "" {
					bucketName = "fs"
				}
				assertGridFSCollectionState(mt, bucket.GetFilesCollection(), bucketName+".files", 1)
				assertGridFSCollectionState(mt, bucket.GetChunksCollection(), bucketName+".chunks", 2)
			})
		}
	})

	mt.RunOpts("round trip", mtest.NewOptions().MaxServerVersion("3.6"), func(mt *mtest.T) {
		skipRoundTripTest(mt)
		oneK := 1024
		smallBuffSize := 100

		tests := []struct {
			name      string
			chunkSize int // make -1 for no capacity for no chunkSize
			fileSize  int
			bufSize   int // make -1 for no capacity for no bufSize
		}{
			{"RoundTrip: original", -1, oneK, -1},
			{"RoundTrip: chunk size multiple of file", oneK, oneK * 16, -1},
			{"RoundTrip: chunk size is file size", oneK, oneK, -1},
			{"RoundTrip: chunk size multiple of file size and with strict buffer size", oneK, oneK * 16, smallBuffSize},
			{"RoundTrip: chunk size multiple of file size and buffer size", oneK, oneK * 16, oneK * 16},
			{"RoundTrip: chunk size, file size, buffer size all the same", oneK, oneK, oneK},
		}

		for _, test := range tests {
			mt.Run(test.name, func(mt *mtest.T) {
				var chunkSize *int32
				var temp int32
				if test.chunkSize != -1 {
					temp = int32(test.chunkSize)
					chunkSize = &temp
				}

				bucket, err := gridfs.NewBucket(mt.DB, &options.BucketOptions{
					ChunkSizeBytes: chunkSize,
				})
				assert.Nil(mt, err, "NewBucket error: %v", err)

				timeout := 5 * time.Second
				if israce.Enabled {
					timeout = 20 * time.Second // race detector causes 2-20x slowdown
				}

				err = bucket.SetWriteDeadline(time.Now().Add(timeout))
				assert.Nil(mt, err, "SetWriteDeadline error: %v", err)

				// Test that Upload works when the buffer to write is longer than the upload stream's internal buffer.
				// This requires multiple calls to uploadChunks.
				size := test.fileSize
				p := make([]byte, size)
				for i := 0; i < size; i++ {
					p[i] = byte(rand.Intn(100))
				}

				_, err = bucket.UploadFromStream("filename", bytes.NewReader(p))
				assert.Nil(mt, err, "UploadFromStream error: %v", err)

				var w *bytes.Buffer
				if test.bufSize == -1 {
					w = bytes.NewBuffer(make([]byte, 0))
				} else {
					w = bytes.NewBuffer(make([]byte, 0, test.bufSize))
				}

				_, err = bucket.DownloadToStreamByName("filename", w)
				assert.Nil(mt, err, "DownloadToStreamByName error: %v", err)
				assert.Equal(mt, p, w.Bytes(), "downloaded file did not match p")
			})
		}
	})
}

func assertGridFSCollectionState(mt *mtest.T, coll *mongo.Collection, expectedName string, expectedNumDocuments int64) {
	mt.Helper()

	assert.Equal(mt, expectedName, coll.Name(), "expected collection name %v, got %v", expectedName, coll.Name())
	count, err := coll.CountDocuments(mtest.Background, bson.D{})
	assert.Nil(mt, err, "CountDocuments error: %v", err)
	assert.Equal(mt, expectedNumDocuments, count, "expected %d documents in collection, got %d", expectedNumDocuments,
		count)
}

func findIndex(ctx context.Context, mt *mtest.T, coll *mongo.Collection, unique bool, keys ...string) {
	mt.Helper()
	cur, err := coll.Indexes().List(ctx)
	assert.Nil(mt, err, "Indexes List error: %v", err)

	foundIndex := false
	for cur.Next(ctx) {
		if _, err := cur.Current.LookupErr(keys...); err == nil {
			if uVal, err := cur.Current.LookupErr("unique"); (unique && err == nil && uVal.Boolean() == true) ||
				(!unique && (err != nil || uVal.Boolean() == false)) {

				foundIndex = true
			}
		}
	}
	assert.True(mt, foundIndex, "index %v not found", keys)
}

func skipRoundTripTest(mt *mtest.T) {
	if runtime.GOOS != "darwin" {
		return
	}

	var serverStatus bson.Raw
	err := mt.DB.RunCommand(
		context.Background(),
		bson.D{{"serverStatus", 1}},
	).Decode(&serverStatus)
	assert.Nil(mt, err, "serverStatus error %v", err)

	// can run on non-sharded clusters or on sharded cluster with auth/ssl disabled
	_, err = serverStatus.LookupErr("sharding")
	if err != nil {
		return
	}
	_, err = serverStatus.LookupErr("security")
	if err != nil {
		return
	}
	mt.Skip("skipping round trip test")
}
