package benchmark

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo"
)

const (
	singleAndMultiDataDir = "single_and_multi_document"
	tweetData             = "tweet.json"
	smallData             = "small_doc.json"
	largeData             = "large_doc.json"
)

func getClientDB(ctx context.Context) (*mongo.Database, error) {
	cs, err := testutil.GetConnString()
	if err != nil {
		return nil, err
	}
	client, err := mongo.NewClientFromConnString(cs)
	if err != nil {
		return nil, err
	}
	if err = client.Connect(ctx); err != nil {
		return nil, err
	}

	db := client.Database(testutil.GetDBName(cs))
	return db, nil
}

func SingleRunCommand(ctx context.Context, tm TimerManager, iters int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}
	defer db.Client().Disconnect(ctx)

	cmd := bson.NewDocument(bson.EC.Boolean("ismaster", true))

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		out, err := db.RunCommand(ctx, cmd)
		if err != nil {
			return err
		}
		// read the document and then throw it away to prevent
		if len(out) == 0 {
			return errors.New("output of ismaster is empty")
		}
	}
	tm.StopTimer()

	return nil
}

func SingleFindOneByID(ctx context.Context, tm TimerManager, iters int) error {
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
		id := int32(i)
		res, err := coll.InsertOne(ctx, doc.Set(bson.EC.Int32("_id", id)))
		if err != nil {
			return err
		}
		if res.InsertedID == nil {
			return errors.New("insert failed")
		}
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		res := coll.FindOne(ctx, bson.NewDocument(bson.EC.Int32("_id", int32(i))))
		if res == nil {
			return errors.New("find one query produced nil result")
		}
	}

	tm.StopTimer()

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func singleInsertCase(ctx context.Context, tm TimerManager, iters int, data string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := getClientDB(ctx)
	if err != nil {
		return err
	}
	defer db.Client().Disconnect(ctx)

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

	coll := db.Collection("corpus")

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		if _, err = coll.InsertOne(ctx, doc); err != nil {
			return err
		}

		// TODO: should be remove after resolving GODRIVER-468
		_ = doc.Delete("_id")
	}

	tm.StopTimer()

	if err = db.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func SingleInsertSmallDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, smallData)
}

func SingleInsertLargeDocument(ctx context.Context, tm TimerManager, iters int) error {
	return singleInsertCase(ctx, tm, iters, largeData)
}
