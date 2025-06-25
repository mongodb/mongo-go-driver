package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestCMAPProse_PendingResponse_ConnectionAliveness(t *testing.T) {
	const timeoutMS = 200 * time.Millisecond

	// Establish a direct connection to the proxy.
	proxyURI := "mongodb://127.0.0.1:28017/?directConnection=true"
	//t.Setenv("MONGODB_URI", proxyURI)
	//
	// Print the MONGODB_URI environment variable for debugging purposes
	fmt.Println("MONGODB_URI", os.Getenv("MONGODB_URI"))

	clientOpts := options.Client().ApplyURI(proxyURI).SetMaxPoolSize(1).SetDirect(true)
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false).ClientOptions(clientOpts))

	fmt.Println("Running TestCMAPProse_PendingResponse_ConnectionAliveness...")

	mt.Run("fails", func(mt *mtest.T) {
		// Create a command document that instructs the proxy to dely 2x the
		// timeoutMS for the operation then never respond.
		proxyTest := bson.D{
			{Key: "actions", Value: bson.A{
				// Causes the timeout in the initial try.
				bson.D{{Key: "delayMs", Value: 400}},
				// Send nothing back to the client, ever.
				bson.D{{Key: "sendBytes", Value: 0}},
			}},
		}

		type myStruct struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		cmd := bson.D{
			{Key: "insert", Value: "mycoll"},
			{Key: "documents", Value: bson.A{myStruct{Name: "Alice", Age: 30}}},
			{Key: "proxyTest", Value: proxyTest},
		}

		db := mt.Client.Database("testdb")
		coll := db.Collection("mycoll")

		_ = coll.Drop(context.Background()) // Ensure the collection is clean before the test.

		// Run the command against the proxy with timeoutMS.
		ctx, cancel := context.WithTimeout(context.Background(), timeoutMS)
		defer cancel()

		err := db.RunCommand(ctx, cmd).Err()
		require.Error(mt, err, "expected command to fail due to timeout")
		assert.ErrorIs(mt, err, context.DeadlineExceeded)

		// Wait 3 seconds to ensure there is time left in the pending response state.
		time.Sleep(3 * time.Second)

		// Run an insertOne without a timeout. Expect the pending response to fail
		// at the aliveness check. However, the insert should succeed since pending
		// response failures are retryable.
		_, err = coll.InsertOne(context.Background(), myStruct{Name: "Bob", Age: 25})
		require.NoError(mt, err, "expected insertOne to succeed after pending response aliveness check")

		// There should be 1 ConnectionPendingResponseStarted event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadStarted())

		// There should be 1 ConnectionPendingResponseFailed event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadFailed())

		// There should be 0 ConnectionPendingResponseSucceeded event.
		assert.Equal(mt, 0, mt.NumberConnectionsPendingReadSucceeded())

		// There should be 1 ConnectionClosed event.
		assert.Equal(mt, 1, mt.NumberConnectionsClosed())
	})

	mt.Run("succeeds", func(mt *mtest.T) {
		// Create a command document that instructs the proxy to dely 2x the
		// timeoutMS for the operation, then responds with exactly 1 byte for
		// the alivness check and finally with the entire message.
		proxyTest := bson.D{
			{Key: "actions", Value: bson.A{
				// Causes the timeout in the initial try.
				bson.D{{Key: "delayMs", Value: 400}},
				// Send exactly one byte so that the aliveness check succeeds.
				bson.D{{Key: "sendBytes", Value: 1}},
				// Cause another delay for the retry operation.
				bson.D{{Key: "delayMs", Value: 10}},
				// Send the rest of the response for discarding on retry.
				bson.D{{Key: "sendAll", Value: true}},
			}},
		}

		type myStruct struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		cmd := bson.D{
			{Key: "insert", Value: "mycoll"},
			{Key: "documents", Value: bson.A{myStruct{Name: "Alice", Age: 30}}},
			{Key: "proxyTest", Value: proxyTest},
		}

		db := mt.Client.Database("testdb")
		coll := db.Collection("mycoll")

		_ = coll.Drop(context.Background()) // Ensure the collection is clean before the test.

		// Run the command against the proxy with timeoutMS.
		ctx, cancel := context.WithTimeout(context.Background(), timeoutMS)
		defer cancel()

		err := db.RunCommand(ctx, cmd).Err()
		require.Error(mt, err, "expected command to fail due to timeout")
		assert.ErrorIs(mt, err, context.DeadlineExceeded)

		// Wait 3 seconds to ensure there is time left in the pending response state.
		time.Sleep(3 * time.Second)

		// Run an insertOne without a timeout. Expect the pending response to fail
		// at the aliveness check.
		_, err = coll.InsertOne(context.Background(), myStruct{Name: "Bob", Age: 25})
		require.NoError(mt, err, "expected insertOne to succeed after pending response aliveness check")

		// There should be 1 ConnectionPendingResponseStarted event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadStarted())

		// There should be 0 ConnectionPendingResponseFailed event.
		assert.Equal(mt, 0, mt.NumberConnectionsPendingReadFailed())

		// There should be 1 ConnectionPendingResponseSucceeded event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadSucceeded())

		// The connection should not have been closed.
		assert.Equal(mt, 0, mt.NumberConnectionsClosed())
	})
}
