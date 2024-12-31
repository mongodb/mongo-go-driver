package benchmark

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"math/rand"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gonum.org/v1/gonum/integrate"
	"gonum.org/v1/gonum/stat"
)

const defaultConnectionChurnDB = "connectionChurnDB"
const defaultLargeCollectionSize = 1000

type ccCfg struct {
	collName   string // name of the large collection
	ops        uint   // number of operations to run
	goRoutines uint   // number of goroutines to split ops over
}

type ccResult struct {
	RunID             int64 // unifying ID for runs between 1 and 100 percentiles
	ShortCircuitRate  float64
	Throughput        float64 // operations per second
	ThroughputActual  float64 // throughput * startedRate
	ThroughputSuccess float64 // throughput * successRate
	Percentile        float64
	TimeoutRate       float64 // timeout errors per operation
	StartedRate       float64
}

func calculateThroughputPercentile(p float64, results []ccResult) float64 {
	throughputs := make([]float64, len(results))
	for i := range throughputs {
		throughputs[i] = results[i].Throughput
	}

	sort.Float64s(throughputs)

	return stat.Quantile(p, stat.Empirical, throughputs, nil)
}

// calculateSimpsonsE returns an approximation of the area under the curve of
// discrete data defined by [100 percentiles, short-circuit succeeded].
func calculateSimpsonsE(results []ccResult) float64 {
	ff64s := []float64{}
	NaNSet := map[int]bool{}

	ff64sAllZero := true
	for idx, result := range results {
		if !math.IsNaN(result.ShortCircuitRate) {
			if result.ShortCircuitRate != 0 {
				ff64sAllZero = false
			}
			ff64s = append(ff64s, result.ShortCircuitRate)
		} else {
			NaNSet[idx] = true
		}
	}

	fx64s := []float64{}
	fx64sAllZero := true
	for idx, result := range results {
		if NaNSet[idx] {
			continue
		}

		if result.Percentile != 0 {
			fx64sAllZero = false
		}

		fx64s = append(fx64s, result.Percentile)
	}

	if fx64sAllZero && ff64sAllZero {
		return 0
	}

	return integrate.Simpsons(fx64s, ff64s)
}

func benchmarkConnectionChurnTO(
	b *testing.B,
	runID int64,
	to time.Duration,
	cfg ccCfg,
	opts ...*options.ClientOptions,
) ccResult {
	clientOpts := options.MergeClientOptions(opts...)

	var connectionsClosed atomic.Int64
	poolMonitor := &event.PoolMonitor{
		Event: func(pe *event.PoolEvent) {
			if pe.Type == event.ConnectionClosed {
				connectionsClosed.Add(1)
			}
		},
	}

	var commandFailed atomic.Int64
	var commandSucceeded atomic.Int64
	var commandStarted atomic.Int64

	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, cse *event.CommandStartedEvent) {
			if cse.CommandName == "find" {
				commandStarted.Add(1)
			}
		},
		Succeeded: func(_ context.Context, cse *event.CommandSucceededEvent) {
			if cse.CommandName == "find" {
				commandSucceeded.Add(1)
			}
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			if evt.CommandName == "find" {
				commandFailed.Add(1)
			}
		},
	}

	clientOpts.SetTimeout(0).SetMonitor(cmdMonitor).SetPoolMonitor(poolMonitor)

	client, err := mongo.Connect(clientOpts)
	require.NoError(b, err)

	defer func() {
		err := client.Disconnect(context.Background())
		require.NoError(b, err)
	}()

	perGoroutine := cfg.ops / cfg.goRoutines

	errs := make(chan error, cfg.goRoutines*perGoroutine)
	done := make(chan struct{}, cfg.goRoutines)

	coll := client.Database(defaultConnectionChurnDB).Collection(cfg.collName)

	query := bson.D{{Key: "field1", Value: "doesntexist"}}

	startTime := time.Now()

	// Run the find query on an unindex collection in partitions upto the number
	// of goroutines.
	for i := 0; i < int(cfg.goRoutines); i++ {
		go func(i int) {
			for j := 0; j < int(perGoroutine); j++ {
				ctx, cancel := context.WithTimeout(context.Background(), to)

				err := coll.FindOne(ctx, query).Err()
				cancel()

				if err != nil && err != mongo.ErrNoDocuments {
					errs <- err
				}
			}

			done <- struct{}{}
		}(i)
	}

	go func() {
		defer close(errs)
		defer close(done)

		for i := 0; i < int(cfg.goRoutines); i++ {
			<-done
		}
	}()

	gotTimeoutErrCount := 0
	for err := range errs {
		if mongo.IsTimeout(err) {
			// We don't consider "ErrDeadlineWouldBeExceeded" errors, these would not
			// result in a connection closing.
			gotTimeoutErrCount++
		}
	}

	elapsed := time.Since(startTime)

	timeoutRate := float64(gotTimeoutErrCount) / float64(cfg.ops)
	startedRate := float64(commandStarted.Load()) / float64(cfg.ops)
	successRate := float64(commandSucceeded.Load()) / float64(cfg.ops)

	throughput := float64(cfg.ops) / elapsed.Seconds()
	throughputActual := throughput * startedRate
	throughputSuccess := throughput * successRate

	shortCircuitRate := 1.0
	if gotTimeoutErrCount != 0 {
		shortCircuitRate = 1.0 - float64(connectionsClosed.Load())/float64(gotTimeoutErrCount)
	}

	//b.ReportMetric(float64(cfg.ops), "ops")
	//b.ReportMetric(float64(commandStarted.Load()), "round-trips")
	//b.ReportMetric(float64(commandFailed.Load()), "failures")
	//b.ReportMetric(float64(gotTimeoutErrCount), "t/o")
	//b.ReportMetric(shortCircuitRate, "closure/t/o")

	//fmt.Println("data: ", cfg.ops, commandStarted, commandSucceeded, commandFailed, gotTimeoutErrCount)
	return ccResult{
		RunID:             runID,
		ShortCircuitRate:  shortCircuitRate,
		Throughput:        throughput,
		ThroughputActual:  throughputActual,
		ThroughputSuccess: throughputSuccess,
		TimeoutRate:       timeoutRate,
		StartedRate:       startedRate,
	}
}

// Record metrics to an Atlas server.
func recordCCThroughputResults(b *testing.B, results []ccResult) {
	metricAtlasURI := os.Getenv("METRICS_ATLAS_URI")
	if metricAtlasURI == "" {
		return
	}

	client, err := mongo.Connect(options.Client().ApplyURI(metricAtlasURI))
	require.NoError(b, err)

	defer func() {
		err := client.Disconnect(context.Background())
		require.NoError(b, err)
	}()

	coll := client.Database(defaultConnectionChurnDB).Collection("throughput")

	_, err = coll.InsertMany(context.Background(), results)
	require.NoError(b, err)
}

func BenchmarkConnectionChurn(b *testing.B) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	tests := []struct {
		name       string
		goRoutines int
		operations int
		clientOpts *options.ClientOptions
	}{
		{
			name:       "low volume and small pool",
			goRoutines: runtime.NumCPU(),
			operations: 100,
			clientOpts: options.Client().SetMaxPoolSize(1).ApplyURI(uri),
		},
	}

	setupContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load the test data for the test case.
	collName, err := loadLargeCollection(setupContext, defaultLargeCollectionSize, options.Client().ApplyURI(uri))
	require.NoError(b, err)

	// Teardown collection.
	defer func() {
		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		require.NoError(b, err)

		coll := client.Database(defaultConnectionChurnDB).Collection(collName)

		err = coll.Drop(context.Background())
		require.NoError(b, err)
	}()

	// Load the 1st through 100th percentiles from concurrently sampling op time.
	latencyPercentiles, err := sampleLatency(setupContext, collName, options.Client().ApplyURI(uri))
	require.NoError(b, err)

	for _, tcase := range tests {
		b.Run(tcase.name, func(b *testing.B) {
			id := time.Now().Unix()

			results := make([]ccResult, len(latencyPercentiles))
			for i := range results {
				cfg := ccCfg{
					collName:   collName,
					goRoutines: uint(tcase.goRoutines),
					ops:        uint(tcase.operations),
				}

				results[i] = benchmarkConnectionChurnTO(b, id, latencyPercentiles[i], cfg, tcase.clientOpts)
				results[i].Percentile = float64(i) / 100.0 // Normalize the percentile
			}

			b.ReportMetric(calculateSimpsonsE(results), "E")
			b.ReportMetric(calculateThroughputPercentile(0.5, results), "T(0.5)")
			b.ReportMetric(calculateThroughputPercentile(0.01, results), "T(0.01)")

			recordCCThroughputResults(b, results)
		})
	}
}

// loadLargeCollection will dedicate a worker pool to inserting test data into
// an unindexed collection. Each record is 31 bytes in size.
func loadLargeCollection(ctx context.Context, size int, opts ...*options.ClientOptions) (string, error) {
	client, err := mongo.Connect(opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// Initialize a collection with the name "large<uuid>".
	collName := fmt.Sprintf("large%s", uuid.NewString())

	goRoutines := runtime.NumCPU()

	// Partition the volume into equal sizes per go routine. Use the floor if the
	// volume is not divisible by the number of goroutines.
	perGoroutine := size / goRoutines

	docs := make([]interface{}, perGoroutine)
	for i := range docs {
		docs[i] = bson.D{
			{Key: "field1", Value: rand.Int63()},
			{Key: "field2", Value: rand.Int31()},
		}
	}

	errs := make(chan error, goRoutines)
	done := make(chan struct{}, goRoutines)

	coll := client.Database(defaultConnectionChurnDB).Collection(collName)

	for i := 0; i < int(goRoutines); i++ {
		go func(i int) {
			_, err := coll.InsertMany(ctx, docs)
			if err != nil {
				errs <- fmt.Errorf("goroutine %v failed: %w", i, err)
			}

			done <- struct{}{}
		}(i)
	}

	go func() {
		defer close(errs)
		defer close(done)

		for i := 0; i < int(goRoutines); i++ {
			<-done
		}
	}()

	// Await errors and return the first error encountered.
	for err := range errs {
		if err != nil {
			return "", err
		}
	}

	return collName, nil
}

func TestLoadLargeCollection(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr string
	}{
		{
			name:    "size of 100",
			size:    100,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			collName, err := loadLargeCollection(ctx, tt.size)
			if err != nil {
				require.Equal(t, tt.wantErr, err.Error())

				return
			}

			assert.NotEmpty(t, collName)

			// Lookup the collection and assert the size is as expected.
			client, err := mongo.Connect()
			require.NoError(t, err)

			defer func() {
				err := client.Disconnect(context.Background())
				require.NoError(t, err)
			}()

			coll := client.Database(defaultConnectionChurnDB).Collection(collName)

			defer func() {
				err := coll.Drop(context.Background())
				require.NoError(t, err)
			}()

			count, err := coll.CountDocuments(context.Background(), bson.D{})
			require.NoError(t, err)

			assert.Equal(t, int64(tt.size), count)
		})
	}

}

// calculatePercentilesDuration calculates the 1st through 100th percentiles of a sample
func calculatePercentilesDuration(durations []time.Duration) []time.Duration {
	if len(durations) == 0 {
		return nil // Handle empty input
	}

	// Convert durations to float64 for processing
	sampleFloats := make([]float64, len(durations))
	for i, d := range durations {
		sampleFloats[i] = float64(d)
	}

	// Sort the sampleFloats array
	sort.Float64s(sampleFloats)

	percentiles := make([]time.Duration, 100)
	for i := 1; i <= 100; i++ {
		// Calculate the percentile using gonum/stat
		value := stat.Quantile(float64(i)/100, stat.Empirical, sampleFloats, nil)
		percentiles[i-1] = time.Duration(value)
	}

	return percentiles
}

// sampleLatency will concurrently run a find query {field1: doesnotexist} a
// pre-defined number of times greater than 100, recording the time each
// operation takes and returning the percetiles 1-100 of the sample data.
func sampleLatency(ctx context.Context, collName string, opts ...*options.ClientOptions) ([]time.Duration, error) {
	const sampleSize = 1000

	client, err := mongo.Connect(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	coll := client.Database(defaultConnectionChurnDB).Collection(collName)

	samples := make([]time.Duration, 0, sampleSize)

	goRoutines := runtime.NumCPU()

	errs := make(chan error, goRoutines)
	done := make(chan struct{}, goRoutines)

	query := bson.D{{Key: "field1", Value: "doesntexist"}}

	// Warm up the server:
	for i := 0; i < 100; i++ {
		coll.FindOne(context.Background(), query)
	}

	var samplesMu sync.Mutex
	for i := 0; i < int(goRoutines); i++ {
		go func() {
			// Higher durations yield more accurate statistics.
			durations := make([]time.Duration, 10)
			for i := 0; i < len(durations); i++ {
				start := time.Now()
				err := coll.FindOne(context.Background(), query).Err()
				durations[i] = time.Since(start)

				if err != nil && err != mongo.ErrNoDocuments {
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

		for i := 0; i < int(goRoutines); i++ {
			<-done
		}
	}()

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	samplesMu.Lock()
	defer samplesMu.Unlock()

	return calculatePercentilesDuration(samples), nil
}
