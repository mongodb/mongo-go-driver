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

// MoreRecentThan returns if this TopologyVersion is more recent than the one passed in.
// This should be called on the original TopologyVersion with the new one passed in.
func (tv *TopologyVersion) MoreRecentThan(other *TopologyVersion) bool {
	if tv == nil || other == nil {
		return false
	}
	if tv.ProcessID != other.ProcessID {
		return false
	}

	return tv.Counter > other.Counter
}
