// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
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
	session                  *session.Client
	client                   *Client
	selector                 description.ServerSelector
	writeConcern             *writeconcern.WriteConcern

	cursorHandlers []func(*cursorInfo, bson.Raw) error
	insIDMap       map[int]interface{}

	result             ClientBulkWriteResult
	writeConcernErrors []WriteConcernError
	writeErrors        map[int]WriteError
}

func (bw *clientBulkWrite) execute(ctx context.Context) error {
	if len(bw.models) == 0 {
		return errors.New("empty write models")
	}
	bw.writeErrors = make(map[int]WriteError)
	batches, retry, err := bw.processModels()
	if err != nil {
		return err
	}
	err = driver.Operation{
		CommandFn:         bw.newCommand(),
		ProcessResponseFn: bw.ProcessResponse,
		Client:            bw.session,
		Clock:             bw.client.clock,
		RetryMode:         retry,
		Type:              driver.Write,
		Batches:           batches,
		CommandMonitor:    bw.client.monitor,
		Database:          "admin",
		Deployment:        bw.client.deployment,
		Selector:          bw.selector,
		WriteConcern:      bw.writeConcern,
		Crypt:             bw.client.cryptFLE,
		ServerAPI:         bw.client.serverAPI,
		Timeout:           bw.client.timeout,
		Logger:            bw.client.logger,
		Authenticator:     bw.client.authenticator,
		Name:              "bulkWrite",
	}.Execute(ctx)
	if err != nil && errors.Is(err, driver.ErrUnacknowledgedWrite) {
		return nil
	}
	fmt.Println("exec", len(bw.writeErrors), err)
	return err
}

type cursorInfo struct {
	Ok        bool
	Idx       int32
	Code      *int32
	Errmsg    *string
	ErrInfo   bson.Raw
	N         int32
	NModified *int32
	Upserted  *struct {
		ID interface{} `bson:"_id"`
	}
}

func (cur *cursorInfo) extractError() *WriteError {
	if cur.Ok {
		return nil
	}
	err := &WriteError{
		Index:   int(cur.Idx),
		Details: cur.ErrInfo,
	}
	if cur.Code != nil {
		err.Code = int(*cur.Code)
	}
	if cur.Errmsg != nil {
		err.Message = *cur.Errmsg
	}
	return err
}

func (bw *clientBulkWrite) ProcessResponse(ctx context.Context, info driver.ResponseInfo) error {
	fmt.Println("ProcessResponse", info.Error)
	var writeCmdErr driver.WriteCommandError
	if errors.As(info.Error, &writeCmdErr) && writeCmdErr.WriteConcernError != nil {
		wce := convertDriverWriteConcernError(writeCmdErr.WriteConcernError)
		if wce != nil {
			bw.writeConcernErrors = append(bw.writeConcernErrors, *wce)
		}
	}
	// closeImplicitSession(sess)
	if len(info.ServerResponse) == 0 {
		return nil
	}
	var res struct {
		Ok        bool
		NDeleted  int32
		NInserted int32
		NMatched  int32
		NModified int32
		NUpserted int32
		NErrors   int32
		Code      int32
		Errmsg    string
	}
	err := bson.Unmarshal(info.ServerResponse, &res)
	if err != nil {
		return err
	}
	bw.result.DeletedCount += int64(res.NDeleted)
	bw.result.InsertedCount += int64(res.NInserted)
	bw.result.MatchedCount += int64(res.NMatched)
	bw.result.ModifiedCount += int64(res.NModified)
	bw.result.UpsertedCount += int64(res.NUpserted)

	var cursorRes driver.CursorResponse
	cursorRes, err = driver.NewCursorResponse(info)
	if err != nil {
		return err
	}
	var bCursor *driver.BatchCursor
	bCursor, err = driver.NewBatchCursor(cursorRes, bw.session, bw.client.clock,
		driver.CursorOptions{
			CommandMonitor:        bw.client.monitor,
			Crypt:                 bw.client.cryptFLE,
			ServerAPI:             bw.client.serverAPI,
			MarshalValueEncoderFn: newEncoderFn(bw.client.bsonOpts, bw.client.registry),
		},
	)
	if err != nil {
		return err
	}
	var cursor *Cursor
	cursor, err = newCursor(bCursor, bw.client.bsonOpts, bw.client.registry)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var cur cursorInfo
		cursor.Decode(&cur)
		if int(cur.Idx) >= len(bw.cursorHandlers) {
			continue
		}
		if err := bw.cursorHandlers[int(cur.Idx)](&cur, cursor.Current); err != nil {
			fmt.Println("ProcessResponse cursorHandlers", err)
			return err
		}
	}
	err = cursor.Err()
	if err != nil {
		return err
	}
	fmt.Println("ProcessResponse toplevelerror", res.Ok, res.NErrors, res.Code, res.Errmsg)
	// if !res.Ok || res.NErrors > 0 {
	// 	exception := bw.formException()
	// 	exception.TopLevelError = &WriteError{
	// 		Code:    int(res.Code),
	// 		Message: res.Errmsg,
	// 		Raw:     bson.Raw(info.ServerResponse),
	// 	}
	// 	return exception
	// }
	return nil
}

func (bw *clientBulkWrite) processModels() ([]driver.Batches, *driver.RetryMode, error) {
	nsMap := make(map[string]int)
	var nsList []bsoncore.Document
	getNsIndex := func(namespace string) int {
		if v, ok := nsMap[namespace]; ok {
			return v
		}
		nsIdx := len(nsList)
		nsMap[namespace] = nsIdx
		idx, doc := bsoncore.AppendDocumentStart(nil)
		doc = bsoncore.AppendStringElement(doc, "ns", namespace)
		doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
		nsList = append(nsList, doc)
		return nsIdx
	}

	bw.cursorHandlers = make([]func(*cursorInfo, bson.Raw) error, len(bw.models))
	bw.insIDMap = make(map[int]interface{})
	canRetry := true
	docs := make([]bsoncore.Document, len(bw.models))
	for i, v := range bw.models {
		var doc bsoncore.Document
		var err error
		switch model := v.(type) {
		case *ClientInsertOneModel:
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendInsertResult
			var id interface{}
			id, doc, err = createClientInsertDoc(int32(nsIdx), model.Document, bw.client.bsonOpts, bw.client.registry)
			if err != nil {
				break
			}
			bw.insIDMap[i] = id
		case *ClientUpdateOneModel:
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendUpdateResult
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
			canRetry = false
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendUpdateResult
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
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendUpdateResult
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
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendDeleteResult
			doc, err = createClientDeleteDoc(
				int32(nsIdx),
				model.Filter,
				model.Collation,
				model.Hint,
				false,
				bw.client.bsonOpts,
				bw.client.registry)
		case *ClientDeleteManyModel:
			canRetry = false
			nsIdx := getNsIndex(model.Namespace)
			bw.cursorHandlers[i] = bw.appendDeleteResult
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
			return nil, nil, err
		}
		docs[i] = doc
	}
	retry := driver.RetryNone
	if bw.client.retryWrites && canRetry {
		retry = driver.RetryOncePerCommand
	}
	ordered := false
	return []driver.Batches{
			{
				Identifier: "ops",
				Documents:  docs,
				Ordered:    bw.ordered,
			},
			{
				Identifier: "nsInfo",
				Documents:  nsList,
				Ordered:    &ordered,
			},
		},
		&retry, nil
}

func (bw *clientBulkWrite) newCommand() func([]byte, description.SelectedServer) ([]byte, error) {
	return func(dst []byte, desc description.SelectedServer) ([]byte, error) {
		dst = bsoncore.AppendInt32Element(dst, "bulkWrite", 1)

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

func (bw *clientBulkWrite) appendDeleteResult(cur *cursorInfo, raw bson.Raw) error {
	if bw.result.DeleteResults == nil {
		bw.result.DeleteResults = make(map[int]ClientDeleteResult)
	}
	bw.result.DeleteResults[int(cur.Idx)] = ClientDeleteResult{int64(cur.N)}
	if err := cur.extractError(); err != nil {
		err.Raw = raw
		bw.writeErrors[int(cur.Idx)] = *err
		if bw.ordered != nil && *bw.ordered {
			return bw.formException()
		}
	}
	return nil
}

func (bw *clientBulkWrite) appendInsertResult(cur *cursorInfo, raw bson.Raw) error {
	if bw.result.InsertResults == nil {
		bw.result.InsertResults = make(map[int]ClientInsertResult)
	}
	bw.result.InsertResults[int(cur.Idx)] = ClientInsertResult{bw.insIDMap[int(cur.Idx)]}
	if err := cur.extractError(); err != nil {
		err.Raw = raw
		bw.writeErrors[int(cur.Idx)] = *err
		if bw.ordered != nil && *bw.ordered {
			return bw.formException()
		}
	}
	return nil
}

func (bw *clientBulkWrite) appendUpdateResult(cur *cursorInfo, raw bson.Raw) error {
	if bw.result.UpdateResults == nil {
		bw.result.UpdateResults = make(map[int]ClientUpdateResult)
	}
	result := ClientUpdateResult{
		MatchedCount: int64(cur.N),
	}
	if cur.NModified != nil {
		result.ModifiedCount = int64(*cur.NModified)
	}
	if cur.Upserted != nil {
		result.UpsertedID = (*cur.Upserted).ID
	}
	bw.result.UpdateResults[int(cur.Idx)] = result
	if err := cur.extractError(); err != nil {
		err.Raw = raw
		bw.writeErrors[int(cur.Idx)] = *err
		if bw.ordered != nil && *bw.ordered {
			return bw.formException()
		}
	}
	return nil
}

func (bw *clientBulkWrite) formException() ClientBulkWriteException {
	return ClientBulkWriteException{
		WriteConcernErrors: bw.writeConcernErrors,
		WriteErrors:        bw.writeErrors,
		PartialResult:      &bw.result,
	}
}
