// This set of tests is temporary and is meant to confirm the behavior of the new
// global Timeout options on Client, Database, Collection and Find work as intended.
// The tests will mostly be removed in favor of spec and prose tests. If we still
// require more test coverage, it may be placed in mongo/integration/client_test.go

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

func TestTimeoutClient(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	cliOpts := options.Client().SetTimeout(10 * time.Second)
	mtOpts := mtest.NewOptions().ClientOptions(cliOpts)
	mt.RunOpts("maxTimeMS is appended", mtOpts, func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		mt.ClearEvents()
		_, err = mt.Coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
		evt := mt.GetStartedEvent()

		val, err := evt.Command.LookupErr("maxTimeMS")
		assert.Nil(mt, err, "'maxTimeMS' not present in Find command")
		mtms, ok := val.AsInt64OK()
		assert.True(mt, ok, "expected 'maxTimeMS' to be of type int64, got %T", mtms)

		// Remaining Timeout appended to Find command should be somewhere between 9500 and 10000
		// milliseconds because a few milliseconds will have passed between the start of the
		// operation and the appension of 'maxTimeMS'.
		timeoutWithinBounds := 9500 < mtms && mtms < 10000
		assert.True(mt, timeoutWithinBounds,
			"expected 'maxTimeMS' to be within 9500 and 10000 ms, got %v", mtms)
	})

	lowTOCliOpts := options.Client().SetTimeout(1 * time.Nanosecond)
	mtOpts = mtest.NewOptions().ClientOptions(lowTOCliOpts)
	mt.RunOpts("deadline exceeded error", mtOpts, func(mt *mtest.T) {
		// Normally, a Timeout of 1ns on the client would cause context deadline exceeded errors
		// on the below InsertOne and the preceding setup commands. For now, though, Timeout is
		// only passed down to Find.
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// Since there is no deadline on the provided context, the operation should honor the
		// Timeout of 1ns, and the operation should short-circuit before being sent to the server
		// with ErrDeadlineWouldBeExceeded.
		_, err = mt.Coll.Find(context.Background(), bson.D{})
		assert.True(mt, strings.Contains(err.Error(), driver.ErrDeadlineWouldBeExceeded.Error()),
			"expected error to contain ErrDeadlineWouldBeExceeded, got %v", err.Error())
	})
	mt.RunOpts("context with deadline supersedes Timeout", mtOpts, func(mt *mtest.T) {
		// Normally, a Timeout of 1ns on the client would cause context deadline exceeded errors
		// on the below InsertOne and the preceding setup commands. For now, though, Timeout is
		// only passed down to Find.
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()

		// Using a new context with a deadline should ignore the Timeout of 1ns, and the operation
		// should succeed.
		_, err = mt.Coll.Find(ctx, bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
}

func TestTimeoutDatabase(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	mt.Run("maxTimeMS is appended", func(mt *mtest.T) {
		dbOptions := options.Database().SetTimeout(10 * time.Second)
		coll := mt.Client.Database("test", dbOptions).Collection("test")
		defer func() {
			_ = coll.Drop(context.Background())
		}()

		_, err := coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		mt.ClearEvents()
		_, err = coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
		evt := mt.GetStartedEvent()

		val, err := evt.Command.LookupErr("maxTimeMS")
		assert.Nil(mt, err, "'maxTimeMS' not present in Find command")
		mtms, ok := val.AsInt64OK()
		assert.True(mt, ok, "expected 'maxTimeMS' to be of type int64, got %T", mtms)

		// Remaining timeout appended to Find command should be somewhere between 9500 and 10000
		// milliseconds because a few milliseconds will have passed between the start of the
		// operation and the appension of 'maxTimeMS'.
		timeoutWithinBounds := 9500 < mtms && mtms < 10000
		assert.True(mt, timeoutWithinBounds,
			"expected 'maxTimeMS' to be within 9500 and 10000 ms, got %v", mtms)
	})
}

func TestTimeoutCollection(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	mt.Run("maxTimeMS is appended", func(mt *mtest.T) {
		collOptions := options.Collection().SetTimeout(10 * time.Second)
		coll := mt.Client.Database("test").Collection("test", collOptions)
		defer func() {
			_ = coll.Drop(context.Background())
		}()

		_, err := coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		mt.ClearEvents()
		_, err = coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
		evt := mt.GetStartedEvent()

		val, err := evt.Command.LookupErr("maxTimeMS")
		assert.Nil(mt, err, "'maxTimeMS' not present in Find command")
		mtms, ok := val.AsInt64OK()
		assert.True(mt, ok, "expected 'maxTimeMS' to be of type int64, got %T", mtms)

		// Remaining timeout appended to Find command should be somewhere between 9500 and 10000
		// milliseconds because a few milliseconds will have passed between the start of the
		// operation and the appension of 'maxTimeMS'.
		timeoutWithinBounds := 9500 < mtms && mtms < 10000
		assert.True(mt, timeoutWithinBounds,
			"expected 'maxTimeMS' to be within 9500 and 10000 ms, got %v", mtms)
	})
}
