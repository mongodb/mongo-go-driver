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

// ServerAPIOptions is a wrapper for *options.ServerAPIOptions. This type implements the bson.Unmarshaler interface
// to convert BSON documents to a ServerAPIOptions instance.
type ServerAPIOptions struct {
	*options.ServerAPIOptions
}

type ServerAPIVersion = options.ServerAPIVersion

var _ bson.Unmarshaler = (*ServerAPIOptions)(nil)

func (to *ServerAPIOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		ServerAPIVersion  ServerAPIVersion       `bson:"version"`
		DeprecationErrors bool                   `bson:"deprecationErrors"`
		Strict            bool                   `bson:"strict"`
		Extra             map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary ServerAPIOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for ServerAPIOptions: %v", MapKeys(temp.Extra))
	}

	to.ServerAPIOptions = options.ServerAPI()
	if err := temp.ServerAPIVersion.CheckServerAPIVersion(); err != nil {
		return err
	}
	to.ServerAPIVersion = temp.ServerAPIVersion
	to.DeprecationErrors = temp.DeprecationErrors
	to.Strict = temp.Strict

	return nil
}
