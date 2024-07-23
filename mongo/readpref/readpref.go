// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package readpref defines read preferences for MongoDB queries.
package readpref

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

var errInvalidReadPreference = errors.New("can not specify tags, max staleness, or hedge with mode primary")

// Options defines the options for constructing a read preference.
type Options struct {
	// Maximum amount of time to allow a server to be considered eligible for
	// selection. The second return value indicates if this value has been set.
	MaxStaleness *time.Duration

	// Tag sets indicating which servers should be considered.
	TagSets []TagSet

	// Specify whether or not hedged reads are enabled for this read preference.
	HedgeEnabled *bool
}

func optionsExist(opts *Options) bool {
	return opts != nil && (opts.MaxStaleness != nil || len(opts.TagSets) > 0 || opts.HedgeEnabled != nil)
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	Mode Mode
	opts Options
}

// New creates a new ReadPref.
func New(mode Mode, opts *Options) (*ReadPref, error) {
	if mode == PrimaryMode && optionsExist(opts) {
		return nil, errInvalidReadPreference
	}

	rp := &ReadPref{
		Mode: mode,
	}

	if opts != nil {
		rp.opts = *opts
	}

	return rp, nil
}

// Primary constructs a read preference with a PrimaryMode.
func Primary() *ReadPref {
	return &ReadPref{Mode: PrimaryMode}
}

// MaxStaleness is the maximum amount of time to allow
// a server to be considered eligible for selection. The
// second return value indicates if this value has been set.
func (r *ReadPref) MaxStaleness() *time.Duration {
	return r.opts.MaxStaleness
}

// TagSets are multiple tag sets indicating
// which servers should be considered.
func (r *ReadPref) TagSets() []TagSet {
	return r.opts.TagSets
}

// HedgeEnabled returns whether or not hedged reads are enabled for this read preference. If this option was not
// specified during read preference construction, nil is returned.
func (r *ReadPref) HedgeEnabled() *bool {
	return r.opts.HedgeEnabled
}

// String returns a human-readable description of the read preference.
func (r *ReadPref) String() string {
	var b bytes.Buffer
	b.WriteString(r.Mode.String())
	delim := "("
	if r.MaxStaleness() != nil {
		fmt.Fprintf(&b, "%smaxStaleness=%v", delim, r.MaxStaleness())
		delim = " "
	}
	for _, tagSet := range r.TagSets() {
		fmt.Fprintf(&b, "%stagSet=%s", delim, tagSet.String())
		delim = " "
	}
	if r.HedgeEnabled() != nil {
		fmt.Fprintf(&b, "%shedgeEnabled=%v", delim, *r.HedgeEnabled())
		delim = " "
	}
	if delim != "(" {
		b.WriteString(")")
	}
	return b.String()
}
