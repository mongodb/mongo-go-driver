// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// TopologyVersion represents a software version.
type TopologyVersion struct {
	ProcessID primitive.ObjectID
	Counter   int64
}

// NewTopologyVersion creates a TopologyVersion based on doc
func NewTopologyVersion(doc bsoncore.Document) (*TopologyVersion, error) {
	elements, err := doc.Elements()
	if err != nil {
		return nil, err
	}
	var tv TopologyVersion
	var ok bool
	for _, element := range elements {
		switch element.Key() {
		case "processId":
			tv.ProcessID, ok = element.Value().ObjectIDOK()
			if !ok {
				return nil, fmt.Errorf("expected 'processId' to be a objectID but it's a BSON %s", element.Value().Type)
			}
		case "counter":
			tv.Counter, ok = element.Value().Int64OK()
			if !ok {
				return nil, fmt.Errorf("expected 'counter' to be an int64 but it's a BSON %s", element.Value().Type)
			}
		}
	}
	return &tv, nil
}

// CompareTopologyVersion returns -1 if tv1<tv2, 0 if tv1==tv2, 1 if tv1>tv2. This comparsion is not communtative
// so the original TopologyVersion should be first.
func CompareTopologyVersion(tv1, tv2 *TopologyVersion) int {
	if tv1 == nil || tv2 == nil {
		return -1
	}
	if tv1.ProcessID != tv2.ProcessID {
		return -1
	}
	if tv1.Counter == tv2.Counter {
		return 0
	}
	if tv1.Counter < tv2.Counter {
		return -1
	}
	return 1
}
