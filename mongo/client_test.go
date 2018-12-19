// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/network/connstring"
)

func ExampleClient_Connect() {
	client, err := NewClient("mongodb://foo:bar@localhost:27017")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return
	}

	return
}

func TestClient(t *testing.T) {
	t.Run("Can Set ReadPreference", func(t *testing.T) {
		t.Run("from connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?readPreference=secondary")
			noerr(t, err)
			want, err := readpref.New(readpref.SecondaryMode)
			noerr(t, err)
			client, err := newClient(cs)
			noerr(t, err)
			got := client.readPreference
			if got.Mode() != want.Mode() {
				t.Errorf("ReadPreference mode not set correctly. got %v; want %v", got.Mode(), want.Mode())
			}
		})
		t.Run("from option", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost")
			noerr(t, err)
			want, err := readpref.New(readpref.SecondaryMode)
			noerr(t, err)
			client, err := newClient(cs, options.Client().SetReadPreference(want))
			noerr(t, err)
			got := client.readPreference
			if got.Mode() != want.Mode() {
				t.Errorf("ReadPreference mode not set correctly. got %v; want %v", got.Mode(), want.Mode())
			}
		})
		t.Run("option overrides connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?readPreference=primaryPreferred")
			noerr(t, err)
			want, err := readpref.New(readpref.SecondaryPreferredMode)
			noerr(t, err)
			client, err := newClient(cs, options.Client().SetReadPreference(want))
			noerr(t, err)
			got := client.readPreference
			if got.Mode() != want.Mode() {
				t.Errorf("ReadPreference mode not set correctly. got %v; want %v", got.Mode(), want.Mode())
			}
		})
	})
	t.Run("Can Set ReadConcern", func(t *testing.T) {
		t.Run("from connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?readConcernlevel=majority")
			noerr(t, err)
			rc := readconcern.Majority()
			client, err := newClient(cs)
			noerr(t, err)
			wantT, wantData, err := rc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.readConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("ReadConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("from option", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost")
			noerr(t, err)
			rc := readconcern.Majority()
			client, err := newClient(cs, options.Client().SetReadConcern(rc))
			noerr(t, err)
			wantT, wantData, err := rc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.readConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("ReadConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("option overrides connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?readConcernLevel=majority")
			noerr(t, err)
			rc := readconcern.Linearizable()
			client, err := newClient(cs, options.Client().SetReadConcern(rc))
			noerr(t, err)
			wantT, wantData, err := rc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.readConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("ReadConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
	})
	t.Run("Can Set WriteConcern", func(t *testing.T) {
		t.Run("from connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?w=majority")
			noerr(t, err)
			wc := writeconcern.New(writeconcern.WMajority())
			client, err := newClient(cs)
			noerr(t, err)
			wantT, wantData, err := wc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.writeConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("WriteConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("from option", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost")
			noerr(t, err)
			wc := writeconcern.New(writeconcern.WMajority())
			client, err := newClient(cs, options.Client().SetWriteConcern(wc))
			noerr(t, err)
			wantT, wantData, err := wc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.writeConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("WriteConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("option overrides connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?w=1")
			noerr(t, err)
			wc := writeconcern.New(writeconcern.WMajority())
			client, err := newClient(cs, options.Client().SetWriteConcern(wc))
			noerr(t, err)
			wantT, wantData, err := wc.MarshalBSONValue()
			noerr(t, err)
			gotT, gotData, err := client.writeConcern.MarshalBSONValue()
			noerr(t, err)
			want, got := bson.RawValue{Type: wantT, Value: wantData}, bson.RawValue{Type: gotT, Value: gotData}
			if !cmp.Equal(got, want) {
				t.Errorf("WriteConcern mode not set correctly. got %v; want %v", got, want)
			}
		})
	})
	t.Run("Can Set RetryWrites", func(t *testing.T) {
		t.Run("from connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?retryWrites=true")
			noerr(t, err)
			client, err := newClient(cs)
			noerr(t, err)
			want := true
			got := client.retryWrites
			if !cmp.Equal(got, want) {
				t.Errorf("RetryWrites mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("from option", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost")
			noerr(t, err)
			client, err := newClient(cs, options.Client().SetRetryWrites(true))
			noerr(t, err)
			want := true
			got := client.retryWrites
			if !cmp.Equal(got, want) {
				t.Errorf("RetryWrites mode not set correctly. got %v; want %v", got, want)
			}
		})
		t.Run("option overrides connstring", func(t *testing.T) {
			cs, err := connstring.Parse("mongodb://localhost/?retryWrites=true")
			noerr(t, err)
			client, err := newClient(cs, options.Client().SetRetryWrites(false))
			noerr(t, err)
			want := false
			got := client.retryWrites
			if !cmp.Equal(got, want) {
				t.Errorf("RetryWrites mode not set correctly. got %v; want %v", got, want)
			}
		})
	})
}
