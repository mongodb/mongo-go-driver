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

type dbOrCollectionOptions struct {
	DBOptions         *options.DatabaseOptions
	CollectionOptions *options.CollectionOptions
}

var _ bson.Unmarshaler = (*dbOrCollectionOptions)(nil)

// UnmarshalBSON specifies custom BSON unmarshalling behavior to convert db/collection options from BSON/JSON documents
// to their corresponding Go objects.
func (d *dbOrCollectionOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		RC    *readConcern           `bson:"readConcern"`
		RP    *ReadPreference        `bson:"readPreference"`
		WC    *writeConcern          `bson:"writeConcern"`
		Extra map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary dbOrCollectionOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for dbOrCollectionOptions: %v", mapKeys(temp.Extra))
	}

	d.DBOptions = options.Database()
	d.CollectionOptions = options.Collection()
	if temp.RC != nil {
		rc := temp.RC.toReadConcernOption()
		d.DBOptions.SetReadConcern(rc)
		d.CollectionOptions.SetReadConcern(rc)
	}
	if temp.RP != nil {
		rp, err := temp.RP.ToReadPrefOption()
		if err != nil {
			return fmt.Errorf("error parsing read preference document: %v", err)
		}

		d.DBOptions.SetReadPreference(rp)
		d.CollectionOptions.SetReadPreference(rp)
	}
	if temp.WC != nil {
		wc, err := temp.WC.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error parsing write concern document: %v", err)
		}

		d.DBOptions.SetWriteConcern(wc)
		d.CollectionOptions.SetWriteConcern(wc)
	}

	return nil
}
