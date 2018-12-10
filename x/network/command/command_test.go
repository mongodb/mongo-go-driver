// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
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
	t.Run("sets slaveOK for all read commands to non-mongos in direct mode", func(t *testing.T) {
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
	t.Run("readPreference primary is overwritten as primaryPreferred in op_msg for read commands to non-mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Primary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.RSSecondary, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "primaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'primaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("default readPreference in op_msg is primaryPreferred for read commands to non-mongos in direct mode", func(t *testing.T) {
		cmd := &Read{}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.RSSecondary, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "primaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'primaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("application-supplied non-primary readPreferences are used in op_msg for read commands to non-mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Secondary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.RSSecondary, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondary" {
			t.Errorf("Expected $readPreference to be set to 'secondary', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference primary is not used in op_msg for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Primary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err == nil {
			t.Errorf("Did not expect $readPreference to be set, but it was. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred without options is used in op_msg for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'secondaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred with positive max staleness is used in op_msg for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(readpref.WithMaxStaleness(time.Duration(10))),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'secondaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred with non-empty tag sets is used in op_msg for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(readpref.WithTags("a", "2")),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'secondaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("all other readPreferences are used in op_msg for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Secondary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos, WireVersion: &description.VersionRange{Max: 6}}, Kind: description.Single})
		noerr(t, err)
		msg, ok := wm.(wiremessage.Msg)
		if !ok {
			t.Errorf("Returned wiremessage is not a msg. got %T; want %T", wm, wiremessage.Msg{})
			t.FailNow()
		}
		res, _ := msg.GetMainDocument()
		rp, err := res.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondary" {
			t.Errorf("Expected $readPreference to be set to 'secondary', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference primary is not used and slaveOk is not set in op_query for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Primary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos}, Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK == wiremessage.SlaveOK {
			t.Errorf("Didn't expect the slaveOk flag to be set, but it was. got %v", query.Flags)
		}
		rp, err := query.Query.LookupErr("$readPreference", "mode")
		if err == nil {
			t.Errorf("Did not expect $readPreference to be set, but it was. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred with positive max staleness is used and slaveOk is set in op_query for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(readpref.WithMaxStaleness(time.Duration(10))),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos}, Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
		rp, err := query.Query.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'secondaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred with non-empty tag sets is used and slaveOk is set in op_query for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(readpref.WithTags("a", "2")),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos}, Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
		rp, err := query.Query.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondaryPreferred" {
			t.Errorf("Expected $readPreference to be set to 'secondaryPreferred', but it wasn't. got %v", rp.StringValue())
		}
	})
	t.Run("readPreference secondaryPreferred without options is not used but slaveOk is set in op_query for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.SecondaryPreferred(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos}, Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
		rp, err := query.Query.LookupErr("$readPreference", "mode")
		if err == nil {
			t.Errorf("Did not expect $readPreference to be set, but it was. got %v", rp.StringValue())
		}
	})
	t.Run("all other readPreferences are used and slaveOk is set in op_query for read commands to mongos in direct mode", func(t *testing.T) {
		cmd := &Read{
			ReadPref: readpref.Secondary(),
		}
		wm, err := cmd.Encode(description.SelectedServer{Server: description.Server{Kind: description.Mongos}, Kind: description.Single})
		noerr(t, err)
		query, ok := wm.(wiremessage.Query)
		if !ok {
			t.Errorf("Returned wiremessage is not a query. got %T; want %T", wm, wiremessage.Query{})
		}
		if query.Flags&wiremessage.SlaveOK != wiremessage.SlaveOK {
			t.Errorf("Expected the slaveOk flag to be set, but it wasn't. got %v; want %v", query.Flags, wiremessage.SlaveOK)
		}
		rp, err := query.Query.LookupErr("$readPreference", "mode")
		if err != nil {
			t.Errorf("Expected $readPreference to be set, but it wasn't.")
		} else if rp.StringValue() != "secondary" {
			t.Errorf("Expected $readPreference to be set to 'secondary', but it wasn't. got %v", rp.StringValue())
		}
	})
}
