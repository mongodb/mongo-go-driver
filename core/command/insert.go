// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/options"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/pkg/errors"
)

// Insert represents the insert command.
//
// The insert command inserts a set of documents into the database.
//
// Since the Insert command does not return any value other than ok or
// an error, this type has no Err method.
type Insert struct {
	NS   Namespace
	Docs []*bson.Document
	Opts []options.InsertOptioner

	result result.Insert
	err    error
}

func (i *Insert) split(maxCount, targetBatchSize int) [][]*bson.Document {
	batches := [][]*bson.Document{}

	// remove
	targetBatchSize -= 16 * 1000

	if maxCount <= 0 {
		maxCount = 1
	}

	startAt := 0
splitInserts:
	for {
		size := 0
		batch := []*bson.Document{}
	assembleBatch:
		for idx := startAt; idx < len(i.Docs); idx++ {
			itsize, _ := i.Docs[idx].Validate()
			if size+int(itsize) > int(targetBatchSize) {
				break assembleBatch
			}

			size += int(itsize)
			batch = append(batch, i.Docs[idx])
			startAt++
			if len(batch) == maxCount {
				break assembleBatch
			}
		}
		batches = append(batches, batch)
		if startAt == len(i.Docs) {
			break splitInserts
		}
	}

	return batches
}

func (i *Insert) getMessage(docs []*bson.Document, desc description.SelectedServer) (wiremessage.WireMessage, error) {

	command := bson.NewDocument(bson.EC.String("insert", i.NS.Collection))

	vals := make([]*bson.Value, 0, len(docs))
	for _, doc := range docs {
		vals = append(vals, bson.VC.Document(doc))
	}
	command.Append(bson.EC.ArrayFromElements("documents", vals...))

	for _, option := range i.Opts {
		if option == nil {
			continue
		}
		err := option.Option(command)
		if err != nil {
			return nil, err
		}
	}

	return (&Command{DB: i.NS.DB, Command: command, isWrite: true}).Encode(desc)
}

// Encode will encode this command into a wire message for the given server description.
func (i *Insert) Encode(desc description.SelectedServer) ([]wiremessage.WireMessage, error) {
	out := []wiremessage.WireMessage{}
	batches := i.split(int(desc.MaxBatchCount), int(desc.MaxDocumentSize))

	for _, docs := range batches {
		cmd, err := i.getMessage(docs, desc)
		if err != nil {
			return nil, err
		}

		out = append(out, cmd)
	}

	return out, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (i *Insert) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Insert {
	rdr, err := (&Command{}).Decode(desc, wm).Result()
	if err != nil {
		i.err = err
		return i
	}

	i.err = bson.Unmarshal(rdr, &i.result)
	return i
}

// Result returns the result of a decoded wire message and server description.
func (i *Insert) Result() (result.Insert, error) {
	if i.err != nil {
		return result.Insert{}, i.err
	}
	return i.result, nil
}

// Err returns the error set on this command.
func (i *Insert) Err() error { return i.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (i *Insert) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.Insert, error) {
	res := result.Insert{}

	wms, err := i.Encode(desc)
	if err != nil {
		return res, err
	}

	for _, wm := range wms {
		err = rw.WriteWireMessage(ctx, wm)
		if err != nil {
			return res, err
		}
		wm, err = rw.ReadWireMessage(ctx)
		if err != nil {
			return res, err
		}

		r, err := i.Decode(desc, wm).Result()
		if err != nil {
			return res, err
		}
		res.WriteErrors = append(res.WriteErrors, r.WriteErrors...)
		if res.WriteConcernError != nil {
			res.WriteConcernError = r.WriteConcernError
			return res, errors.New(res.WriteConcernError.ErrMsg)

		}
		res.N += r.N
	}

	fmt.Println(res)
	return res, nil
}
