package benchmark

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/pkg/errors"
)

func MultiFindMany(ctx context.Context, tm TimerManager, iters int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}

	db = db.Client().Database("perftest")
	if err = db.Drop(ctx); err != nil {
		return err
	}

	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, singleAndMultiDataDir, tweetData)
	if err != nil {
		return err
	}

	coll := db.Collection("corpus")

	for i := 0; i < iters; i++ {
		if _, err = coll.InsertOne(ctx, doc); err != nil {
			return err
		}

		// TODO: should be remove after resolving GODRIVER-468
		_ = doc.Delete("_id")
	}

	tm.ResetTimer()

	cursor, err := coll.Find(ctx, bson.NewDocument())
	if err != nil {
		return err
	}

	counter := 0
	for cursor.Next(ctx) {
		err = cursor.Err()
		if err != nil {
			return err
		}
		counter++
	}

	if counter != iters {
		return errors.New("problem iterating cursors")

	}

	tm.StopTimer()

	if err = cursor.Close(ctx); err != nil {
		return err
	}

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func multiInsertCase(ctx context.Context, tm TimerManager, iters int, data string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}

	db = db.Client().Database("perftest")
	if err = db.Drop(ctx); err != nil {
		return err
	}

	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, singleAndMultiDataDir, data)
	if err != nil {
		return err
	}

	_, err = db.RunCommand(ctx, bson.NewDocument(bson.EC.String("create", "corpus")))
	if err != nil {
		return err
	}

	payload := make([]interface{}, iters)
	for idx := range payload {
		payload[idx] = *doc
	}

	coll := db.Collection("corpus")

	tm.ResetTimer()
	res, err := coll.InsertMany(ctx, payload)
	if err != nil {
		return err
	}
	tm.StopTimer()

	if len(res.InsertedIDs) != iters {
		return errors.New("bulk operation did not complete")
	}

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func MultiInsertSmallDocument(ctx context.Context, tm TimerManager, iters int) error {
	return multiInsertCase(ctx, tm, iters, smallData)
}

func MultiInsertLargeDocument(ctx context.Context, tm TimerManager, iters int) error {
	return multiInsertCase(ctx, tm, iters, largeData)
}
