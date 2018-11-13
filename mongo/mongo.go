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

	"github.com/mongodb/mongo-go-driver/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
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
	return fmt.Sprintf("cannot transform type %s to a *bsonx.Document", reflect.TypeOf(me.Value))
}

// Pipeline is a type that makes creating aggregation pipelines easier. It is a
// helper and is intended for serializing to BSON.
//
// Example usage:
//
// 		mongo.Pipeline{{
// 			{"$group", bson.D{{"_id", "$state"}, {"totalPop", bson.D{"$sum", "$pop"}}}},
// 			{"$match": bson.D{{"totalPop", bson.D{"$gte", 10*1000*1000}}}},
// 		}}
//
type Pipeline []bson.D

func transformDocument(registry *bsoncodec.Registry, val interface{}) (bsonx.Doc, error) {
	if registry == nil {
		registry = bson.NewRegistryBuilder().Build()
	}
	if val == nil {
		return bsonx.Doc{}, nil
	}
	if doc, ok := val.(bsonx.Doc); ok {
		return doc.Copy(), nil
	}
	if bs, ok := val.([]byte); ok {
		// Slight optimization so we'll just use MarshalBSON and not go through the codec machinery.
		val = bson.Raw(bs)
	}

	// TODO(skriptble): Use a pool of these instead.
	buf := make([]byte, 0, 256)
	b, err := bson.MarshalAppendWithRegistry(registry, buf[:0], val)
	if err != nil {
		return nil, MarshalError{Value: val, Err: err}
	}
	return bsonx.ReadDoc(b)
}

func ensureID(d bsonx.Doc) (bsonx.Doc, interface{}) {
	var id interface{}

	elem, err := d.LookupElementErr("_id")
	switch err.(type) {
	case nil:
		id = elem
	default:
		oid := objectid.New()
		d = append(d, bsonx.Elem{"_id", bsonx.ObjectID(oid)})
		id = oid
	}
	return d, id
}

func ensureDollarKey(doc bsonx.Doc) error {
	if len(doc) > 0 && !strings.HasPrefix(doc[0].Key, "$") {
		return errors.New("update document must contain key beginning with '$'")
	}
	return nil
}

func transformAggregatePipeline(registry *bsoncodec.Registry, pipeline interface{}) (bsonx.Arr, error) {
	pipelineArr := bsonx.Arr{}
	switch t := pipeline.(type) {
	case Pipeline:
		for _, d := range t {
			doc, err := transformDocument(registry, d)
			if err != nil {
				return nil, err
			}
			pipelineArr = append(pipelineArr, bsonx.Document(doc))
		}
	case bsonx.Arr:
		pipelineArr = make(bsonx.Arr, len(t))
		copy(pipelineArr, t)
	case []bsonx.Doc:
		pipelineArr = bsonx.Arr{}

		for _, doc := range t {
			pipelineArr = append(pipelineArr, bsonx.Document(doc))
		}
	case []interface{}:
		pipelineArr = bsonx.Arr{}

		for _, val := range t {
			doc, err := transformDocument(registry, val)
			if err != nil {
				return nil, err
			}

			pipelineArr = append(pipelineArr, bsonx.Document(doc))
		}
	default:
		p, err := transformDocument(registry, pipeline)
		if err != nil {
			return nil, err
		}

		for _, elem := range p {
			pipelineArr = append(pipelineArr, elem.Value)
		}
	}

	return pipelineArr, nil
}

// Build the aggregation pipeline for the CountDocument command.
func countDocumentsAggregatePipeline(registry *bsoncodec.Registry, filter interface{}, opts *options.CountOptions) (bsonx.Arr, error) {
	pipeline := bsonx.Arr{}
	filterDoc, err := transformDocument(registry, filter)

	if err != nil {
		return nil, err
	}
	pipeline = append(pipeline, bsonx.Document(bsonx.Doc{{"$match", bsonx.Document(filterDoc)}}))

	if opts != nil {
		if opts.Skip != nil {
			pipeline = append(pipeline, bsonx.Document(bsonx.Doc{{"$skip", bsonx.Int64(*opts.Skip)}}))
		}
		if opts.Limit != nil {
			pipeline = append(pipeline, bsonx.Document(bsonx.Doc{{"$limit", bsonx.Int64(*opts.Limit)}}))
		}
	}

	pipeline = append(pipeline, bsonx.Document(bsonx.Doc{
		{"$group", bsonx.Document(bsonx.Doc{
			{"_id", bsonx.Null()},
			{"n", bsonx.Document(bsonx.Doc{{"$sum", bsonx.Int32(1)}})},
		})},
	},
	))

	return pipeline, nil
}
