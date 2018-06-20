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

// Delete represents the delete command.
//
// The delete command executes a delete with a given set of delete documents
// and options.
type Delete struct {
	NS           Namespace
	Deletes      []*bson.Document
	Opts         []option.DeleteOptioner
	WriteConcern *writeconcern.WriteConcern

	result result.Delete
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (d *Delete) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := d.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (d *Delete) encode(desc description.SelectedServer) (*Write, error) {
	if err := d.NS.Validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.EC.String("delete", d.NS.Collection))

	arr := bson.NewArray()
	for _, doc := range d.Deletes {
		arr.Append(bson.VC.Document(doc))
	}
	command.Append(bson.EC.Array("deletes", arr))

	for _, opt := range d.Opts {
		switch opt.(type) {
		case nil:
		case option.OptCollation:
			for _, doc := range d.Deletes {
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
		DB:           d.NS.DB,
		Command:      command,
		WriteConcern: d.WriteConcern,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (d *Delete) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Delete {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		d.err = err
		return d
	}

	return d.decode(desc, rdr)
}

func (d *Delete) decode(desc description.SelectedServer, rdr bson.Reader) *Delete {
	d.err = bson.Unmarshal(rdr, &d.result)
	return d
}

// Result returns the result of a decoded wire message and server description.
func (d *Delete) Result() (result.Delete, error) {
	if d.err != nil {
		return result.Delete{}, d.err
	}
	return d.result, nil
}

// Err returns the error set on this command.
func (d *Delete) Err() error { return d.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (d *Delete) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.Delete, error) {
	cmd, err := d.encode(desc)
	if err != nil {
		return result.Delete{}, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return result.Delete{}, err
	}

	return d.decode(desc, rdr).Result()
}
