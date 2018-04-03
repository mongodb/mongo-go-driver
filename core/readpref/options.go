// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"errors"
	"time"

	"github.com/mongodb/mongo-go-driver/core/tag"
)

// ErrInvalidTagSet indicates that an invalid set of tags was specified.
var ErrInvalidTagSet = errors.New("an even number of tags must be specified")

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
	return func(rp *ReadPref) error {
		if len(tags)%2 != 0 {
			return ErrInvalidTagSet
		}

		tagset := make(tag.Set, 0)

		for i := 0; i < len(tags)/2; i++ {
			tagset = append(tagset, tag.Tag{Name: tags[i], Value: tags[i+1]})
		}

		return WithTagSets(tagset)(rp)
	}
}

// WithTagSets sets the tag sets used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTagSets(tagSets ...tag.Set) Option {
	return func(rp *ReadPref) error {
		rp.tagSets = tagSets
		return nil
	}
}
