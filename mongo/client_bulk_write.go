// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
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

	result ClientBulkWriteResult
}

func (bw *clientBulkWrite) execute(ctx context.Context) error {
	docs := make([]bsoncore.Document, len(bw.models))
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
	resMap := make([]interface{}, len(bw.models))
	insIdMap := make(map[int]interface{})
	for i, v := range bw.models {
		var doc bsoncore.Document
		var err error
		var nsIdx int
		switch model := v.(type) {
		case *ClientInsertOneModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.InsertResults == nil {
				bw.result.InsertResults = make(map[int64]ClientInsertResult)
			}
			resMap[i] = bw.result.InsertResults
			var id interface{}
			id, doc, err = createClientInsertDoc(int32(nsIdx), model.Document, bw.client.bsonOpts, bw.client.registry)
			if err != nil {
				break
			}
			insIdMap[i] = id
		case *ClientUpdateOneModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.UpdateResults == nil {
				bw.result.UpdateResults = make(map[int64]ClientUpdateResult)
			}
			resMap[i] = bw.result.UpdateResults
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
		case *ClientUpdateManyModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.UpdateResults == nil {
				bw.result.UpdateResults = make(map[int64]ClientUpdateResult)
			}
			resMap[i] = bw.result.UpdateResults
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
		case *ClientReplaceOneModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.UpdateResults == nil {
				bw.result.UpdateResults = make(map[int64]ClientUpdateResult)
			}
			resMap[i] = bw.result.UpdateResults
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
		case *ClientDeleteOneModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.DeleteResults == nil {
				bw.result.DeleteResults = make(map[int64]ClientDeleteResult)
			}
			resMap[i] = bw.result.DeleteResults
			doc, err = createClientDeleteDoc(
				int32(nsIdx),
				model.Filter,
				model.Collation,
				model.Hint,
				false,
				bw.client.bsonOpts,
				bw.client.registry)
		case *ClientDeleteManyModel:
			nsIdx = getNsIndex(model.Namespace)
			if bw.result.DeleteResults == nil {
				bw.result.DeleteResults = make(map[int64]ClientDeleteResult)
			}
			resMap[i] = bw.result.DeleteResults
			doc, err = createClientDeleteDoc(
				int32(nsIdx),
				model.Filter,
				model.Collation,
				model.Hint,
				true,
				bw.client.bsonOpts,
				bw.client.registry)
		}
		if err != nil {
			return err
		}
		docs[i] = doc
	}
	batches := &driver.Batches{
		Identifier: "ops",
		Documents:  docs,
		Ordered:    bw.ordered,
	}
	op := operation.NewCommandFn(bw.newCommand(nsList)).Batches(batches).
		Session(bw.session).WriteConcern(bw.writeConcern).CommandMonitor(bw.client.monitor).
		ServerSelector(bw.selector).ClusterClock(bw.client.clock).
		Database("admin").
		Deployment(bw.client.deployment).Crypt(bw.client.cryptFLE).
		ServerAPI(bw.client.serverAPI).Timeout(bw.client.timeout).
		Logger(bw.client.logger).Authenticator(bw.client.authenticator)
	err := op.Execute(ctx)
	if err != nil {
		return err
	}
	var res struct {
		Cursor struct {
			FirstBatch []bson.Raw
		}
		NDeleted  int32
		NInserted int32
		NMatched  int32
		NModified int32
		NUpserted int32
	}
	rawRes := op.Result()
	err = bson.Unmarshal(rawRes, &res)
	if err != nil {
		return err
	}
	bw.result.DeletedCount = int64(res.NDeleted)
	bw.result.InsertedCount = int64(res.NInserted)
	bw.result.MatchedCount = int64(res.NMatched)
	bw.result.ModifiedCount = int64(res.NModified)
	bw.result.UpsertedCount = int64(res.NUpserted)
	for i, cur := range res.Cursor.FirstBatch {
		switch res := resMap[i].(type) {
		case map[int64]ClientDeleteResult:
			if err = appendDeleteResult(cur, res); err != nil {
				return err
			}
		case map[int64]ClientInsertResult:
			if err = appendInsertResult(cur, res, insIdMap); err != nil {
				return err
			}
		case map[int64]ClientUpdateResult:
			if err = appendUpdateResult(cur, res); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bw *clientBulkWrite) newCommand(nsList []string) func([]byte, description.SelectedServer) ([]byte, error) {
	return func(dst []byte, desc description.SelectedServer) ([]byte, error) {
		dst = bsoncore.AppendInt32Element(dst, "bulkWrite", 1)

		var idx int32
		var err error
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

		dst = bsoncore.AppendBooleanElement(dst, "errorsOnly", bw.errorsOnly)
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
}

func createClientInsertDoc(
	namespace int32,
	document interface{},
	bsonOpts *options.BSONOptions,
	registry *bsoncodec.Registry,
) (interface{}, bsoncore.Document, error) {
	uidx, doc := bsoncore.AppendDocumentStart(nil)

	doc = bsoncore.AppendInt32Element(doc, "insert", namespace)
	f, err := marshal(document, bsonOpts, registry)
	if err != nil {
		return nil, nil, err
	}
	var id interface{}
	f, id, err = ensureID(f, primitive.NilObjectID, bsonOpts, registry)
	if err != nil {
		return nil, nil, err
	}
	doc = bsoncore.AppendDocumentElement(doc, "document", f)
	doc, err = bsoncore.AppendDocumentEnd(doc, uidx)
	return id, doc, err
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
	doc = bsoncore.AppendBooleanElement(doc, "multi", multi)

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
	doc = bsoncore.AppendBooleanElement(doc, "multi", multi)

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

func appendDeleteResult(cur bson.Raw, m map[int64]ClientDeleteResult) error {
	var res struct {
		Idx int32
		N   int32
	}
	if err := bson.Unmarshal(cur, &res); err != nil {
		return err
	}
	m[int64(res.Idx)] = ClientDeleteResult{int64(res.N)}
	return nil
}

func appendInsertResult(cur bson.Raw, m map[int64]ClientInsertResult, insIdMap map[int]interface{}) error {
	var res struct {
		Idx int32
	}
	if err := bson.Unmarshal(cur, &res); err != nil {
		return err
	}
	m[int64(res.Idx)] = ClientInsertResult{insIdMap[int(res.Idx)]}
	return nil
}

func appendUpdateResult(cur bson.Raw, m map[int64]ClientUpdateResult) error {
	var res struct {
		Idx       int32
		N         int32
		NModified int32
		Upserted  struct {
			Id interface{} `bson:"_id"`
		}
	}
	if err := bson.Unmarshal(cur, &res); err != nil {
		return err
	}
	m[int64(res.Idx)] = ClientUpdateResult{
		MatchedCount:  int64(res.N),
		ModifiedCount: int64(res.NModified),
		UpsertedID:    res.Upserted.Id,
	}
	return nil
}
