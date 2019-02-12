// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

var connectDirectOpMsgTests = []struct {
	serverKind           description.ServerKind
	suppliedReadPref     *readpref.ReadPref
	expectedReadPrefMode readpref.Mode
}{
	{description.RSSecondary, nil, readpref.PrimaryPreferredMode},
	{description.RSSecondary, readpref.Primary(), readpref.PrimaryPreferredMode},
	{description.RSSecondary, readpref.Secondary(), readpref.SecondaryMode},
	{description.Mongos, readpref.Primary(), readpref.Mode(0)},
	{description.Mongos, readpref.SecondaryPreferred(), readpref.SecondaryPreferredMode},
	{description.Mongos, readpref.SecondaryPreferred(readpref.WithTags("a", "2")), readpref.SecondaryPreferredMode},
	{description.Mongos, readpref.SecondaryPreferred(readpref.WithMaxStaleness(time.Duration(10))), readpref.SecondaryPreferredMode},
	{description.Mongos, readpref.Secondary(), readpref.SecondaryMode},
}

var connectDirectOpQueryTests = []struct {
	serverKind           description.ServerKind
	suppliedReadPref     *readpref.ReadPref
	expectedReadPrefMode readpref.Mode
	expectedSlaveOkBit   wiremessage.QueryFlag
}{
	{description.Mongos, readpref.Primary(), readpref.Mode(0), 0},
	{description.Mongos, readpref.SecondaryPreferred(), readpref.Mode(0), wiremessage.SlaveOK},
	{description.Mongos, readpref.SecondaryPreferred(readpref.WithTags("a", "2")), readpref.SecondaryPreferredMode, wiremessage.SlaveOK},
	{description.Mongos, readpref.SecondaryPreferred(readpref.WithMaxStaleness(time.Duration(10))), readpref.SecondaryPreferredMode, wiremessage.SlaveOK},
	{description.Mongos, readpref.Secondary(), readpref.SecondaryMode, wiremessage.SlaveOK},
}

func TestCommandEncode(t *testing.T) {
	for _, tt := range connectDirectOpMsgTests {
		t.Run("connect direct op_msg", func(t *testing.T) {
			cmd := &Read{ReadPref: tt.suppliedReadPref}
			wm, err := cmd.Encode(description.SelectedServer{
				Server: description.Server{
					Kind:        tt.serverKind,
					WireVersion: &description.VersionRange{Max: 6},
				},
				Kind: description.Single,
			})
			noerr(t, err)

			msg, ok := wm.(wiremessage.Msg)
			if !ok {
				t.Fatalf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			}
			res, _ := msg.GetMainDocument()
			rp, err := res.LookupErr("$readPreference", "mode")
			if tt.expectedReadPrefMode == readpref.Mode(0) {
				if err == nil {
					t.Errorf("Did not expect $readPreference to be set, but it was. got %v", rp.StringValue())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected $readPreference to be set, but it wasn't.")
				}
				if mode, _ := readpref.ModeFromString(rp.StringValue()); mode != tt.expectedReadPrefMode {
					t.Errorf("Expected $readPreference to be set to %v, but it wasn't. got %v", tt.expectedReadPrefMode, mode)
				}
			}
		})
	}

	for _, tt := range connectDirectOpQueryTests {
		t.Run("connect direct op_query", func(t *testing.T) {
			cmd := &Read{ReadPref: tt.suppliedReadPref}
			wm, err := cmd.Encode(description.SelectedServer{
				Server: description.Server{Kind: tt.serverKind},
				Kind:   description.Single,
			})
			noerr(t, err)

			query, ok := wm.(wiremessage.Query)
			if !ok {
				t.Fatalf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
			}
			if query.Flags&wiremessage.SlaveOK != tt.expectedSlaveOkBit {
				t.Errorf("slaveOk bit did not have expected value. got %v; want %v", query.Flags, wiremessage.SlaveOK)
			}
			rp, err := query.Query.LookupErr("$readPreference", "mode")
			if tt.expectedReadPrefMode == readpref.Mode(0) {
				if err == nil {
					t.Errorf("Did not expect $readPreference to be set, but it was. got %v", rp.StringValue())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected $readPreference to be set, but it wasn't.")
				}
				if mode, _ := readpref.ModeFromString(rp.StringValue()); mode != tt.expectedReadPrefMode {
					t.Errorf("Expected $readPreference to be set to %v, but it wasn't. got %v", tt.expectedReadPrefMode, mode)
				}
			}
		})
	}

	t.Run("sets slaveOk for non-primary read preference mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(),
		}
		wm, err := cmd.Encode(description.SelectedServer{})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Fatalf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
	})
	t.Run("sets slaveOk for all write commands in direct mode", func(t *testing.T) {
		cmd := &Write{}
		wm, err := cmd.Encode(description.SelectedServer{Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Fatalf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
	})
	t.Run("sets slaveOK for all read commands to non-mongos in direct mode", func(t *testing.T) {
		cmd := &Read{}
		wm, err := cmd.Encode(description.SelectedServer{Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Fatalf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
	})
}
