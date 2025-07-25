// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package writeconcern defines write concerns for MongoDB operations.
//
// For more information about MongoDB write concerns, see
// https://www.mongodb.com/docs/manual/reference/write-concern/
package writeconcern

// WCMajority can be used to create a WriteConcern with a W value of "majority".
const WCMajority = "majority"

// A WriteConcern defines a MongoDB write concern, which describes the level of acknowledgment
// requested from MongoDB for write operations to a standalone mongod, to replica sets, or to
// sharded clusters.
//
// For more information about MongoDB write concerns, see
// https://www.mongodb.com/docs/manual/reference/write-concern/
type WriteConcern struct {
	// W requests acknowledgment that the write operation has propagated to a
	// specified number of mongod instances or to mongod instances with
	// specified tags. It sets the "w" option in a MongoDB write concern.
	//
	// W values must be a string or an int.
	//
	// Common values are:
	//   - "majority": requests acknowledgment that write operations have been
	//     durably committed to the calculated majority of the data-bearing
	//     voting members.
	//   - 1: requests acknowledgment that write operations have been written
	//     to 1 node.
	//   - 0: requests no acknowledgment of write operations
	//
	// For more information about the "w" option, see
	// https://www.mongodb.com/docs/manual/reference/write-concern/#w-option
	W any

	// Journal requests acknowledgment from MongoDB that the write operation has
	// been written to the on-disk journal. It sets the "j" option in a MongoDB
	// write concern.
	//
	// For more information about the "j" option, see
	// https://www.mongodb.com/docs/manual/reference/write-concern/#j-option
	Journal *bool
}

// Unacknowledged returns a WriteConcern that requests no acknowledgment of
// write operations.
//
// For more information about write concern "w: 0", see
// https://www.mongodb.com/docs/manual/reference/write-concern/#mongodb-writeconcern-writeconcern.-number-
func Unacknowledged() *WriteConcern {
	return &WriteConcern{W: 0}
}

// W1 returns a WriteConcern that requests acknowledgment that write operations
// have been written to memory on one node (e.g. the standalone mongod or the
// primary in a replica set).
//
// For more information about write concern "w: 1", see
// https://www.mongodb.com/docs/manual/reference/write-concern/#mongodb-writeconcern-writeconcern.-number-
func W1() *WriteConcern {
	return &WriteConcern{W: 1}
}

// Journaled returns a WriteConcern that requests acknowledgment that write
// operations have been written to the on-disk journal on MongoDB.
//
// The database's default value for "w" determines how many nodes must write to
// their on-disk journal before the write operation is acknowledged.
//
// For more information about write concern "j: true", see
// https://www.mongodb.com/docs/manual/reference/write-concern/#mongodb-writeconcern-ournal
func Journaled() *WriteConcern {
	journal := true
	return &WriteConcern{Journal: &journal}
}

// Majority returns a WriteConcern that requests acknowledgment that write
// operations have been durably committed to the calculated majority of the
// data-bearing voting members.
//
// Write concern "w: majority" typically requires write operations to be written
// to the on-disk journal before they are acknowledged, unless journaling is
// disabled on MongoDB or the "writeConcernMajorityJournalDefault" replica set
// configuration is set to false.
//
// For more information about write concern "w: majority", see
// https://www.mongodb.com/docs/manual/reference/write-concern/#mongodb-writeconcern-writeconcern.-majority-
func Majority() *WriteConcern {
	return &WriteConcern{W: WCMajority}
}

// Custom returns a WriteConcern that requests acknowledgment that write
// operations have propagated to tagged members that satisfy the custom write
// concern defined in "settings.getLastErrorModes".
//
// For more information about custom write concern names, see
// https://www.mongodb.com/docs/manual/reference/write-concern/#mongodb-writeconcern-writeconcern.-custom-write-concern-name-
func Custom(tag string) *WriteConcern {
	return &WriteConcern{W: tag}
}

// Acknowledged indicates whether or not a write with the given write concern will be acknowledged.
func (wc *WriteConcern) Acknowledged() bool {
	// Only {w: 0} or {w: 0, j: false} are an unacknowledged write concerns. All other values are
	// acknowledged.
	return wc == nil || wc.W != 0 || (wc.Journal != nil && *wc.Journal)
}

// IsValid returns true if the WriteConcern is valid.
func (wc *WriteConcern) IsValid() bool {
	if wc == nil {
		return true
	}

	switch w := wc.W.(type) {
	case int:
		// A write concern with {w: int} must have a non-negative value and
		// cannot have the combination {w: 0, j: true}.
		return w >= 0 && (w > 0 || wc.Journal == nil || !*wc.Journal)
	case string, nil:
		// A write concern with {w: string} or no w specified is always valid.
		return true
	default:
		// A write concern with an unsupported w type is not valid.
		return false
	}
}
