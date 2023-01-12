package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCommandLoggingAndMonitoringProse(t *testing.T) {
	t.Parallel()

	const minServerVersion42 = "4.2"

	mt := mtest.New(t, mtest.NewOptions().
		MinServerVersion(minServerVersion42).
		Topologies(mtest.ReplicaSet).
		CreateClient(false))

	defer mt.Close()

	// inc is used to ensure parallel tests don't use the same client name.
	inc := 0
	incMutex := &sync.Mutex{}

	defaultLengthWithSuffix := len(logger.TruncationSuffix) + logger.DefaultMaxDocumentLength

	for _, tcase := range []struct {
		// name is the name of the test case
		name string

		// collectionName is the name to assign the collection for processing the operations. This should be
		// unique across test cases.
		collectionName string

		// maxDocumentLength is the maximum document length for a command message.
		maxDocumentLength uint

		// orderedLogValidators is a slice of log validators that should be 1-1 with the actual logs that are
		// propagated by the LogSink. The order here matters, the first log will be validated by the 0th
		// validator, the second log will be validated by the 1st validator, etc.
		orderedLogValidators []logTruncCaseValidator

		// operation is the operation to perform on the collection that will result in log propagation. The logs
		// created by "operation" will be validated against the "orderedLogValidators."
		operation func(context.Context, *mtest.T, *mongo.Collection)
	}{
		{
			name:           "1 Default truncation limit",
			collectionName: "46a624c57c72463d90f88a733e7b28b4",
			operation: func(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
				const documentsSize = 100

				// Construct an array docs containing the document {"x" : "y"} repeated 100 times.
				docs := []interface{}{}
				for i := 0; i < documentsSize; i++ {
					docs = append(docs, bson.D{{"x", "y"}})
				}

				// Insert docs to a collection via insertMany.
				_, err := coll.InsertMany(ctx, docs)
				assert.Nil(mt, err, "InsertMany error: %v", err)

				// Run find() on the collection where the document was inserted.
				_, err = coll.Find(ctx, bson.D{})
				assert.Nil(mt, err, "Find error: %v", err)
			},
			orderedLogValidators: []logTruncCaseValidator{
				newLogTruncCaseValidator(mt, "command", func(cmd string) error {
					if len(cmd) != defaultLengthWithSuffix {
						return fmt.Errorf("expected command to be %d bytes, got %d",
							defaultLengthWithSuffix, len(cmd))
					}

					return nil
				}),
				newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
					if len(cmd) > defaultLengthWithSuffix {
						return fmt.Errorf("expected reply to be less than %d bytes, got %d",
							defaultLengthWithSuffix, len(cmd))
					}

					return nil
				}),
				nil,
				newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
					if len(cmd) != defaultLengthWithSuffix {
						return fmt.Errorf("expected reply to be %d bytes, got %d",
							defaultLengthWithSuffix, len(cmd))
					}

					return nil
				}),
			},
		},
		{
			name:              "2 Explicitly configured truncation limit",
			collectionName:    "540baa64dc854ca2a639627e2f0918df",
			maxDocumentLength: 5,
			operation: func(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
				result := coll.Database().RunCommand(ctx, bson.D{{"hello", true}})
				assert.Nil(mt, result.Err(), "RunCommand error: %v", result.Err())
			},
			orderedLogValidators: []logTruncCaseValidator{
				newLogTruncCaseValidator(mt, "command", func(cmd string) error {
					if len(cmd) != 5+len(logger.TruncationSuffix) {
						return fmt.Errorf("expected command to be %d bytes, got %d",
							5+len(logger.TruncationSuffix), len(cmd))
					}

					return nil
				}),
				newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
					if len(cmd) != 5+len(logger.TruncationSuffix) {
						return fmt.Errorf("expected reply to be %d bytes, got %d",
							5+len(logger.TruncationSuffix), len(cmd))
					}

					return nil
				}),
			},
		},
		//{
		//	name:              "3 Truncation with multi-byte codepoints",
		//	collectionName:    "41fe9a6918044733875617b56a3125a9",
		//	maxDocumentLength: 454, // One byte away from the end of the UTF-8 sequence 世.
		//	operation: func(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
		//		_, err := coll.InsertOne(ctx, bson.D{{"x", "hello 世"}})
		//		assert.Nil(mt, err, "InsertOne error: %v", err)
		//	},
		//	orderedLogValidators: []logTruncCaseValidator{
		//		nil,
		//		newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
		//			fmt.Println("cmd: ", cmd)
		//			// Ensure that the tail of the command string is "hello ".
		//			if !strings.HasSuffix(cmd, "hello "+logger.TruncationSuffix) {
		//				return fmt.Errorf("expected command to end with 'hello ', got %q", cmd)
		//			}

		//			return nil
		//		}),
		//	},
		//},
	} {
		tcase := tcase

		mt.Run(tcase.name, func(mt *mtest.T) {
			mt.Parallel()

			incMutex.Lock()
			inc++

			incMutex.Unlock()

			const deadline = 1 * time.Second
			ctx := context.Background()

			sinkCtx, sinkCancel := context.WithDeadline(ctx, time.Now().Add(deadline))
			defer sinkCancel()

			validator := func(order int, level int, msg string, keysAndValues ...interface{}) error {
				// If the order exceeds the length of the "orderedCaseValidators," then throw an error.
				if order >= len(tcase.orderedLogValidators) {
					return fmt.Errorf("not enough expected cases to validate")
				}

				caseValidator := tcase.orderedLogValidators[order]
				if caseValidator == nil {
					return nil
				}

				return tcase.orderedLogValidators[order](keysAndValues...)
			}

			sink := newTestLogSink(sinkCtx, mt, len(tcase.orderedLogValidators), validator)

			// Configure logging with a minimum severity level of "debug" for the "command" component
			// without explicitly configure the max document length.
			loggerOpts := options.Logger().SetSink(sink).
				SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)

			if mdl := tcase.maxDocumentLength; mdl != 0 {
				loggerOpts.SetMaxDocumentLength(mdl)
			}

			clientOpts := options.Client().SetLoggerOptions(loggerOpts).ApplyURI(mtest.ClusterURI())

			client, err := mongo.Connect(context.TODO(), clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)

			coll := mt.CreateCollection(mtest.Collection{
				Name:   tcase.collectionName,
				Client: client,
			}, false)

			tcase.operation(ctx, mt, coll)

			// Verify the logs.
			if err := <-sink.errs(); err != nil {
				mt.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
