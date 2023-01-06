package integration

import (
	"context"
	"errors"
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

var (
	ErrLogNotTruncated = fmt.Errorf("log message not truncated")
)

type testLogSink struct {
	logs       chan func() (int, string, []interface{})
	bufferSize int
	logsCount  int
	errsCh     chan error
}

type logValidator func(order int, level int, msg string, keysAndValues ...interface{}) error

func newTestLogSink(ctx context.Context, bufferSize int, validator logValidator) *testLogSink {
	sink := &testLogSink{
		logs:       make(chan func() (int, string, []interface{}), bufferSize),
		errsCh:     make(chan error, bufferSize),
		bufferSize: bufferSize,
	}

	go func() {
		order := 0
		for log := range sink.logs {
			select {
			case <-ctx.Done():
				sink.errsCh <- ctx.Err()

				return
			default:
			}

			level, msg, args := log()
			if err := validator(order, level, msg, args...); err != nil {
				sink.errsCh <- fmt.Errorf("invalid log at order %d for level %d and msg %q: %v", order,
					level, msg, err)
			}

			order++
		}

		close(sink.errsCh)
	}()

	return sink
}

func (sink *testLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	sink.logs <- func() (int, string, []interface{}) {
		return level, msg, keysAndValues
	}

	if sink.logsCount++; sink.logsCount == sink.bufferSize {
		close(sink.logs)
	}
}

func (sink *testLogSink) errs() <-chan error {
	return sink.errsCh
}

func findLogValue(t *testing.T, key string, values ...interface{}) interface{} {
	t.Helper()

	for i := 0; i < len(values); i += 2 {
		if values[i] == key {
			return values[i+1]
		}
	}

	return nil
}

func validateCommandTruncated(t *testing.T, commandName string, values ...interface{}) error {
	t.Helper()

	cmd := findLogValue(t, commandName, values...)
	if cmd == nil {
		return fmt.Errorf("%q not found in keys and values", commandName)
	}

	cmdStr, ok := cmd.(string)
	if !ok {
		return fmt.Errorf("command is not a string")
	}

	if len(cmdStr) != 1000+len(logger.TruncationSuffix) {
		return ErrLogNotTruncated
	}

	return nil
}

func TestCommandLoggingAndMonitoringProse(t *testing.T) {
	t.Parallel()

	const minServerVersion42 = "4.2"

	mt := mtest.New(t, mtest.NewOptions().
		MinServerVersion(minServerVersion42).
		CreateClient(false))

	defer mt.Close()

	// inc is used to ensure parallel tests don't use the same client name.
	inc := 0
	incMutex := &sync.Mutex{}

	mt.Run("1 Default truncation limit", func(mt *mtest.T) {
		mt.Parallel()

		incMutex.Lock()
		inc++

		incMutex.Unlock()

		const documentsSize = 100
		const expectedNumberOfLogs = 4
		const deadline = 1 * time.Second

		collectionName := "46a624c57c72463d90f88a733e7b28b4" + fmt.Sprintf("%d", inc)

		ctx := context.Background()

		sinkCtx, sinkCancel := context.WithDeadline(ctx, time.Now().Add(deadline))
		defer sinkCancel()

		// Construct a log sink that will validate the logs as they propagate.
		validator := func(order int, level int, msg string, keysAndValues ...interface{}) error {
			switch order {
			case 0: // Command started for "insert"
				return validateCommandTruncated(mt.T, "command", keysAndValues...)
			case 1: // Command succeeded for "insert"
				err := validateCommandTruncated(mt.T, "reply", keysAndValues...)
				if err != nil && !errors.Is(err, ErrLogNotTruncated) {
					return err
				}

				return nil
			case 2: // Command started for "find"
				return nil
			case 3: // Command succeeded for "find"
				return validateCommandTruncated(mt.T, "reply", keysAndValues...)
			}

			return nil
		}

		sink := newTestLogSink(sinkCtx, expectedNumberOfLogs, validator)

		// Configure logging with a minimum severity level of "debug" for the "command" component without
		// explicitly configure the max document length.
		loggerOpts := options.Logger().SetSink(sink).
			SetComponentLevels(map[options.LogComponent]options.LogLevel{
				options.CommandLogComponent: options.DebugLogLevel,
			})

		clientOpts := options.Client().SetLoggerOptions(loggerOpts).ApplyURI(mtest.ClusterURI())

		client, err := mongo.Connect(context.TODO(), clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)

		coll := mt.CreateCollection(mtest.Collection{
			Name:   collectionName,
			Client: client,
		}, false)

		// Construct an array docs containing the document {"x" : "y"} repeated 100 times.
		docs := []interface{}{}
		for i := 0; i < documentsSize; i++ {
			docs = append(docs, bson.D{{"x", "y"}})
		}

		// Insert docs to a collection via insertMany.
		_, err = coll.InsertMany(context.Background(), docs)
		assert.Nil(mt, err, "InsertMany error: %v", err)

		// Run find() on the collection where the document was inserted.
		_, err = coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)

		// Verify the logs.
		if err := <-sink.errs(); err != nil {
			mt.Fatalf("unexpected error: %v", err)
		}
	})
}
