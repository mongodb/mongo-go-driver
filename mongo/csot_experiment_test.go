package mongo

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultVolume = 100_000

type csotTestCase struct {
	failRate   float64 // Rate [0,1] of volume that should short-circuit
	volume     uint    // Number of records to load
	goroutines uint    // Number of goroutines to evenly split op execution
}

func loadLargeCollection(t *testing.T, coll *Collection, tcase csotTestCase) {
	const batchSize = 500 // Size of batches to load for testing

	docs := make([]interface{}, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		docs = append(docs, bson.D{
			{"field1", rand.Int63()},
			{"field2", rand.Int31()},
		})
	}

	// Partition the volume into equal sizes per go routine. Use the floor if the
	// volume is not divisible by the number of goroutines.
	perGoroutine := tcase.volume / tcase.goroutines

	// Number of batches to insert per goroutine. Use the floor if perGoroutine
	// is not divisible by the batchSize.
	batches := perGoroutine / batchSize

	errs := make(chan error, tcase.goroutines)
	done := make(chan struct{}, tcase.goroutines)

	for i := 0; i < int(tcase.goroutines); i++ {
		go func(i int) {
			for j := 0; j < int(batches); j++ {
				_, err := coll.InsertMany(context.Background(), docs)
				if err != nil {
					errs <- fmt.Errorf("goroutine %v failed: %w", i, err)

					break
				}
			}

			done <- struct{}{}
		}(i)
	}

	go func() {
		defer close(errs)

		for i := 0; i < int(tcase.goroutines); i++ {
			<-done
		}
	}()

	for err := range errs {
		require.NoError(t, err)
	}
}

type latencyStats struct {
	max    time.Duration
	min    time.Duration
	median time.Duration
	mean   time.Duration
	p10    time.Duration
	p90    time.Duration
	p99    time.Duration
}

func getStats(t *testing.T, times []time.Duration) *latencyStats {
	t.Helper()

	samples := make(stats.Float64Data, len(times))
	for i := range times {
		samples[i] = float64(times[i])
	}

	maxv, err := stats.Max(samples)
	require.NoError(t, err)

	minv, err := stats.Min(samples)
	require.NoError(t, err)

	medv, err := stats.Median(samples)
	require.NoError(t, err)

	mean, err := stats.Mean(samples)
	require.NoError(t, err)

	p90, err := stats.Percentile(samples, 90)
	require.NoError(t, err)

	p99, err := stats.Percentile(samples, 99)
	require.NoError(t, err)

	return &latencyStats{
		max:    time.Duration(maxv),
		min:    time.Duration(minv),
		median: time.Duration(medv),
		mean:   time.Duration(mean),
		p90:    time.Duration(p90),
		p99:    time.Duration(p99),
	}
}

func getQueryStats(t *testing.T, coll *Collection, query bson.D, tcase csotTestCase) *latencyStats {
	t.Helper()

	samples := make([]time.Duration, 0, tcase.goroutines*10)
	var samplesMu sync.Mutex

	errs := make(chan error, tcase.goroutines)
	done := make(chan struct{}, tcase.goroutines)

	for i := 0; i < int(tcase.goroutines); i++ {
		go func() {
			// Higher durations yield more accurate statistics.
			durations := make([]time.Duration, 10)
			for i := 0; i < len(durations); i++ {
				start := time.Now()
				err := coll.FindOne(context.Background(), query).Err()
				durations[i] = time.Since(start)

				if err != nil && err != ErrNoDocuments {
					errs <- fmt.Errorf("failed to collect query stats: %w", err)

					break
				}
			}

			samplesMu.Lock()
			samples = append(samples, durations...)
			samplesMu.Unlock()

			done <- struct{}{}
		}()
	}

	go func() {
		defer close(errs)

		for i := 0; i < int(tcase.goroutines); i++ {
			<-done
		}
	}()

	for err := range errs {
		require.NoError(t, err)
	}

	samplesMu.Lock()
	defer samplesMu.Unlock()

	return getStats(t, samples[:])
}

func runCSOTTestCase(t *testing.T, client *Client, tcase csotTestCase) {
	t.Helper()

	require.NotNil(t, client.timeout, "CSOT must be enabled")

	// Ensure the failRate is in the interval [0,1]
	require.LessOrEqual(t, tcase.failRate, 1.0)
	require.GreaterOrEqual(t, tcase.failRate, 0.0)

	// Initialize a collection with the name "large<uuid>".
	uuid, err := uuid.New()
	require.NoError(t, err, "failed to create uuid for collection name")

	collName := fmt.Sprintf("large%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])

	unindexedColl := client.Database("testdb").Collection(collName)
	defer func() { unindexedColl.Drop(context.Background()) }()

	// Load the test data.
	loadLargeCollection(t, unindexedColl, tcase)

	// Use the query that will be used in the "find" operations to get a various
	// query statistics that can be used to determine how to timeout an operation.
	query := bson.D{{"field1", "doesntexist"}}

	qstats := getQueryStats(t, unindexedColl, query, tcase)

	// Set the timeout values in the "timeouts" array where there a
	// failRatio-percent of the timeouts are the minimum from query stats and
	// the rest are max.
	expectedFailures := math.Floor(float64(tcase.volume) * tcase.failRate)

	timeouts := make([]time.Duration, tcase.volume)
	for i := 0; i < int(tcase.volume); i++ {
		if i >= int(expectedFailures) {
			timeouts[i] = qstats.max
		} else {
			timeouts[i] = qstats.min
		}
	}

	// Partition the volume into equal sizes per go routine. Use the floor if the
	// volume is not divisible by the number of goroutines.
	perGoroutine := tcase.volume / tcase.goroutines

	errs := make(chan error, tcase.goroutines*perGoroutine)
	done := make(chan struct{}, tcase.goroutines)

	// Run the find query on an unindex collection in partitions upto the number
	// of goroutines.
	for i := 0; i < int(tcase.goroutines); i++ {
		go func(i int) {
			for j := 0; j < int(perGoroutine); j++ {
				idx := (int(perGoroutine)-1)*i + j + i
				timeout := timeouts[idx]

				ctx, cancel := context.WithTimeout(context.Background(), timeout)

				err := unindexedColl.FindOne(ctx, query).Err()
				cancel()

				if err != nil && err != ErrNoDocuments {
					errs <- err
				}
			}

			done <- struct{}{}
		}(i)
	}

	go func() {
		defer close(errs)
		for i := 0; i < int(tcase.goroutines); i++ {
			<-done
		}
	}()

	errCount := 0
	for err := range errs {
		if IsTimeout(err) {
			errCount++
		}
	}

	t.Log("number of timeout errors: ", errCount)
}

func TestExperimentalCSOT(t *testing.T) {
	const uri = "mongodb://localhost:27017"

	clientOpts := options.Client().SetTimeout(10 * time.Minute).ApplyURI(uri)

	client, err := Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	ex := csotTestCase{
		failRate:   0.314,
		goroutines: 10,
		volume:     100,
	}

	runCSOTTestCase(t, client, ex)
}
