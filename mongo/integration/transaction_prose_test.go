package integration

import (
	"context"
	"errors"
	"log"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	xsession "go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"golang.org/x/sync/errgroup"
)

func TestTransactionProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().Topologies(mtest.LoadBalanced).CreateClient(false))
	defer mt.Close()

	mt.Run("Concurrent Abort Transactions on a Load-Balanced Cluster", func(mt *mtest.T) {
		const threadCount = 100

		// Start a new ClientSession with default options and start a transaction.
		session, err := mt.Client.StartSession()
		if err != nil {
			log.Fatalf("error starting session: %v", err)
		}

		if err = session.StartTransaction(); err != nil {
			log.Fatalf("error starting transaction: %v", err)
		}

		abortTransactionCount := 0          // number of times abortTransaction was called
		abortTransactionErrors := []error{} // errors returned by abortTransaction

		ctx := context.Background()

		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			// Within the transaction, insert a document into a
			// collection.
			if _, err := mt.Coll.InsertOne(sc, bson.D{{"x", 1}}); err != nil {
				return err
			}

			// After the insert operation, concurrently abort the
			// transaction 100 times.
			g, _ := errgroup.WithContext(sc)
			for i := 0; i < threadCount; i++ {
				g.Go(func() error {
					if err := session.AbortTransaction(sc); err != nil {
						abortTransactionErrors = append(abortTransactionErrors, err)

						return err
					}

					abortTransactionCount++

					return nil
				})
			}

			return g.Wait()
		})

		// Assert that only one of the abortTransaction operations
		// succeeds.
		if abortTransactionCount != 1 {
			mt.Fatalf("expected abortTransactionCount to be 1, got %d", abortTransactionCount)
		}

		// There should be "concurrencyCount - 1" errors in
		// abortTransactionErrors.
		if len(abortTransactionErrors) != threadCount-1 {
			mt.Fatalf("expected %d errors, got %d", threadCount-1, len(abortTransactionErrors))
		}

		// Assert that 99 operations raise an error containing the
		// message "Cannot call abortTransaction twice".
		for _, err := range abortTransactionErrors {
			if !errors.Is(err, xsession.ErrAbortTwice) {
				mt.Fatalf("expected error %v, got %v", xsession.ErrAbortTwice, err)
			}
		}

		session.EndSession(ctx)
	})
}
