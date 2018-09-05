// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// this is the amount of reserved buffer space in a message that the
// driver reserves for command overhead.
const reservedCommandBufferBytes = 16 * 10 * 10 * 10

// Insert represents the insert command.
//
// The insert command inserts a set of documents into the database.
//
// Since the Insert command does not return any value other than ok or
// an error, this type has no Err method.
type Insert struct {
	Clock        *session.ClusterClock
	NS           Namespace
	Docs         []*bson.Document
	Opts         []option.InsertOptioner
	WriteConcern *writeconcern.WriteConcern
	Session      *session.Client

	batches         []*Write
	result          result.Insert
	err             error
	continueOnError bool
}

func (i *Insert) split(maxCount, targetBatchSize int) ([][]*bson.Document, error) {
	batches := [][]*bson.Document{}

	if targetBatchSize > reservedCommandBufferBytes {
		targetBatchSize -= reservedCommandBufferBytes
	}

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
			itsize, err := i.Docs[idx].Validate()
			if err != nil {
				return nil, err
			}

			if int(itsize) > targetBatchSize {
				return nil, ErrDocumentTooLarge
			}
			if size+int(itsize) > targetBatchSize {
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

	return batches, nil
}

// Encode will encode this command into a wire message for the given server description.
func (i *Insert) Encode(desc description.SelectedServer) ([]wiremessage.WireMessage, error) {
	err := i.encode(desc)
	if err != nil {
		return nil, err
	}

	wms := make([]wiremessage.WireMessage, len(i.batches))
	for _, cmd := range i.batches {
		wm, err := cmd.Encode(desc)
		if err != nil {
			return nil, err
		}

		wms = append(wms, wm)
	}

	return wms, nil
}

func (i *Insert) encodeBatch(docs []*bson.Document, desc description.SelectedServer) (*Write, error) {

	command := bson.NewDocument(bson.EC.String("insert", i.NS.Collection))

	vals := make([]*bson.Value, 0, len(docs))
	for _, doc := range docs {
		vals = append(vals, bson.VC.Document(doc))
	}
	command.Append(bson.EC.ArrayFromElements("documents", vals...))

	for _, opt := range i.Opts {
		if opt == nil {
			continue
		}

		if ordered, ok := opt.(option.OptOrdered); ok {
			if !ordered {
				i.continueOnError = true
			}
		}

		err := opt.Option(command)
		if err != nil {
			return nil, err
		}
	}

	return &Write{
		Clock:        i.Clock,
		DB:           i.NS.DB,
		Command:      command,
		WriteConcern: i.WriteConcern,
		Session:      i.Session,
	}, nil
}

func (i *Insert) encode(desc description.SelectedServer) error {
	batches, err := i.split(int(desc.MaxBatchCount), int(desc.MaxDocumentSize))
	if err != nil {
		return err
	}

	for _, docs := range batches {
		cmd, err := i.encodeBatch(docs, desc)
		if err != nil {
			return err
		}

		i.batches = append(i.batches, cmd)
	}
	return nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (i *Insert) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Insert {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		i.err = err
		return i
	}

	return i.decode(desc, rdr)
}

func (i *Insert) decode(desc description.SelectedServer, rdr bson.Reader) *Insert {
	i.err = bsoncodec.Unmarshal(rdr, &i.result)
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
	if i.batches == nil {
		err := i.encode(desc)
		if err != nil {
			return res, err
		}
	}

	// hold onto txnNumber, reset it when loop exits to ensure reuse of same
	// transaction number if retry is needed
	var txnNumber int64
	if i.Session != nil && i.Session.RetryWrite {
		txnNumber = i.Session.TxnNumber
	}
	for j, cmd := range i.batches {
		rdr, err := cmd.RoundTrip(ctx, desc, rw)
		if err != nil {
			if i.Session != nil && i.Session.RetryWrite {
				i.Session.TxnNumber = txnNumber + int64(j)
			}
			return res, err
		}

		r, err := i.decode(desc, rdr).Result()
		if err != nil {
			return res, err
		}

		res.WriteErrors = append(res.WriteErrors, r.WriteErrors...)

		if r.WriteConcernError != nil {
			res.WriteConcernError = r.WriteConcernError
			if i.Session != nil && i.Session.RetryWrite {
				i.Session.TxnNumber = txnNumber
				return res, nil // report writeconcernerror for retry
			}
		}

		res.N += r.N

		if !i.continueOnError && len(res.WriteErrors) > 0 {
			return res, nil
		}

		// Increment txnNumber for each batch
		if i.Session != nil && i.Session.RetryWrite {
			i.Session.IncrementTxnNumber()
			i.batches = i.batches[1:] // if batch encoded successfully, remove it from the slice
		}
	}

	if i.Session != nil && i.Session.RetryWrite {
		// if retryable write succeeded, transaction number will be incremented one extra time,
		// so we decrement it here
		i.Session.TxnNumber--
	}

	return res, nil
}
