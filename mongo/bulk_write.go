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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

type bulkWriteBatch struct {
	models   []WriteModel
	canRetry bool
	indexes  []int
}

// bulkWrite performs a bulkwrite operation
type bulkWrite struct {
	comment                  interface{}
	ordered                  *bool
	bypassDocumentValidation *bool
	models                   []WriteModel
	session                  *session.Client
	collection               *Collection
	selector                 description.ServerSelector
	writeConcern             *writeconcern.WriteConcern
	result                   BulkWriteResult
	let                      interface{}
}

func (bw *bulkWrite) execute(ctx context.Context) error {
	ordered := true
	if bw.ordered != nil {
		ordered = *bw.ordered
	}

	batches := createBatches(bw.models, ordered)
	bw.result = BulkWriteResult{
		UpsertedIDs: make(map[int64]interface{}),
	}

	bwErr := BulkWriteException{
		WriteErrors: make([]BulkWriteError, 0),
	}

	var lastErr error
	continueOnError := !ordered
	for _, batch := range batches {
		if len(batch.models) == 0 {
			continue
		}

		batchRes, batchErr, err := bw.runBatch(ctx, batch)

		bw.mergeResults(batchRes)

		bwErr.WriteConcernError = batchErr.WriteConcernError
		bwErr.Labels = append(bwErr.Labels, batchErr.Labels...)

		bwErr.WriteErrors = append(bwErr.WriteErrors, batchErr.WriteErrors...)

		commandErrorOccurred := err != nil && !errors.Is(err, driver.ErrUnacknowledgedWrite)
		writeErrorOccurred := len(batchErr.WriteErrors) > 0 || batchErr.WriteConcernError != nil
		if !continueOnError && (commandErrorOccurred || writeErrorOccurred) {
			if err != nil {
				return err
			}

			return bwErr
		}

		if err != nil {
			lastErr = err
		}
	}

	bw.result.MatchedCount -= bw.result.UpsertedCount

	rr, err := processWriteError(lastErr)
	if err != nil {
		return err
	}

	bw.result.Acknowledged = rr.isAcknowledged()

	if len(bwErr.WriteErrors) > 0 || bwErr.WriteConcernError != nil {
		return bwErr
	}
	return nil
}

func (bw *bulkWrite) runBatch(ctx context.Context, batch bulkWriteBatch) (BulkWriteResult, BulkWriteException, error) {
	batchRes := BulkWriteResult{
		UpsertedIDs: make(map[int64]interface{}),
	}
	batchErr := BulkWriteException{}

	var writeErrors []driver.WriteError
	switch batch.models[0].(type) {
	case *InsertOneModel:
		res, err := bw.runInsert(ctx, batch)
		if err != nil {
			var writeErr driver.WriteCommandError
			if !errors.As(err, &writeErr) {
				return BulkWriteResult{}, batchErr, err
			}
			writeErrors = writeErr.WriteErrors
			batchErr.Labels = writeErr.Labels
			batchErr.WriteConcernError = convertDriverWriteConcernError(writeErr.WriteConcernError)
		}
		batchRes.InsertedCount = res.N
	case *DeleteOneModel, *DeleteManyModel:
		res, err := bw.runDelete(ctx, batch)
		if err != nil {
			var writeErr driver.WriteCommandError
			if !errors.As(err, &writeErr) {
				return BulkWriteResult{}, batchErr, err
			}
			writeErrors = writeErr.WriteErrors
			batchErr.Labels = writeErr.Labels
			batchErr.WriteConcernError = convertDriverWriteConcernError(writeErr.WriteConcernError)
		}
		batchRes.DeletedCount = res.N
	case *ReplaceOneModel, *UpdateOneModel, *UpdateManyModel:
		res, err := bw.runUpdate(ctx, batch)
		if err != nil {
			var writeErr driver.WriteCommandError
			if !errors.As(err, &writeErr) {
				return BulkWriteResult{}, batchErr, err
			}
			writeErrors = writeErr.WriteErrors
			batchErr.Labels = writeErr.Labels
			batchErr.WriteConcernError = convertDriverWriteConcernError(writeErr.WriteConcernError)
		}
		batchRes.MatchedCount = res.N
		batchRes.ModifiedCount = res.NModified
		batchRes.UpsertedCount = int64(len(res.Upserted))
		for _, upsert := range res.Upserted {
			batchRes.UpsertedIDs[int64(batch.indexes[upsert.Index])] = upsert.ID
		}
	}

	batchErr.WriteErrors = make([]BulkWriteError, 0, len(writeErrors))
	convWriteErrors := writeErrorsFromDriverWriteErrors(writeErrors)
	for _, we := range convWriteErrors {
		request := batch.models[we.Index]
		we.Index = batch.indexes[we.Index]
		batchErr.WriteErrors = append(batchErr.WriteErrors, BulkWriteError{
			WriteError: we,
			Request:    request,
		})
	}
	return batchRes, batchErr, nil
}

func (bw *bulkWrite) runInsert(ctx context.Context, batch bulkWriteBatch) (operation.InsertResult, error) {
	docs := make([]bsoncore.Document, len(batch.models))
	for i, model := range batch.models {
		converted := model.(*InsertOneModel)
		doc, err := marshal(converted.Document, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return operation.InsertResult{}, err
		}
		doc, _, err = ensureID(doc, bson.NilObjectID, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return operation.InsertResult{}, err
		}

		docs[i] = doc
	}

	op := operation.NewInsert(docs...).
		Session(bw.session).WriteConcern(bw.writeConcern).CommandMonitor(bw.collection.client.monitor).
		ServerSelector(bw.selector).ClusterClock(bw.collection.client.clock).
		Database(bw.collection.db.name).Collection(bw.collection.name).
		Deployment(bw.collection.client.deployment).Crypt(bw.collection.client.cryptFLE).
		ServerAPI(bw.collection.client.serverAPI).Timeout(bw.collection.client.timeout).
		Logger(bw.collection.client.logger).Authenticator(bw.collection.client.authenticator)
	if bw.comment != nil {
		comment, err := marshalValue(bw.comment, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return op.Result(), err
		}
		op.Comment(comment)
	}
	if bw.bypassDocumentValidation != nil && *bw.bypassDocumentValidation {
		op = op.BypassDocumentValidation(*bw.bypassDocumentValidation)
	}
	if bw.ordered != nil {
		op = op.Ordered(*bw.ordered)
	}

	retry := driver.RetryNone
	if bw.collection.client.retryWrites && batch.canRetry {
		retry = driver.RetryOncePerCommand
	}
	op = op.Retry(retry)

	err := op.Execute(ctx)

	return op.Result(), err
}

func (bw *bulkWrite) runDelete(ctx context.Context, batch bulkWriteBatch) (operation.DeleteResult, error) {
	docs := make([]bsoncore.Document, len(batch.models))
	var i int
	var hasHint bool

	for _, model := range batch.models {
		var doc bsoncore.Document
		var err error

		switch converted := model.(type) {
		case *DeleteOneModel:
			doc, err = createDeleteDoc(
				converted.Filter,
				converted.Collation,
				converted.Hint,
				true,
				bw.collection.bsonOpts,
				bw.collection.registry)
			hasHint = hasHint || (converted.Hint != nil)
		case *DeleteManyModel:
			doc, err = createDeleteDoc(
				converted.Filter,
				converted.Collation,
				converted.Hint,
				false,
				bw.collection.bsonOpts,
				bw.collection.registry)
			hasHint = hasHint || (converted.Hint != nil)
		}

		if err != nil {
			return operation.DeleteResult{}, err
		}

		docs[i] = doc
		i++
	}

	op := operation.NewDelete(docs...).
		Session(bw.session).WriteConcern(bw.writeConcern).CommandMonitor(bw.collection.client.monitor).
		ServerSelector(bw.selector).ClusterClock(bw.collection.client.clock).
		Database(bw.collection.db.name).Collection(bw.collection.name).
		Deployment(bw.collection.client.deployment).Crypt(bw.collection.client.cryptFLE).Hint(hasHint).
		ServerAPI(bw.collection.client.serverAPI).Timeout(bw.collection.client.timeout).
		Logger(bw.collection.client.logger).Authenticator(bw.collection.client.authenticator)
	if bw.comment != nil {
		comment, err := marshalValue(bw.comment, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return op.Result(), err
		}
		op.Comment(comment)
	}
	if bw.let != nil {
		let, err := marshal(bw.let, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return operation.DeleteResult{}, err
		}
		op = op.Let(let)
	}
	if bw.ordered != nil {
		op = op.Ordered(*bw.ordered)
	}
	retry := driver.RetryNone
	if bw.collection.client.retryWrites && batch.canRetry {
		retry = driver.RetryOncePerCommand
	}
	op = op.Retry(retry)

	err := op.Execute(ctx)

	return op.Result(), err
}

func createDeleteDoc(
	filter interface{},
	collation *options.Collation,
	hint interface{},
	deleteOne bool,
	bsonOpts *options.BSONOptions,
	registry *bson.Registry,
) (bsoncore.Document, error) {
	if filter == nil {
		return nil, fmt.Errorf("delete filter cannot be nil")
	}
	f, err := marshal(filter, bsonOpts, registry)
	if err != nil {
		return nil, err
	}

	var limit int32
	if deleteOne {
		limit = 1
	}
	didx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendDocumentElement(doc, "q", f)
	doc = bsoncore.AppendInt32Element(doc, "limit", limit)
	if collation != nil {
		doc = bsoncore.AppendDocumentElement(doc, "collation", toDocument(collation))
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
	doc, _ = bsoncore.AppendDocumentEnd(doc, didx)

	return doc, nil
}

func (bw *bulkWrite) runUpdate(ctx context.Context, batch bulkWriteBatch) (operation.UpdateResult, error) {
	docs := make([]bsoncore.Document, len(batch.models))
	var hasHint bool
	var hasArrayFilters bool
	for i, model := range batch.models {
		var doc bsoncore.Document
		var err error

		switch converted := model.(type) {
		case *ReplaceOneModel:
			doc, err = updateDoc{
				filter:    converted.Filter,
				update:    converted.Replacement,
				hint:      converted.Hint,
				sort:      converted.Sort,
				collation: converted.Collation,
				upsert:    converted.Upsert,
			}.marshal(bw.collection.bsonOpts, bw.collection.registry)
			hasHint = hasHint || (converted.Hint != nil)
		case *UpdateOneModel:
			doc, err = updateDoc{
				filter:         converted.Filter,
				update:         converted.Update,
				hint:           converted.Hint,
				sort:           converted.Sort,
				arrayFilters:   converted.ArrayFilters,
				collation:      converted.Collation,
				upsert:         converted.Upsert,
				checkDollarKey: true,
			}.marshal(bw.collection.bsonOpts, bw.collection.registry)
			hasHint = hasHint || (converted.Hint != nil)
			hasArrayFilters = hasArrayFilters || (converted.ArrayFilters != nil)
		case *UpdateManyModel:
			doc, err = updateDoc{
				filter:         converted.Filter,
				update:         converted.Update,
				hint:           converted.Hint,
				arrayFilters:   converted.ArrayFilters,
				collation:      converted.Collation,
				upsert:         converted.Upsert,
				multi:          true,
				checkDollarKey: true,
			}.marshal(bw.collection.bsonOpts, bw.collection.registry)
			hasHint = hasHint || (converted.Hint != nil)
			hasArrayFilters = hasArrayFilters || (converted.ArrayFilters != nil)
		}
		if err != nil {
			return operation.UpdateResult{}, err
		}

		docs[i] = doc
	}

	op := operation.NewUpdate(docs...).
		Session(bw.session).WriteConcern(bw.writeConcern).CommandMonitor(bw.collection.client.monitor).
		ServerSelector(bw.selector).ClusterClock(bw.collection.client.clock).
		Database(bw.collection.db.name).Collection(bw.collection.name).
		Deployment(bw.collection.client.deployment).Crypt(bw.collection.client.cryptFLE).Hint(hasHint).
		ArrayFilters(hasArrayFilters).ServerAPI(bw.collection.client.serverAPI).
		Timeout(bw.collection.client.timeout).Logger(bw.collection.client.logger).
		Authenticator(bw.collection.client.authenticator)
	if bw.comment != nil {
		comment, err := marshalValue(bw.comment, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return op.Result(), err
		}
		op.Comment(comment)
	}
	if bw.let != nil {
		let, err := marshal(bw.let, bw.collection.bsonOpts, bw.collection.registry)
		if err != nil {
			return operation.UpdateResult{}, err
		}
		op = op.Let(let)
	}
	if bw.ordered != nil {
		op = op.Ordered(*bw.ordered)
	}
	if bw.bypassDocumentValidation != nil && *bw.bypassDocumentValidation {
		op = op.BypassDocumentValidation(*bw.bypassDocumentValidation)
	}
	retry := driver.RetryNone
	if bw.collection.client.retryWrites && batch.canRetry {
		retry = driver.RetryOncePerCommand
	}
	op = op.Retry(retry)

	err := op.Execute(ctx)

	return op.Result(), err
}

type updateDoc struct {
	filter         interface{}
	update         interface{}
	hint           interface{}
	sort           interface{}
	arrayFilters   []interface{}
	collation      *options.Collation
	upsert         *bool
	multi          bool
	checkDollarKey bool
}

func (doc updateDoc) marshal(bsonOpts *options.BSONOptions, registry *bson.Registry) (bsoncore.Document, error) {
	if doc.filter == nil {
		return nil, fmt.Errorf("update filter cannot be nil")
	}
	f, err := marshal(doc.filter, bsonOpts, registry)
	if err != nil {
		return nil, err
	}

	uidx, updateDoc := bsoncore.AppendDocumentStart(nil)
	updateDoc = bsoncore.AppendDocumentElement(updateDoc, "q", f)

	u, err := marshalUpdateValue(doc.update, bsonOpts, registry, doc.checkDollarKey)
	if err != nil {
		return nil, err
	}

	updateDoc = bsoncore.AppendValueElement(updateDoc, "u", u)

	if doc.multi {
		updateDoc = bsoncore.AppendBooleanElement(updateDoc, "multi", doc.multi)
	}
	if doc.sort != nil {
		if isUnorderedMap(doc.sort) {
			return nil, ErrMapForOrderedArgument{"sort"}
		}
		s, err := marshal(doc.sort, bsonOpts, registry)
		if err != nil {
			return nil, err
		}
		updateDoc = bsoncore.AppendDocumentElement(updateDoc, "sort", s)
	}

	if doc.arrayFilters != nil {
		reg := registry
		arr, err := marshalValue(doc.arrayFilters, bsonOpts, reg)
		if err != nil {
			return nil, err
		}
		updateDoc = bsoncore.AppendArrayElement(updateDoc, "arrayFilters", arr.Data)
	}

	if doc.collation != nil {
		updateDoc = bsoncore.AppendDocumentElement(updateDoc, "collation", bsoncore.Document(toDocument(doc.collation)))
	}

	if doc.upsert != nil {
		updateDoc = bsoncore.AppendBooleanElement(updateDoc, "upsert", *doc.upsert)
	}

	if doc.hint != nil {
		if isUnorderedMap(doc.hint) {
			return nil, ErrMapForOrderedArgument{"hint"}
		}
		hintVal, err := marshalValue(doc.hint, bsonOpts, registry)
		if err != nil {
			return nil, err
		}
		updateDoc = bsoncore.AppendValueElement(updateDoc, "hint", hintVal)
	}

	updateDoc, _ = bsoncore.AppendDocumentEnd(updateDoc, uidx)
	return updateDoc, nil
}

func createBatches(models []WriteModel, ordered bool) []bulkWriteBatch {
	if ordered {
		return createOrderedBatches(models)
	}

	batches := make([]bulkWriteBatch, 5)
	batches[insertCommand].canRetry = true
	batches[deleteOneCommand].canRetry = true
	batches[updateOneCommand].canRetry = true

	// TODO(GODRIVER-1157): fix batching once operation retryability is fixed
	for i, model := range models {
		switch model.(type) {
		case *InsertOneModel:
			batches[insertCommand].models = append(batches[insertCommand].models, model)
			batches[insertCommand].indexes = append(batches[insertCommand].indexes, i)
		case *DeleteOneModel:
			batches[deleteOneCommand].models = append(batches[deleteOneCommand].models, model)
			batches[deleteOneCommand].indexes = append(batches[deleteOneCommand].indexes, i)
		case *DeleteManyModel:
			batches[deleteManyCommand].models = append(batches[deleteManyCommand].models, model)
			batches[deleteManyCommand].indexes = append(batches[deleteManyCommand].indexes, i)
		case *ReplaceOneModel, *UpdateOneModel:
			batches[updateOneCommand].models = append(batches[updateOneCommand].models, model)
			batches[updateOneCommand].indexes = append(batches[updateOneCommand].indexes, i)
		case *UpdateManyModel:
			batches[updateManyCommand].models = append(batches[updateManyCommand].models, model)
			batches[updateManyCommand].indexes = append(batches[updateManyCommand].indexes, i)
		}
	}

	return batches
}

func createOrderedBatches(models []WriteModel) []bulkWriteBatch {
	var batches []bulkWriteBatch
	var prevKind writeCommandKind = -1
	i := -1 // batch index

	for ind, model := range models {
		var createNewBatch bool
		var canRetry bool
		var newKind writeCommandKind

		// TODO(GODRIVER-1157): fix batching once operation retryability is fixed
		switch model.(type) {
		case *InsertOneModel:
			createNewBatch = prevKind != insertCommand
			canRetry = true
			newKind = insertCommand
		case *DeleteOneModel:
			createNewBatch = prevKind != deleteOneCommand
			canRetry = true
			newKind = deleteOneCommand
		case *DeleteManyModel:
			createNewBatch = prevKind != deleteManyCommand
			newKind = deleteManyCommand
		case *ReplaceOneModel, *UpdateOneModel:
			createNewBatch = prevKind != updateOneCommand
			canRetry = true
			newKind = updateOneCommand
		case *UpdateManyModel:
			createNewBatch = prevKind != updateManyCommand
			newKind = updateManyCommand
		}

		if createNewBatch {
			batches = append(batches, bulkWriteBatch{
				models:   []WriteModel{model},
				canRetry: canRetry,
				indexes:  []int{ind},
			})
			i++
		} else {
			batches[i].models = append(batches[i].models, model)
			if !canRetry {
				batches[i].canRetry = false // don't make it true if it was already false
			}
			batches[i].indexes = append(batches[i].indexes, ind)
		}

		prevKind = newKind
	}

	return batches
}

func (bw *bulkWrite) mergeResults(newResult BulkWriteResult) {
	bw.result.InsertedCount += newResult.InsertedCount
	bw.result.MatchedCount += newResult.MatchedCount
	bw.result.ModifiedCount += newResult.ModifiedCount
	bw.result.DeletedCount += newResult.DeletedCount
	bw.result.UpsertedCount += newResult.UpsertedCount

	for index, upsertID := range newResult.UpsertedIDs {
		bw.result.UpsertedIDs[index] = upsertID
	}
}

// WriteCommandKind is the type of command represented by a Write
type writeCommandKind int8

// These constants represent the valid types of write commands.
const (
	insertCommand writeCommandKind = iota
	updateOneCommand
	updateManyCommand
	deleteOneCommand
	deleteManyCommand
)
