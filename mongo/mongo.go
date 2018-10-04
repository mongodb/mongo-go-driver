// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
)

// Dialer is used to make network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// BSONAppender is an interface implemented by types that can marshal a
// provided type into BSON bytes and append those bytes to the provided []byte.
// The AppendBSON can return a non-nil error and non-nil []byte. The AppendBSON
// method may also write incomplete BSON to the []byte.
type BSONAppender interface {
	AppendBSON([]byte, interface{}) ([]byte, error)
}

// BSONAppenderFunc is an adapter function that allows any function that
// satisfies the AppendBSON method signature to be used where a BSONAppender is
// used.
type BSONAppenderFunc func([]byte, interface{}) ([]byte, error)

// AppendBSON implements the BSONAppender interface
func (baf BSONAppenderFunc) AppendBSON(dst []byte, val interface{}) ([]byte, error) {
	return baf(dst, val)
}

// MarshalError is returned when attempting to transform a value into a document
// results in an error.
type MarshalError struct {
	Value interface{}
	Err   error
}

// Error implements the error interface.
func (me MarshalError) Error() string {
	return fmt.Sprintf("cannot transform type %s to a *bson.Document", reflect.TypeOf(me.Value))
}

func transformDocument(registry *bsoncodec.Registry, val interface{}) (*bson.Document, error) {
	if registry == nil {
		registry = bson.NewRegistryBuilder().Build()
	}
	if val == nil {
		return bson.NewDocument(), nil
	}
	if doc, ok := val.(*bson.Document); ok {
		return doc.Copy(), nil
	}
	if bs, ok := val.([]byte); ok {
		// Slight optimization so we'll just use MarshalBSON and not go through the codec machinery.
		val = bson.Reader(bs)
	}

	// TODO(skriptble): Use a pool of these instead.
	buf := make([]byte, 0, 256)
	b, err := bson.MarshalAppendWithRegistry(registry, buf[:0], val)
	if err != nil {
		return nil, MarshalError{Value: val, Err: err}
	}
	return bson.ReadDocument(b)
}

func ensureID(d *bson.Document) (interface{}, error) {
	var id interface{}

	elem, err := d.LookupElementErr("_id")
	switch {
	case err == bson.ErrElementNotFound:
		oid := objectid.New()
		d.Append(bson.EC.ObjectID("_id", oid))
		id = oid
	case err != nil:
		return nil, err
	default:
		id = elem
	}
	return id, nil
}

func ensureDollarKey(doc *bson.Document) error {
	if elem, ok := doc.ElementAtOK(0); !ok || !strings.HasPrefix(elem.Key(), "$") {
		return errors.New("update document must contain key beginning with '$'")
	}
	return nil
}

func transformAggregatePipeline(registry *bsoncodec.Registry, pipeline interface{}) (*bson.Array, error) {
	var pipelineArr *bson.Array
	switch t := pipeline.(type) {
	case *bson.Array:
		pipelineArr = t
	case []*bson.Document:
		pipelineArr = bson.NewArray()

		for _, doc := range t {
			pipelineArr.Append(bson.VC.Document(doc))
		}
	case []interface{}:
		pipelineArr = bson.NewArray()

		for _, val := range t {
			doc, err := transformDocument(registry, val)
			if err != nil {
				return nil, err
			}

			pipelineArr.Append(bson.VC.Document(doc))
		}
	default:
		p, err := transformDocument(registry, pipeline)
		if err != nil {
			return nil, err
		}

		pipelineArr = bson.ArrayFromDocument(p)
	}

	return pipelineArr, nil
}

// Build the aggregation pipeline for the CountDocument command.
func countDocumentsAggregatePipeline(registry *bsoncodec.Registry, filter interface{}, opts ...countopt.Count) (*bson.Array, error) {
	pipeline := bson.NewArray()
	filterDoc, err := transformDocument(registry, filter)

	if err != nil {
		return nil, err
	}
	pipeline.Append(bson.VC.Document(bson.NewDocument(bson.EC.SubDocument("$match", filterDoc))))
	for _, opt := range opts {
		switch t := opt.(type) {
		case countopt.OptSkip:
			skip := int64(t)
			pipeline.Append(bson.VC.Document(bson.NewDocument(bson.EC.Int64("$skip", skip))))
		case countopt.OptLimit:
			limit := int64(t)
			pipeline.Append(bson.VC.Document(bson.NewDocument(bson.EC.Int64("$limit", limit))))
		}
	}
	pipeline.Append(bson.VC.Document(bson.NewDocument(
		bson.EC.SubDocument("$group", bson.NewDocument(
			bson.EC.Null("_id"),
			bson.EC.SubDocument("n", bson.NewDocument(
				bson.EC.Int32("$sum", 1)),
			)),
		)),
	))

	return pipeline, nil
}
