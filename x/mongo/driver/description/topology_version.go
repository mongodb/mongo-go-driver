// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TopologyVersion represents a software version.
type TopologyVersion struct {
	ProcessID primitive.ObjectID
	Counter   int64
}

// MoreRecentThan returns if this TopologyVersion is more recent than the one passed in
func (tv *TopologyVersion) MoreRecentThan(other *TopologyVersion) bool {
	if tv == nil || other == nil {
		return false
	}
	if tv.ProcessID != other.ProcessID {
		return false
	}

	return tv.Counter > other.Counter
}
