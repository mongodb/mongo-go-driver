// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// gridFSBucketOptions is a wrapper for *options.BucketOptionsBuilder. This type
// implements the bson.Unmarshaler interface to convert BSON documents to a
// BucketOptionsBuilder instance.
type gridFSBucketOptions struct {
	*options.BucketOptionsBuilder
}

var _ bson.Unmarshaler = (*gridFSBucketOptions)(nil)

func (bo *gridFSBucketOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		Name      *string                `bson:"name"`
		ChunkSize *int32                 `bson:"chunkSizeBytes"`
		RC        *readConcern           `bson:"readConcern"`
		RP        *ReadPreference        `bson:"readPreference"`
		WC        *writeConcern          `bson:"writeConcern"`
		Extra     map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary gridFSBucketOptions object: %w", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for gridFSBucketOptions: %v", mapKeys(temp.Extra))
	}

	bo.BucketOptionsBuilder = options.GridFSBucket()
	if temp.Name != nil {
		bo.SetName(*temp.Name)
	}
	if temp.ChunkSize != nil {
		bo.SetChunkSizeBytes(*temp.ChunkSize)
	}
	if temp.RC != nil {
		bo.SetReadConcern(temp.RC.toReadConcernOption())
	}
	if temp.RP != nil {
		rp, err := temp.RP.ToReadPrefOption()
		if err != nil {
			return fmt.Errorf("error parsing read preference document: %w", err)
		}
		bo.SetReadPreference(rp)
	}
	if temp.WC != nil {
		wc, err := temp.WC.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error parsing write concern document: %w", err)
		}
		bo.SetWriteConcern(wc)
	}
	return nil
}
