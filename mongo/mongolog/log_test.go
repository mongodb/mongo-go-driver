// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestMongoLogger(t *testing.T) {
	var buf bytes.Buffer
	testLogger := defaultLogger{writer: &buf}
	allOptions := NewOptions().SetLogger(testLogger).SetLevel(Debug)
	individualOptions := NewOptions().SetLogger(testLogger).
		SetCommandLevel(Info).
		SetConnectionLevel(Notice).
		SetSDAMLevel(Trace).
		SetServerSelectionLevel(Error)
	testCases := []struct {
		name      string
		component Component
		opts      *Options
		logLevel  Level
		logged    bool
	}{
		{"Command Info ignores Trace", Command, individualOptions, Trace, false},
		{"Connection Notice logs Notice", Connection, individualOptions, Notice, true},
		{"SDAM Trace logs Warning", SDAM, individualOptions, Warning, true},
		{"ServerSelection Error ignores Debug", ServerSelection, individualOptions, Debug, false},
		{"SetLevel Debug logs Error", Command, allOptions, Error, true},
		{"SetLevel Debug ignores Trace", Connection, allOptions, Trace, false},
		{"Invalid Component ignored", Component(0), allOptions, Error, false},
		{"Invalid Level ignored", Connection, allOptions, Level(0), false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			ml, err := NewMongoLogger(tc.opts)
			assert.Nil(t, err, "Error creating MongoLogger: %v", err)
			ml.Log(tc.component, tc.logLevel, "foo", Int64("apple", int64(1)))
			got := buf.String()
			if tc.logged {
				want := fmt.Sprintf("{level:%v,msg:foo,apple:1}\n", tc.logLevel)
				assert.Equal(t, got, want, "wanted log: %v, got: %v", want, got)
			} else {
				assert.Equal(t, len(got), 0, "wanted empty log, got: %v", got)
			}
		})
	}
}

type testStringer struct {
	foo bool
}

func (testStringer) String() string {
	return "bar"
}

func TestDefaultLogger(t *testing.T) {
	t.Run("getField", func(t *testing.T) {
		var buf bytes.Buffer
		logger := defaultLogger{writer: &buf}
		stringer := testStringer{true}
		logger.log(Warning, "all fields",
			Bool("bool", false),
			Int64("int64", int64(2)),
			String("string", "apple"),
			Stringer("stringer", stringer),
		)
		got := buf.String()
		want := fmt.Sprintf("{level:warning,msg:all fields,bool:false,int64:2,string:apple,stringer:%v}\n", stringer)
		diff := cmp.Diff(got, want)
		assert.Equal(t, diff, "", "mismatched logs:%v", diff)
	})
}
