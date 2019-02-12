// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/network/description"
	"testing"
)

func TestAggregate(t *testing.T) {
	wc := writeconcern.New(writeconcern.W(10))
	legacyDesc := description.SelectedServer{
		Server: description.Server{
			WireVersion: &description.VersionRange{
				Max: 4,
			},
		},
	}
	desc := description.SelectedServer{
		Server: description.Server{
			WireVersion: &description.VersionRange{
				Max: 5,
			},
		},
	}
	outDoc := bsonx.Doc{{"$out", bsonx.Int32(1)}}
	outPipeline := bsonx.Arr{bsonx.Document(outDoc)}

	testCases := []struct {
		name       string
		desc       description.SelectedServer
		pipeline   bsonx.Arr
		wcExpected bool
	}{
		{"LegacyDescNoOut", legacyDesc, bsonx.Arr{}, false},
		{"LegacyDescOut", legacyDesc, outPipeline, false},
		{"NewDescNoOut", desc, bsonx.Arr{}, false},
		{"NewDescOut", desc, outPipeline, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Aggregate{
				NS: Namespace{
					DB:         "db",
					Collection: "coll",
				},
				Pipeline:     tc.pipeline,
				WriteConcern: wc,
			}

			readCmd, err := cmd.encode(tc.desc)
			testhelpers.RequireNil(t, err, "error encoding: %s", err)

			_, err = readCmd.Command.LookupErr("writeConcern")
			nilErr := err == nil
			// err should be nil if wc was expected
			if nilErr != tc.wcExpected {
				t.Fatalf("write concern mismatch: expected %v got %v", tc.wcExpected, nilErr)
			}
		})
	}
}
