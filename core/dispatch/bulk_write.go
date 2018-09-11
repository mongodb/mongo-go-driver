package dispatch

import (
	"context"

	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// ErrCollation is caused if a collation is given for an invalid server version.
var ErrCollation = errors.New("collation cannot be set for server versions < 3.4")

// ErrArrayFilters is caused if array filters are given for an invalid server version.
var ErrArrayFilters = errors.New("array filters cannot be set for server versions < 3.6")

type bulkWriteBatch struct {
	models   []WriteModel
	canRetry bool
}

// BulkWrite handles the full dispatch cycle for a bulk write operation.
func BulkWrite(
	ctx context.Context,
	ns command.Namespace,
	models []WriteModel,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	retryWrite bool,
	sess *session.Client,
	writeConcern *writeconcern.WriteConcern,
	ordered bool,
	clock *session.ClusterClock,
	bypassDocValidation bool,
	bypassDocValidationSet bool,
) (result.BulkWrite, error) {
	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.BulkWrite{}, err
	}

	err = verifyOptions(models, ss)
	if err != nil {
		return result.BulkWrite{}, err
	}

	// If no explicit session and deployment supports sessions, start implicit session.
	if sess == nil && topo.SupportsSessions() {
		sess, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return result.BulkWrite{}, err
		}

		defer sess.EndSession()
	}

	batches := createBatches(models, ordered)
	var bwRes result.BulkWrite
	for i, batch := range batches {
		err = runBatch(ctx, ns, topo, selector, ss, sess, clock, writeConcern, retryWrite, bypassDocValidation,
			bypassDocValidationSet, !ordered, batch, i, &bwRes)
		if err != nil {
			return bwRes, err
		}
	}

	return bwRes, nil
}

func runBatch(
	ctx context.Context,
	ns command.Namespace,
	topo *topology.Topology,
	selector description.ServerSelector,
	ss *topology.SelectedServer,
	sess *session.Client,
	clock *session.ClusterClock,
	wc *writeconcern.WriteConcern,
	retryWrite bool,
	bypassDocValidation bool,
	bypassDocValidationSet bool,
	continueOnError bool,
	batch bulkWriteBatch,
	batchIndex int,
	bwRes *result.BulkWrite,
) error {
	switch batch.models[0].(type) {
	case InsertOneModel:
		res, err := runInsert(ctx, ns, topo, selector, ss, sess, clock, wc, retryWrite, batch, bypassDocValidation,
			bypassDocValidationSet)
		if err != nil && !continueOnError {
			return err
		}

		bwRes.InsertedCount += int64(res.N)
	case DeleteOneModel, DeleteManyModel:
		res, err := runDelete(ctx, ns, topo, selector, ss, sess, clock, wc, retryWrite, batch)
		if err != nil && !continueOnError {
			return err
		}

		bwRes.DeletedCount += int64(res.N)
	case ReplaceOneModel, UpdateOneModel, UpdateManyModel:
		res, err := runUpdate(ctx, ns, topo, selector, ss, sess, clock, wc, retryWrite, batch, bypassDocValidation,
			bypassDocValidationSet)
		if err != nil && !continueOnError {
			return err
		}

		bwRes.MatchedCount += res.MatchedCount
		bwRes.ModifiedCount += res.ModifiedCount
		bwRes.UpsertedCount += int64(len(res.Upserted))
		if len(res.Upserted) > 0 {
			bwRes.UpsertedIDs[int64(batchIndex)] = res.Upserted[0].ID
		}
	}

	return nil
}

func runInsert(
	ctx context.Context,
	ns command.Namespace,
	topo *topology.Topology,
	selector description.ServerSelector,
	ss *topology.SelectedServer,
	sess *session.Client,
	clock *session.ClusterClock,
	wc *writeconcern.WriteConcern,
	retryWrite bool,
	batch bulkWriteBatch,
	bypassDocValidation bool,
	bypassDocValidationSet bool,
) (result.Insert, error) {
	docs := make([]*bson.Document, len(batch.models))
	var i int
	for _, model := range batch.models {
		converted := model.(InsertOneModel)
		doc, err := option.TransformDocument(converted.Document)
		if err != nil {
			return result.Insert{}, err
		}

		docs[i] = doc
		i++
	}

	cmd := command.Insert{
		NS:           ns,
		Docs:         docs,
		Session:      sess,
		Clock:        clock,
		WriteConcern: wc,
	}

	if bypassDocValidationSet {
		cmd.Opts = []option.InsertOptioner{option.OptBypassDocumentValidation(bypassDocValidation)}
	}

	if !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) || !retryWrite || !batch.canRetry {
		cmd.Session.RetryWrite = false
		return insert(ctx, cmd, ss, nil)
	}

	cmd.Session.RetryWrite = retryWrite
	cmd.Session.IncrementTxnNumber()

	res, origErr := insert(ctx, cmd, ss, nil)
	if shouldRetry(origErr, res.WriteConcernError) {
		newServer, err := topo.SelectServer(ctx, selector)
		if err != nil || !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) {
			return res, origErr
		}

		return insert(ctx, cmd, newServer, origErr)
	}

	return res, origErr
}

func runDelete(
	ctx context.Context,
	ns command.Namespace,
	topo *topology.Topology,
	selector description.ServerSelector,
	ss *topology.SelectedServer,
	sess *session.Client,
	clock *session.ClusterClock,
	wc *writeconcern.WriteConcern,
	retryWrite bool,
	batch bulkWriteBatch,
) (result.Delete, error) {
	docs := make([]*bson.Document, len(batch.models))
	var i int

	for _, model := range batch.models {
		var doc *bson.Document
		var err error

		if dom, ok := model.(DeleteOneModel); ok {
			doc, err = createDeleteDoc(dom.Filter, dom.Collation, dom.CollationSet, false)
		} else if dmm, ok := model.(DeleteManyModel); ok {
			doc, err = createDeleteDoc(dmm.Filter, dmm.Collation, dmm.CollationSet, true)
		}

		if err != nil {
			return result.Delete{}, err
		}

		docs[i] = doc
		i++
	}

	cmd := command.Delete{
		NS:           ns,
		Deletes:      docs,
		Session:      sess,
		Clock:        clock,
		WriteConcern: wc,
	}

	if !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) || !retryWrite || !batch.canRetry {
		cmd.Session.RetryWrite = false
		return delete(ctx, cmd, ss, nil)
	}

	cmd.Session.RetryWrite = retryWrite
	cmd.Session.IncrementTxnNumber()

	res, origErr := delete(ctx, cmd, ss, nil)
	if shouldRetry(origErr, res.WriteConcernError) {
		newServer, err := topo.SelectServer(ctx, selector)
		if err != nil || !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) {
			return res, origErr
		}

		return delete(ctx, cmd, newServer, origErr)
	}

	return res, origErr
}

func runUpdate(
	ctx context.Context,
	ns command.Namespace,
	topo *topology.Topology,
	selector description.ServerSelector,
	ss *topology.SelectedServer,
	sess *session.Client,
	clock *session.ClusterClock,
	wc *writeconcern.WriteConcern,
	retryWrite bool,
	batch bulkWriteBatch,
	bypassDocValidation bool,
	bypassDocValidationSet bool,
) (result.Update, error) {
	docs := make([]*bson.Document, len(batch.models))
	var i int

	for _, model := range batch.models {
		var doc *bson.Document
		var err error

		if rom, ok := model.(ReplaceOneModel); ok {
			doc, err = createUpdateDoc(rom.Filter, rom.Replacement, nil, false,
				rom.UpdateModel, false)
		} else if uom, ok := model.(UpdateOneModel); ok {
			doc, err = createUpdateDoc(uom.Filter, uom.Update, uom.ArrayFilters, uom.ArrayFiltersSet, uom.UpdateModel,
				false)
		} else if umm, ok := model.(UpdateManyModel); ok {
			doc, err = createUpdateDoc(umm.Filter, umm.Update, umm.ArrayFilters, umm.ArrayFiltersSet, umm.UpdateModel,
				true)
		}

		if err != nil {
			return result.Update{}, err
		}

		docs[i] = doc
		i++
	}

	cmd := command.Update{
		NS:           ns,
		Docs:         docs,
		Session:      sess,
		Clock:        clock,
		WriteConcern: wc,
	}
	if bypassDocValidationSet {
		cmd.Opts = []option.UpdateOptioner{option.OptBypassDocumentValidation(bypassDocValidation)}
	}

	if !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) || !retryWrite || !batch.canRetry {
		cmd.Session.RetryWrite = false
		return update(ctx, cmd, ss, nil)
	}

	cmd.Session.RetryWrite = retryWrite
	cmd.Session.IncrementTxnNumber()

	res, origErr := update(ctx, cmd, ss, nil)
	if shouldRetry(origErr, res.WriteConcernError) {
		newServer, err := topo.SelectServer(ctx, selector)
		if err != nil || !retrySupported(topo, ss.Description(), cmd.Session, cmd.WriteConcern) {
			return res, origErr
		}

		return update(ctx, cmd, newServer, origErr)
	}

	return res, origErr
}

func verifyOptions(models []WriteModel, ss *topology.SelectedServer) error {
	maxVersion := ss.Description().WireVersion.Max
	// 3.4 is wire version 5
	// 3.6 is wire version 6

	for _, model := range models {
		var collationSet bool
		var afSet bool // arrayFilters

		switch converted := model.(type) {
		case DeleteOneModel:
			collationSet = converted.CollationSet
		case DeleteManyModel:
			collationSet = converted.CollationSet
		case ReplaceOneModel:
			collationSet = converted.CollationSet
		case UpdateOneModel:
			afSet = converted.ArrayFiltersSet
			collationSet = converted.CollationSet
		case UpdateManyModel:
			afSet = converted.ArrayFiltersSet
			collationSet = converted.CollationSet
		}

		if afSet && maxVersion < 6 {
			return ErrArrayFilters
		}

		if collationSet && maxVersion < 5 {
			return ErrCollation
		}
	}

	return nil
}

func createBatches(models []WriteModel, ordered bool) []bulkWriteBatch {
	if ordered {
		return createOrderedBatches(models)
	}

	batches := make([]bulkWriteBatch, 3)
	var i int
	for i = 0; i < 3; i++ {
		batches[i].canRetry = true
	}

	var numBatches int // number of batches used. can't use len(batches) because it's set to 3
	insertInd := -1
	updateInd := -1
	deleteInd := -1

	for _, model := range models {
		switch converted := model.(type) {
		case InsertOneModel:
			if insertInd == -1 {
				// this is the first InsertOneModel
				insertInd = numBatches
				numBatches++
			}

			batches[insertInd].models = append(batches[insertInd].models, model)
		case DeleteOneModel, DeleteManyModel:
			if deleteInd == -1 {
				deleteInd = numBatches
				numBatches++
			}

			batches[deleteInd].models = append(batches[deleteInd].models, model)
			if _, ok := converted.(DeleteManyModel); ok {
				batches[deleteInd].canRetry = false
			}
		case ReplaceOneModel, UpdateOneModel, UpdateManyModel:
			if updateInd == -1 {
				updateInd = numBatches
				numBatches++
			}

			batches[updateInd].models = append(batches[updateInd].models, model)
			if _, ok := converted.(UpdateManyModel); ok {
				batches[updateInd].canRetry = false
			}
		}
	}

	return batches
}

func createOrderedBatches(models []WriteModel) []bulkWriteBatch {
	var batches []bulkWriteBatch
	var prevKind command.WriteCommandKind = -1
	i := -1 // batch index

	for _, model := range models {
		var createNewBatch bool
		var canRetry bool
		var newKind command.WriteCommandKind

		switch model.(type) {
		case InsertOneModel:
			createNewBatch = prevKind != command.InsertCommand
			canRetry = true
			newKind = command.InsertCommand
		case DeleteOneModel:
			createNewBatch = prevKind != command.DeleteCommand
			canRetry = true
			newKind = command.DeleteCommand
		case DeleteManyModel:
			createNewBatch = prevKind != command.DeleteCommand
			newKind = command.DeleteCommand
		case ReplaceOneModel, UpdateOneModel:
			createNewBatch = prevKind != command.UpdateCommand
			canRetry = true
			newKind = command.UpdateCommand
		case UpdateManyModel:
			createNewBatch = prevKind != command.UpdateCommand
			newKind = command.UpdateCommand
		}

		if createNewBatch {
			batches = append(batches, bulkWriteBatch{
				models:   []WriteModel{model},
				canRetry: canRetry,
			})
			i++
		} else {
			batches[i].models = append(batches[i].models, model)
			if !canRetry {
				batches[i].canRetry = false // don't make it true if it was already false
			}
		}

		prevKind = newKind
	}

	return batches
}

func shouldRetry(cmdErr error, wcErr *result.WriteConcernError) bool {
	if cerr, ok := cmdErr.(command.Error); ok && cerr.Retryable() ||
		wcErr != nil && command.IsWriteConcernErrorRetryable(wcErr) {
		return true
	}

	return false
}

func createUpdateDoc(
	filter interface{},
	update interface{},
	arrayFilters []interface{},
	arrayFiltersSet bool,
	updateModel UpdateModel,
	multi bool,
) (*bson.Document, error) {
	f, err := option.TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	u, err := option.TransformDocument(update)
	if err != nil {
		return nil, err
	}

	doc := bson.NewDocument(
		bson.EC.SubDocument("q", f),
		bson.EC.SubDocument("u", u),
		bson.EC.Boolean("multi", multi),
	)

	if arrayFiltersSet {
		arr := bson.NewArray()
		for _, f := range arrayFilters {
			d, err := option.TransformDocument(f)
			if err != nil {
				return nil, err
			}

			arr.Append(bson.VC.Document(d))
		}

		doc.Append(bson.EC.Array("arrayFilters", arr))
	}

	if updateModel.CollationSet {
		collationDoc, err := updateModel.Collation.MarshalBSONDocument()
		if err != nil {
			return nil, err
		}

		doc.Append(bson.EC.SubDocument("collation", collationDoc))
	}

	if updateModel.UpsertSet {
		doc.Append(bson.EC.Boolean("upsert", updateModel.Upsert))
	}

	return doc, nil
}

func createDeleteDoc(
	filter interface{},
	collation *option.Collation,
	collationSet bool,
	many bool,
) (*bson.Document, error) {
	f, err := option.TransformDocument(filter)
	if err != nil {
		return nil, err
	}

	var limit int32 = 1
	if many {
		limit = 0
	}

	doc := bson.NewDocument(
		bson.EC.SubDocument("q", f),
		bson.EC.Int32("limit", limit),
	)

	if collationSet {
		collationDoc, err := collation.MarshalBSONDocument()
		if err != nil {
			return nil, err
		}

		doc.Append(bson.EC.SubDocument("collation", collationDoc))
	}

	return doc, nil
}
