// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/tag"
)

// This file defines helper types to convert BSON documents to ReadConcern, WriteConcern, and ReadPref objects.

type readConcern struct {
	Level string `bson:"level"`
}

func (rc *readConcern) toReadConcernOption() *readconcern.ReadConcern {
	return readconcern.New(readconcern.Level(rc.Level))
}

type writeConcern struct {
	Journal    *bool       `bson:"journal"`
	W          interface{} `bson:"w"`
	WTimeoutMS *int32      `bson:"wtimeoutMS"`
}

func (wc *writeConcern) toWriteConcernOption() (*writeconcern.WriteConcern, error) {
	var wcOptions []writeconcern.Option
	if wc.Journal != nil {
		wcOptions = append(wcOptions, writeconcern.J(*wc.Journal))
	}
	if wc.W != nil {
		switch converted := wc.W.(type) {
		case string:
			if converted != "majority" {
				return nil, fmt.Errorf("invalid write concern 'w' string value %q", converted)
			}
			wcOptions = append(wcOptions, writeconcern.WMajority())
		case int32:
			wcOptions = append(wcOptions, writeconcern.W(int(converted)))
		default:
			return nil, fmt.Errorf("invalid type for write concern 'w' field %T", wc.W)
		}
	}
	if wc.WTimeoutMS != nil {
		wTimeout := time.Duration(*wc.WTimeoutMS) * time.Millisecond
		wcOptions = append(wcOptions, writeconcern.WTimeout(wTimeout))
	}

	return writeconcern.New(wcOptions...), nil
}

// ReadPreference is a representation of BSON readPreference objects in tests.
type ReadPreference struct {
	Mode                string              `bson:"mode"`
	TagSets             []map[string]string `bson:"tagSets"`
	MaxStalenessSeconds *int64              `bson:"maxStalenessSeconds"`
	Hedge               bson.M              `bson:"hedge"`
}

// ToReadPrefOption converts a ReadPreference into a readpref.ReadPref object and will
// error if the original ReadPreference is malformed.
func (rp *ReadPreference) ToReadPrefOption() (*readpref.ReadPref, error) {
	mode, err := readpref.ModeFromString(rp.Mode)
	if err != nil {
		return nil, fmt.Errorf("invalid read preference mode %q", rp.Mode)
	}

	var rpOptions []readpref.Option
	if rp.TagSets != nil {
		// Each item in the TagSets slice is a document that represents one set.
		sets := make([]tag.Set, 0, len(rp.TagSets))
		for _, rawSet := range rp.TagSets {
			parsed := make(tag.Set, 0, len(rawSet))
			for k, v := range rawSet {
				parsed = append(parsed, tag.Tag{Name: k, Value: v})
			}
			sets = append(sets, parsed)
		}

		rpOptions = append(rpOptions, readpref.WithTagSets(sets...))
	}
	if rp.MaxStalenessSeconds != nil {
		maxStaleness := time.Duration(*rp.MaxStalenessSeconds) * time.Second
		rpOptions = append(rpOptions, readpref.WithMaxStaleness(maxStaleness))
	}
	if rp.Hedge != nil {
		if len(rp.Hedge) > 1 {
			return nil, fmt.Errorf("invalid read preference hedge document: length cannot be greater than 1")
		}
		if enabled, ok := rp.Hedge["enabled"]; ok {
			rpOptions = append(rpOptions, readpref.WithHedgeEnabled(enabled.(bool)))
		}
	}

	return readpref.New(mode, rpOptions...)
}
