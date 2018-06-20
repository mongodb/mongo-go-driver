// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Update represents the update command.
//
// The update command updates a set of documents with the database.
type Update struct {
	NS           Namespace
	Docs         []*bson.Document
	Opts         []option.UpdateOptioner
	WriteConcern *writeconcern.WriteConcern

	result result.Update
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (u *Update) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	encoded, err := u.encode(desc)
	if err != nil {
		return nil, err
	}
	return encoded.Encode(desc)
}

func (u *Update) encode(desc description.SelectedServer) (*Write, error) {
	command := bson.NewDocument(bson.EC.String("update", u.NS.Collection))
	vals := make([]*bson.Value, 0, len(u.Docs))
	for _, doc := range u.Docs {
		vals = append(vals, bson.VC.Document(doc))
	}
	command.Append(bson.EC.ArrayFromElements("updates", vals...))

	for _, opt := range u.Opts {
		switch opt.(type) {
		case nil:
			continue
		case option.OptUpsert, option.OptCollation, option.OptArrayFilters:
			for _, doc := range u.Docs {
				err := opt.Option(doc)
				if err != nil {
					return nil, err
				}
			}
		default:
			err := opt.Option(command)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Write{
		DB:           u.NS.DB,
		Command:      command,
		WriteConcern: u.WriteConcern,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (u *Update) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Update {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		u.err = err
		return u
	}
	return u.decode(desc, rdr)
}

func (u *Update) decode(desc description.SelectedServer, rdr bson.Reader) *Update {
	u.err = bson.Unmarshal(rdr, &u.result)
	return u
}

// Result returns the result of a decoded wire message and server description.
func (u *Update) Result() (result.Update, error) {
	if u.err != nil {
		return result.Update{}, u.err
	}
	return u.result, nil
}

// Err returns the error set on this command.
func (u *Update) Err() error { return u.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (u *Update) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.Update, error) {
	cmd, err := u.encode(desc)
	if err != nil {
		return result.Update{}, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return result.Update{}, err
	}
	return u.decode(desc, rdr).Result()
}
