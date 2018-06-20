// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unepexted error: %v", err)
		t.FailNow()
	}
}

func TestCommandEncode(t *testing.T) {
	t.Run("sets slaveOk for non-primary read preference mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(),
		}
		wm, err := cmd.Encode(description.SelectedServer{})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
			t.FailNow()
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
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
			t.FailNow()
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
	})
	t.Run("sets slaveOK for all read commands in direct mode", func(t *testing.T) {
		cmd := &Read{}
		wm, err := cmd.Encode(description.SelectedServer{Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
			t.FailNow()
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
	})
}
