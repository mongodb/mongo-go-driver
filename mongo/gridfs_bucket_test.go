// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestBucket_openDownloadStream(t *testing.T) {
	tests := []struct {
		name   string
		filter any
		err    error
	}{
		{
			name:   "nil filter",
			filter: nil,
			err:    ErrNilDocument,
		},
		{
			name:   "nonmatching filter",
			filter: bson.D{{"x", 1}},
			err:    ErrFileNotFound,
		},
	}

	cs := integtest.ConnString(t)
	clientOpts := options.Client().ApplyURI(cs.Original)

	integtest.AddTestServerAPIVersion(clientOpts)

	client, err := Connect(clientOpts)
	assert.Nil(t, err, "Connect error: %v", err)

	db := client.Database("bucket")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bucket := db.GridFSBucket()
			_, err = bucket.openDownloadStream(context.Background(), test.filter)
			assert.ErrorIs(t, err, test.err)
		})
	}
}
