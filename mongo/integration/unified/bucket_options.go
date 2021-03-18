// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// gridFSBucketOptions is a wrapper for *options.BucketOptions. This type implements the bson.Unmarshaler interface to
// convert BSON documents to a BucketOptions instance.
type gridFSBucketOptions struct {
	*options.BucketOptions
}

var _ bson.Unmarshaler = (*gridFSBucketOptions)(nil)

func (bo *gridFSBucketOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		Name      *string                `bson:"name"`
		ChunkSize *int32                 `bson:"chunkSizeBytes"`
		RC        *readConcern           `bson:"readConcern"`
		RP        *readPreference        `bson:"readPreference"`
		WC        *writeConcern          `bson:"writeConcern"`
		Extra     map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary gridFSBucketOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for gridFSBucketOptions: %v", mapKeys(temp.Extra))
	}

	bo.BucketOptions = options.GridFSBucket()
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
		rp, err := temp.RP.toReadPrefOption()
		if err != nil {
			return fmt.Errorf("error parsing read preference document: %v", err)
		}
		bo.SetReadPreference(rp)
	}
	if temp.WC != nil {
		wc, err := temp.WC.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error parsing write concern document: %v", err)
		}
		bo.SetWriteConcern(wc)
	}
	return nil
}
