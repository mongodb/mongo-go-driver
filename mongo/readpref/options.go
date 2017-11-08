// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"time"

	"github.com/10gen/mongo-go-driver/mongo/model"
)

// Option configures a read preference
type Option func(*ReadPref) error

// WithMaxStaleness sets the maximum staleness a
// server is allowed.
func WithMaxStaleness(ms time.Duration) Option {
	return func(rp *ReadPref) error {
		rp.maxStaleness = ms
		rp.maxStalenessSet = true
		return nil
	}
}

// WithTags sets a single tag set used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTags(tags ...string) Option {
	return WithTagSets(model.NewTagSet(tags...))
}

// WithTagSets sets the tag sets used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTagSets(tagSets ...model.TagSet) Option {
	return func(rp *ReadPref) error {
		rp.tagSets = tagSets
		return nil
	}
}
