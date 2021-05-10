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

// serverAPIOptions is a wrapper for *options.ServerAPIOptions. This type implements the bson.Unmarshaler interface
// to convert BSON documents to a serverAPIOptions instance.
type serverAPIOptions struct {
	*options.ServerAPIOptions
}

type serverAPIVersion = options.ServerAPIVersion

var _ bson.Unmarshaler = (*serverAPIOptions)(nil)

func (s *serverAPIOptions) UnmarshalBSON(data []byte) error {
	var temp struct {
		ServerAPIVersion  serverAPIVersion       `bson:"version"`
		DeprecationErrors *bool                  `bson:"deprecationErrors"`
		Strict            *bool                  `bson:"strict"`
		Extra             map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary serverAPIOptions object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for serverAPIOptions: %v", mapKeys(temp.Extra))
	}

	s.ServerAPIOptions = options.ServerAPI(temp.ServerAPIVersion)
	if temp.DeprecationErrors != nil {
		s.SetDeprecationErrors(*temp.DeprecationErrors)
	}
	if temp.Strict != nil {
		s.SetStrict(*temp.Strict)
	}

	return nil
}
