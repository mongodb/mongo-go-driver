// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// bulkWrite performs a bulkwrite operation
type clientBulkWrite struct {
	models                   []interface{}
	errorsOnly               bool
	ordered                  *bool
	bypassDocumentValidation *bool
	comment                  interface{}
	let                      interface{}

	session      *session.Client
	client       *Client
	selector     description.ServerSelector
	writeConcern *writeconcern.WriteConcern

	result *ClientBulkWriteResult
}

func (bw *clientBulkWrite) execute(ctx context.Context) error {
	batches := &driver.Batches{
		Ordered: bw.ordered,
	}
	op := operation.NewCommandFn(bw.command).Batches(batches).
		Session(bw.session).WriteConcern(bw.writeConcern).CommandMonitor(bw.client.monitor).
		ServerSelector(bw.selector).ClusterClock(bw.client.clock).
		Database("admin").
		Deployment(bw.client.deployment).Crypt(bw.client.cryptFLE).
		ServerAPI(bw.client.serverAPI).Timeout(bw.client.timeout).
		Logger(bw.client.logger).Authenticator(bw.client.authenticator)
	err := op.Execute(ctx)
	bw.result = newClientBulkWriteResult(op.Result())
	return err
}

func (bw *clientBulkWrite) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "bulkWrite", 1)
	nsMap := make(map[string]int)
	var nsList []string
	getNsIndex := func(namespace string) int {
		if v, ok := nsMap[namespace]; ok {
			return v
		} else {
			nsIdx := len(nsList)
			nsMap[namespace] = nsIdx
			nsList = append(nsList, namespace)
			return nsIdx
		}
	}
	var err error
	var idx int32
	idx, dst = bsoncore.AppendArrayElementStart(dst, "ops")
	for i, v := range bw.models {
		var doc bsoncore.Document
		var nsIdx int
		switch model := v.(type) {
		case ClientInsertOneModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientInsertDoc(int32(nsIdx), model.Document, bw.client.bsonOpts, bw.client.registry)
			if err != nil {
				break
			}
			doc, _, err = ensureID(doc, primitive.NilObjectID, bw.client.bsonOpts, bw.client.registry)
		case ClientUpdateOneModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientUpdateDoc(
				int32(nsIdx),
				model.Filter,
				model.Update,
				model.Hint,
				model.ArrayFilters,
				model.Collation,
				model.Upsert,
				false,
				true,
				bw.client.bsonOpts,
				bw.client.registry)
		case ClientUpdateManyModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientUpdateDoc(
				int32(nsIdx),
				model.Filter,
				model.Update,
				model.Hint,
				model.ArrayFilters,
				model.Collation,
				model.Upsert,
				true,
				true,
				bw.client.bsonOpts,
				bw.client.registry)
		case ClientReplaceOneModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientUpdateDoc(
				int32(nsIdx),
				model.Filter,
				model.Replacement,
				model.Hint,
				nil,
				model.Collation,
				model.Upsert,
				false,
				false,
				bw.client.bsonOpts,
				bw.client.registry)
		case ClientDeleteOneModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientDeleteDoc(
				int32(nsIdx),
				model.Filter,
				model.Collation,
				model.Hint,
				true,
				bw.client.bsonOpts,
				bw.client.registry)
		case ClientDeleteManyModel:
			nsIdx = getNsIndex(model.Namespace)
			doc, err = createClientDeleteDoc(
				int32(nsIdx),
				model.Filter,
				model.Collation,
				model.Hint,
				false,
				bw.client.bsonOpts,
				bw.client.registry)
		}
		if err != nil {
			return nil, err
		}
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
	}
	dst, err = bsoncore.AppendArrayEnd(dst, idx)
	if err != nil {
		return nil, err
	}

	idx, dst = bsoncore.AppendArrayElementStart(dst, "nsInfo")
	for i, v := range nsList {
		doc, err := marshal(struct {
			Namespace string `bson:"ns"`
		}{v}, bw.client.bsonOpts, bw.client.registry)
		if err != nil {
			return nil, err
		}
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
	}
	dst, err = bsoncore.AppendArrayEnd(dst, idx)
	if err != nil {
		return nil, err
	}

	if bw.errorsOnly {
		dst = bsoncore.AppendBooleanElement(dst, "errorsOnly", bw.errorsOnly)
	}
	if bw.bypassDocumentValidation != nil && (desc.WireVersion != nil && desc.WireVersion.Includes(4)) {
		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *bw.bypassDocumentValidation)
	}
	if bw.comment != nil {
		comment, err := marshalValue(bw.comment, bw.client.bsonOpts, bw.client.registry)
		if err != nil {
			return nil, err
		}
		dst = bsoncore.AppendValueElement(dst, "comment", comment)
	}
	if bw.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *bw.ordered)
	}
	if bw.let != nil {
		let, err := marshal(bw.let, bw.client.bsonOpts, bw.client.registry)
		if err != nil {
			return nil, err
		}
		dst = bsoncore.AppendDocumentElement(dst, "let", let)
	}
	return dst, nil
}

func createClientInsertDoc(
	namespace int32,
	document interface{},
	bsonOpts *options.BSONOptions,
	registry *bsoncodec.Registry,
) (bsoncore.Document, error) {
	uidx, insertDoc := bsoncore.AppendDocumentStart(nil)

	insertDoc = bsoncore.AppendInt32Element(insertDoc, "update", namespace)
	f, err := marshal(document, bsonOpts, registry)
	if err != nil {
		return nil, err
	}
	insertDoc = bsoncore.AppendDocumentElement(insertDoc, "document", f)

	return bsoncore.AppendDocumentEnd(insertDoc, uidx)
}

func createClientUpdateDoc(
	namespace int32,
	filter interface{},
	update interface{},
	hint interface{},
	arrayFilters *options.ArrayFilters,
	collation *options.Collation,
	upsert *bool,
	multi bool,
	checkDollarKey bool,
	bsonOpts *options.BSONOptions,
	registry *bsoncodec.Registry,
) (bsoncore.Document, error) {
	uidx, doc := bsoncore.AppendDocumentStart(nil)

	doc = bsoncore.AppendInt32Element(doc, "update", namespace)

	f, err := marshal(filter, bsonOpts, registry)
	if err != nil {
		return nil, err
	}
	doc = bsoncore.AppendDocumentElement(doc, "filter", f)

	u, err := marshalUpdateValue(update, bsonOpts, registry, checkDollarKey)
	if err != nil {
		return nil, err
	}
	doc = bsoncore.AppendValueElement(doc, "updateMods", u)

	if multi {
		doc = bsoncore.AppendBooleanElement(doc, "multi", multi)
	}

	if arrayFilters != nil {
		reg := registry
		if arrayFilters.Registry != nil {
			reg = arrayFilters.Registry
		}
		arr, err := marshalValue(arrayFilters.Filters, bsonOpts, reg)
		if err != nil {
			return nil, err
		}
		doc = bsoncore.AppendArrayElement(doc, "arrayFilters", arr.Data)
	}

	if collation != nil {
		doc = bsoncore.AppendDocumentElement(doc, "collation", bsoncore.Document(collation.ToDocument()))
	}

	if upsert != nil {
		doc = bsoncore.AppendBooleanElement(doc, "upsert", *upsert)
	}

	if hint != nil {
		if isUnorderedMap(hint) {
			return nil, ErrMapForOrderedArgument{"hint"}
		}
		hintVal, err := marshalValue(hint, bsonOpts, registry)
		if err != nil {
			return nil, err
		}
		doc = bsoncore.AppendValueElement(doc, "hint", hintVal)
	}

	return bsoncore.AppendDocumentEnd(doc, uidx)
}

func createClientDeleteDoc(
	namespace int32,
	filter interface{},
	collation *options.Collation,
	hint interface{},
	multi bool,
	bsonOpts *options.BSONOptions,
	registry *bsoncodec.Registry,
) (bsoncore.Document, error) {
	didx, doc := bsoncore.AppendDocumentStart(nil)

	doc = bsoncore.AppendInt32Element(doc, "delete", namespace)

	f, err := marshal(filter, bsonOpts, registry)
	if err != nil {
		return nil, err
	}
	doc = bsoncore.AppendDocumentElement(doc, "filter", f)

	if multi {
		doc = bsoncore.AppendBooleanElement(doc, "multi", multi)
	}

	if collation != nil {
		doc = bsoncore.AppendDocumentElement(doc, "collation", collation.ToDocument())
	}
	if hint != nil {
		if isUnorderedMap(hint) {
			return nil, ErrMapForOrderedArgument{"hint"}
		}
		hintVal, err := marshalValue(hint, bsonOpts, registry)
		if err != nil {
			return nil, err
		}
		doc = bsoncore.AppendValueElement(doc, "hint", hintVal)
	}
	return bsoncore.AppendDocumentEnd(doc, didx)
}
