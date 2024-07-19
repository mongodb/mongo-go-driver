// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package readpref defines read preferences for MongoDB queries.
package readpref

import (
	"bytes"
	"fmt"
	"time"
)

// Primary constructs a read preference with a PrimaryMode.
func Primary() *ReadPref {
	return &ReadPref{Mode: PrimaryMode}
}

// PrimaryPreferred constructs a read preference with a PrimaryPreferredMode.
func PrimaryPreferred() *ReadPref {
	return &ReadPref{Mode: PrimaryPreferredMode}
}

// SecondaryPreferred constructs a read preference with a SecondaryPreferredMode.
func SecondaryPreferred() *ReadPref {
	return &ReadPref{Mode: SecondaryPreferredMode}
}

// Secondary constructs a read preference with a SecondaryMode.
func Secondary() *ReadPref {
	return &ReadPref{Mode: SecondaryMode}
}

// Nearest constructs a read preference with a NearestMode.
func Nearest() *ReadPref {
	return &ReadPref{Mode: NearestMode}
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	// Maximum amount of time to allow a server to be considered eligible for
	// selection. The second return value indicates if this value has been set.
	MaxStaleness *time.Duration

	// Mode indicates the mode of the read preference.
	Mode Mode

	// Tag sets indicating which servers should be considered.
	TagSets []TagSet

	// Specify whether or not hedged reads are enabled for this read preference.
	HedgeEnabled *bool
}

// String returns a human-readable description of the read preference.
func (r *ReadPref) String() string {
	var b bytes.Buffer
	b.WriteString(r.Mode.String())
	delim := "("
	if r.MaxStaleness != nil {
		fmt.Fprintf(&b, "%smaxStaleness=%v", delim, r.MaxStaleness)
		delim = " "
	}
	for _, tagSet := range r.TagSets {
		fmt.Fprintf(&b, "%stagSet=%s", delim, tagSet.String())
		delim = " "
	}
	if r.HedgeEnabled != nil {
		fmt.Fprintf(&b, "%shedgeEnabled=%v", delim, *r.HedgeEnabled)
		delim = " "
	}
	if delim != "(" {
		b.WriteString(")")
	}
	return b.String()
}
