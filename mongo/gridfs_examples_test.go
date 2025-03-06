// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ExampleGridFSBucket_OpenUploadStream() {
	var fileContent []byte
	var bucket *mongo.GridFSBucket

	// Specify the Metadata option to include a "metadata" field in the files
	// collection document.
	uploadOpts := options.GridFSUpload().
		SetMetadata(bson.D{{"metadata tag", "tag"}})

	// Use WithContext to force a timeout if the upload does not succeed in
	// 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	uploadStream, err := bucket.OpenUploadStream(ctx, "filename", uploadOpts)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err = uploadStream.Close(); err != nil {
			log.Panic(err)
		}
	}()

	if _, err = uploadStream.Write(fileContent); err != nil {
		log.Panic(err)
	}
}

func ExampleGridFSBucket_UploadFromStream() {
	var fileContent []byte
	var bucket *mongo.GridFSBucket

	// Specify the Metadata option to include a "metadata" field in the files
	// collection document.
	uploadOpts := options.GridFSUpload().
		SetMetadata(bson.D{{"metadata tag", "tag"}})
	fileID, err := bucket.UploadFromStream(
		context.Background(),
		"filename",
		bytes.NewBuffer(fileContent),
		uploadOpts)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("new file created with ID %s", fileID)
}

func ExampleGridFSBucket_OpenDownloadStream() {
	var bucket *mongo.GridFSBucket
	var fileID bson.ObjectID

	// Use WithContext to force a timeout if the download does not succeed in
	// 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	downloadStream, err := bucket.OpenDownloadStream(ctx, fileID)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := downloadStream.Close(); err != nil {
			log.Panic(err)
		}
	}()

	fileBuffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(fileBuffer, downloadStream); err != nil {
		log.Panic(err)
	}
}

func ExampleGridFSBucket_DownloadToStream() {
	var bucket *mongo.GridFSBucket
	var fileID bson.ObjectID

	ctx := context.Background()

	fileBuffer := bytes.NewBuffer(nil)
	if _, err := bucket.DownloadToStream(ctx, fileID, fileBuffer); err != nil {
		log.Panic(err)
	}
}

func ExampleGridFSBucket_Delete() {
	var bucket *mongo.GridFSBucket
	var fileID bson.ObjectID

	if err := bucket.Delete(context.Background(), fileID); err != nil {
		log.Panic(err)
	}
}

func ExampleGridFSBucket_Find() {
	var bucket *mongo.GridFSBucket

	// Specify a filter to find all files with a length greater than 1000 bytes.
	filter := bson.D{
		{"length", bson.D{{"$gt", 1000}}},
	}
	cursor, err := bucket.Find(context.Background(), filter)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := cursor.Close(context.TODO()); err != nil {
			log.Panic(err)
		}
	}()

	type gridfsFile struct {
		Name   string `bson:"filename"`
		Length int64  `bson:"length"`
	}
	var foundFiles []gridfsFile
	if err = cursor.All(context.TODO(), &foundFiles); err != nil {
		log.Panic(err)
	}

	for _, file := range foundFiles {
		fmt.Printf("filename: %s, length: %d\n", file.Name, file.Length)
	}
}

func ExampleGridFSBucket_Rename() {
	var bucket *mongo.GridFSBucket
	var fileID bson.ObjectID

	ctx := context.Background()

	if err := bucket.Rename(ctx, fileID, "new file name"); err != nil {
		log.Panic(err)
	}
}

func ExampleGridFSBucket_Drop() {
	var bucket *mongo.GridFSBucket

	if err := bucket.Drop(context.Background()); err != nil {
		log.Panic(err)
	}
}
