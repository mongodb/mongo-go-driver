// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	defaultOutputFileName = "perf.json"
	legacyHelloLowercase  = "ismaster"
	tarFile               = "perf.tar.gz"
	perfDir               = "perf"
	testdataURL           = "https://s3.amazonaws.com/boxes.10gen.com/build/driver-test-data.tar.gz"
	perftestDB            = "perftest"
	corpusColl            = "corpus"
	bsonDataDir           = "extended_bson"

	// Bson test data
	flatBSONData = "flat_bson.json"
	deepBSONData = "deep_bson.json"
	fullBSONData = "full_bson.json"

	// Single test data
	singleAndMultiDataDir = "single_and_multi_document"
	tweetData             = "tweet.json"
	smallData             = "small_doc.json"
	largeData             = "large_doc.json"

	// Reusable metric names
	opsPerSecondMaxName = "ops_per_second_max"
	opsPerSecondMinName = "ops_per_second_min"
	opsPerSecondMedName = "ops_per_second_med"
)

// fullRun will run all of the tests. This is default to false, since it's
// unlikely a user would need to run all tests locally. We still need to run
// the test up to the point where the performance data is downloaded.
var fullRun bool

func init() {
	flag.BoolVar(&fullRun, "fullRun", false, "run all benchmarks in TestRunAllBenchmarks")
}

type metrics struct {
	opsPerSecond []float64
}

func recordMetrics(b *testing.B, throughput *metrics, fn func(*testing.B)) {
	b.Helper()

	start := time.Now()

	fn(b)

	duration := time.Since(start)
	throughput.opsPerSecond = append(throughput.opsPerSecond, 1/duration.Seconds())
}

// calculate min, max, and median operations per second and report the metric
// on the benchmark object.
func reportThroughputStats(b *testing.B, times []float64) {
	sort.Float64s(times)

	b.ReportMetric(times[0], opsPerSecondMinName)
	b.ReportMetric(times[len(times)-1], opsPerSecondMaxName)

	var median float64
	if len(times)%2 == 0 {
		median = (times[len(times)/2-1] + times[len(times)/2]) / 2
	} else {
		median = times[len(times)/2]
	}

	b.ReportMetric(median, opsPerSecondMedName)
}

func reportMetrics(b *testing.B, metrics *metrics) {
	b.Helper()

	reportThroughputStats(b, metrics.opsPerSecond)
}

// find the testdata directory. We do this instead of hardcoding a relative path
// (i.e. ../../../testdata) so that these benchmarks can be run from both the
// root as a taskfile as well as from the benchmark directory.
func testdataDir(tb testing.TB) string {
	tb.Helper()

	wd, err := os.Getwd()
	require.NoError(tb, err, "failed to source working directory")

	for {
		tdPath := filepath.Join(wd, "testdata")
		if _, err := os.Stat(tdPath); !os.IsNotExist(err) {
			return tdPath
		}

		wd = filepath.Dir(wd)
		if filepath.Base(wd) == "mongodb-go-driver" {
			tb.Fatal("'testdata' directory not found")
		}
	}
}

// where to download the tarball
func testdataTarFileName(tb testing.TB) string {
	return filepath.Join(testdataDir(tb), tarFile)
}

// where to extract the tarball
func testdataPerfDir(tb testing.TB) string {
	return filepath.Join(testdataDir(tb), perfDir)
}

// download the tarball of test data to testdata/perf
func downloadTestDataTgz(t *testing.T) {
	resp, err := http.Get(testdataURL)
	require.NoError(t, err, "failed to get response from %q", testdataURL)

	defer resp.Body.Close()

	out, err := os.Create(testdataTarFileName(t))
	require.NoError(t, err, "failed to open testdata perf dir")

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err, "failed to copy response body to testdata perf dir")
}

// extract the tarball to the perf dir.
func extractTestDataTgz(t *testing.T) {
	tarPath := testdataTarFileName(t)
	defer func() { _ = os.Remove(tarPath) }()

	file, err := os.Open(tarPath)
	require.NoError(t, err, "failed to open tar file")

	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	require.NoError(t, err, "failed to create a gzip reader")

	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err, "failed to advance tar entry")

		targetPath := filepath.Join(testdataPerfDir(t), strings.TrimPrefix(header.Name, "data/"))

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(targetPath, 0755)
			require.NoError(t, err, "failed to extract dir from tgz")
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			require.NoError(t, err, "failed to create path to extract file from tgz")

			_, err = io.Copy(outFile, tarReader)
			require.NoError(t, err, "failed to extract file from tgz")

			outFile.Close()
		}
	}
}

func loadSourceDocument(b *testing.B, canonicalOnly bool, pathParts ...string) bson.D {
	b.Helper()

	data, err := os.ReadFile(filepath.Join(pathParts...))
	require.NoError(b, err, "failed to load source document")

	var doc bson.D

	err = bson.UnmarshalExtJSON(data, canonicalOnly, &doc)
	require.NoError(b, err, "failed to unmarshal source document")

	require.NotEmpty(b, doc)

	return doc
}

func benchmarkBSONEncoding(b *testing.B, canonicalOnly bool, source string) {
	doc := loadSourceDocument(b, canonicalOnly, testdataPerfDir(b), bsonDataDir, source)

	b.ResetTimer()

	metrics := new(metrics)

	for i := 0; i < b.N; i++ {
		recordMetrics(b, metrics, func(b *testing.B) {
			out, err := bson.Marshal(doc)
			require.NoError(b, err, "failed to encode flat bson data")

			require.NotEmpty(b, out)
		})
	}

	reportMetrics(b, metrics)
}

func benchmarkBSONDecoding(b *testing.B, canonicalOnly bool, source string) {
	doc := loadSourceDocument(b, canonicalOnly, testdataPerfDir(b), bsonDataDir, source)

	raw, err := bson.Marshal(doc)
	require.NoError(b, err, "failed to encode bson data")

	b.ResetTimer()

	metrics := new(metrics)

	for i := 0; i < b.N; i++ {
		recordMetrics(b, metrics, func(b *testing.B) {
			time.Sleep(100 * time.Millisecond)
			var out bson.D

			err := bson.Unmarshal(raw, &out)
			require.NoError(b, err, "failed to encode flat bson data")
		})
	}

	reportMetrics(b, metrics)
}

// Test driver performance encoding documents with top level key/value pairs
// involving the most commonly-used BSON types.
func BenchmarkBSONFlatDocumentEncoding(b *testing.B) {
	benchmarkBSONEncoding(b, true, flatBSONData)
}

// Test driver performance decoding documents with top level key/value pairs
// involving the most commonly-used BSON types.
func BenchmarkBSONFlatDocumentDecoding(b *testing.B) {
	benchmarkBSONDecoding(b, true, flatBSONData)
}

// Test driver performance encoding documents with deeply nested key/value pairs
// involving subdocuments, strings, integers, doubles and booleans.
func BenchmarkBSONDeepDocumentEncoding(b *testing.B) {
	benchmarkBSONEncoding(b, true, deepBSONData)
}

// Test driver performance decoding documents with deeply nested key/value pairs
// involving subdocuments, strings, integers, doubles and booleans.
func BenchmarkBSONDeepDocumentDecoding(b *testing.B) {
	benchmarkBSONDecoding(b, true, deepBSONData)
}

// Test driver performance encoding documents with top level key/value pairs
// involving the full range of BSON types.
func BenchmarkBSONFullDocumentEncoding(b *testing.B) {
	benchmarkBSONEncoding(b, false, fullBSONData)
}

// Test driver performance decoding documents with top level key/value pairs
// involving the full range of BSON types.
func BenchmarkBSONFullDocumentDecoding(b *testing.B) {
	benchmarkBSONDecoding(b, false, fullBSONData)
}

// Test driver performance sending a command to the database and reading a
// response.
func BenchmarkSingleRunCommand(b *testing.B) {
	coll, teardown := setupBench(b)
	defer teardown(b)

	cmd := bson.D{{Key: legacyHelloLowercase, Value: true}}

	metrics := new(metrics)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recordMetrics(b, metrics, func(b *testing.B) {
			b.Helper()

			var doc bson.D
			err := coll.Database().RunCommand(context.Background(), cmd).Decode(&doc)
			require.NoError(b, err)

			// read the document and then throw it away to prevent
			out, err := bson.Marshal(doc)
			require.NoError(b, err)

			require.NotEmpty(b, out)
		})
	}

	reportMetrics(b, metrics)
}

func setupBench(b *testing.B) (*mongo.Collection, func(b *testing.B)) {
	b.Helper()

	client, err := mongo.Connect()
	require.NoError(b, err, "failed to connect to server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	err = client.Ping(ctx, nil)
	require.NoError(b, err, "failed to ping the server")

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	db := client.Database(perftestDB)

	err = db.Drop(ctx)
	require.NoError(b, err, "failed to drop %q database", perftestDB)

	err = db.Collection(corpusColl).Drop(ctx)
	require.NoError(b, err, "failed to drop %q", corpusColl)

	err = db.CreateCollection(ctx, corpusColl)
	require.NoError(b, err, "failed to create %q collection", corpusColl)

	coll := db.Collection(corpusColl)

	return coll, func(b *testing.B) {
		err := client.Disconnect(context.Background())
		require.NoError(b, err, "failed to disconnect client")
	}
}

// Test driver performance sending an indexed query to the database and reading
// a single document in response.
func BenchmarkSingleFindOneByID(b *testing.B) {
	coll, teardown := setupBench(b)
	defer teardown(b)

	doc := loadSourceDocument(b, true, testdataPerfDir(b), singleAndMultiDataDir, tweetData)

	// Insert 10_000 documents into the corpus.
	const docCount = 10_000

	docsToInsert := make([]bson.D, docCount)
	for i := range docsToInsert {
		docsToInsert[i] = make(bson.D, 0, len(doc)+1)
		docsToInsert[i] = append(docsToInsert[i], bson.E{Key: "_id", Value: i})
		docsToInsert[i] = append(docsToInsert[i], doc...)
	}

	_, err := coll.InsertMany(context.Background(), docsToInsert)
	require.NoError(b, err, "failed to insert corpus")

	metrics := new(metrics)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recordMetrics(b, metrics, func(b *testing.B) {
			err := coll.FindOne(context.Background(), bson.D{{Key: "_id", Value: i % docCount}}).Decode(&bson.D{})
			require.NoError(b, err, "failed to find one")
		})
	}

	reportMetrics(b, metrics)
}

func benchmarkSingleInsert(b *testing.B, source string) {
	b.Helper()

	coll, teardown := setupBench(b)
	defer teardown(b)

	doc := loadSourceDocument(b, true, testdataPerfDir(b), singleAndMultiDataDir, smallData)

	metrics := new(metrics)

	for i := 0; i < b.N; i++ {
		recordMetrics(b, metrics, func(b *testing.B) {
			b.Helper()
			_, err := coll.InsertOne(context.Background(), doc)
			require.NoError(b, err, "failed to insert small doc")
		})
	}

	reportMetrics(b, metrics)
}

// Test driver performance inserting a single, small document to the database.
func BenchmarkSmallDocInsertOne(b *testing.B) {
	benchmarkSingleInsert(b, smallData)
}

// Test driver performance inserting a single, large document to the database.
func BenchmarkLargeDocInsertOne(b *testing.B) {
	benchmarkSingleInsert(b, largeData)
}

// Test driver performance retrieving multiple documents from a query.
func BenchmarkMultiFindMany(b *testing.B) {
	coll, teardown := setupBench(b)
	defer teardown(b)

	doc := loadSourceDocument(b, true, testdataPerfDir(b), singleAndMultiDataDir, tweetData)

	docsToInsert := make([]bson.D, b.N)
	for i := range docsToInsert {
		docsToInsert[i] = doc
	}

	_, err := coll.InsertMany(context.Background(), docsToInsert)
	require.NoError(b, err, "failed to insert docs")

	b.ResetTimer()

	cursor, err := coll.Find(context.Background(), bson.D{})
	require.NoError(b, err, "failed to find data")

	defer cursor.Close(context.Background())

	metrics := new(metrics)

	counter := 0
	for cursor.Next(context.Background()) {
		recordMetrics(b, metrics, func(b *testing.B) {
			err = cursor.Err()
			require.NoError(b, err, "failed to advance cursor")

			if len(cursor.Current) == 0 {
				b.Fatalf("error retrieving document")
			}

			counter++
		})
	}

	if counter != b.N {
		b.Fatalf("problem iterating cursors")
	}

	err = cursor.Close(context.Background())
	require.NoError(b, err, "failed to close cursor")

	reportMetrics(b, metrics)
}

func benchmarkMultiInsert(b *testing.B, source string) {
	b.Helper()

	coll, teardown := setupBench(b)
	defer teardown(b)

	doc := loadSourceDocument(b, true, testdataPerfDir(b), singleAndMultiDataDir, source)

	docsToInsert := make([]bson.D, b.N)
	for i := range docsToInsert {
		docsToInsert[i] = doc
	}

	b.ResetTimer()

	res, err := coll.InsertMany(context.Background(), docsToInsert)
	require.NoError(b, err, "failed to insert many")

	require.Len(b, res.InsertedIDs, b.N)
}

// Test driver performance inserting multiple, small documents to the database.
func BenchmarkMultiInsertSmallDocument(b *testing.B) {
	benchmarkMultiInsert(b, smallData)
}

// Test driver performance inserting multiple, large documents to the database.
func BenchmarkMultiInsertLargeDocument(b *testing.B) {
	benchmarkMultiInsert(b, largeData)
}

func runBenchmark(name string, fn func(*testing.B)) (poplarTest, error) {
	test := poplarTest{
		ID:        fmt.Sprintf("%d", time.Now().UnixMilli()),
		CreatedAt: time.Now(),
	}

	result := testing.Benchmark(fn)

	if result.N == 0 {
		return test, fmt.Errorf("benchmark failed to run")
	}

	test.CompletedAt = test.CreatedAt.Add(result.T)

	test.Metrics = []poplarTestMetrics{
		{Name: "total_time_seconds", Type: "SUM", Value: result.T.Seconds()},
		{Name: "iterations", Type: "SUM", Value: result.N},
		{Name: "allocated_bytes_per_op", Type: "MEAN", Value: result.AllocedBytesPerOp()},
		{Name: "allocs_per_op", Type: "MEAN", Value: result.AllocsPerOp()},
		{Name: "total_mem_allocs", Type: "SUM", Value: result.MemAllocs},
		{Name: "total_bytes_allocated", Type: "SUM", Value: result.MemBytes},
		{Name: "ns_per_op", Type: "MEAN", Value: result.NsPerOp()},
	}

	// Only tests that set bytes in the benchmark will have this metric.
	if result.Bytes != 0 {
		megaBytesPerOp := (float64(result.Bytes) / 1024 / 1024) / float64(result.NsPerOp()) * 1e9

		test.Metrics = append(test.Metrics,
			poplarTestMetrics{Name: "megabytes_per_second", Type: "THROUGHPUT", Value: megaBytesPerOp})
	}

	if opsPerSecondMin := result.Extra[opsPerSecondMinName]; opsPerSecondMin != 0 {
		test.Metrics = append(test.Metrics,
			poplarTestMetrics{Name: opsPerSecondMinName, Type: "THROUGHPUT", Value: opsPerSecondMin})
	}

	if opsPerSecondMax := result.Extra[opsPerSecondMaxName]; opsPerSecondMax != 0 {
		test.Metrics = append(test.Metrics,
			poplarTestMetrics{Name: opsPerSecondMaxName, Type: "THROUGHPUT", Value: opsPerSecondMax})
	}

	if opsPerSecondMed := result.Extra[opsPerSecondMedName]; opsPerSecondMed != 0 {
		test.Metrics = append(test.Metrics,
			poplarTestMetrics{Name: opsPerSecondMedName, Type: "THROUGHPUT", Value: opsPerSecondMed})
	}

	test.Info = poplarTestInfo{
		TestName: name,
	}

	return test, nil
}

func TestRunAllBenchmarks(t *testing.T) {
	flag.Parse()

	// Download and extract the data if it doesn't exist.
	if _, err := os.Stat(testdataPerfDir(t)); os.IsNotExist(err) {
		downloadTestDataTgz(t)
		extractTestDataTgz(t)
	}

	// The test will be run any time a benchmark is run. To avoid running all
	// benchmarks in this case, we require a flag that defaults to false. The
	// intention is that this test will only ever need to fully run in CI.
	if !fullRun {
		return
	}

	// Run the cases and accumulate the results.
	cases := []struct {
		name      string
		benchmark func(*testing.B)
	}{
		{name: "BenchmarkBSONFlatDocumentEncoding", benchmark: BenchmarkBSONFlatDocumentEncoding},
		{name: "BenchmarkBSONFlatDocumentDecoding", benchmark: BenchmarkBSONFlatDocumentDecoding},
		{name: "BenchmarkBSONDeepDocumentEncoding", benchmark: BenchmarkBSONDeepDocumentEncoding},
		{name: "BenchmarkBSONDeepDocumentDecoding", benchmark: BenchmarkBSONDeepDocumentDecoding},
		{name: "BenchmarkBSONFullDocumentEncoding", benchmark: BenchmarkBSONFullDocumentEncoding},
		{name: "BenchmarkBSONFullDocumentDecoding", benchmark: BenchmarkBSONFullDocumentDecoding},
		{name: "BenchmarkSingleRunCommand", benchmark: BenchmarkSingleRunCommand},
		{name: "BenchmarkSingleFindOneByID", benchmark: BenchmarkSingleFindOneByID},
		{name: "BenchmarkSmallDocInsertOne", benchmark: BenchmarkSmallDocInsertOne},
		{name: "BenchmarkLargeDocInsertOne", benchmark: BenchmarkLargeDocInsertOne},
		{name: "BenchmarkMultiFindMany", benchmark: BenchmarkMultiFindMany},
		{name: "BenchmarkMultiInsertSmallDocument", benchmark: BenchmarkMultiInsertSmallDocument},
		{name: "BenchmarkMultiInsertLargeDocument", benchmark: BenchmarkMultiInsertLargeDocument},
	}

	results := make([]poplarTest, len(cases))
	for i := range cases {
		t.Run(cases[i].name, func(t *testing.T) {
			var err error

			results[i], err = runBenchmark(cases[i].name, cases[i].benchmark)
			if err != nil { // Avoid asserting to prevent failures on single-run benchmarks
				t.Logf("failed to run benchmark: %v", err)
			}
		})
	}

	// Write the results to the performance file.
	evgOutput, err := json.MarshalIndent(results, "", "   ")
	require.NoError(t, err, "failed to encode results")

	evgOutput = append(evgOutput, []byte("\n")...)

	// Ignore gosec warning "Expect WriteFile permissions to be 0600 or less" for
	// benchmark result file.
	/* #nosec G306 */
	err = os.WriteFile(filepath.Join(filepath.Dir(testdataDir(t)), defaultOutputFileName), evgOutput, 0644)
	require.NoError(t, err, "failed to write results")
}

// poplarTest was copied from
// https://github.com/evergreen-ci/poplar/blob/8d03d2bacde0897cedd73ed79ddc167ed1ed4c77/report.go#L38
type poplarTest struct {
	ID          string               `bson:"_id" json:"id" yaml:"id"`
	Info        poplarTestInfo       `bson:"info" json:"info" yaml:"info"`
	CreatedAt   time.Time            `bson:"created_at" json:"created_at" yaml:"created_at"`
	CompletedAt time.Time            `bson:"completed_at" json:"completed_at" yaml:"completed_at"`
	Artifacts   []poplarTestArtifact `bson:"artifacts" json:"artifacts" yaml:"artifacts"`
	Metrics     []poplarTestMetrics  `bson:"metrics" json:"metrics" yaml:"metrics"`
	SubTests    []poplarTest         `bson:"sub_tests" json:"sub_tests" yaml:"sub_tests"`
}

// poplarTestInfo was copied from
// https://github.com/evergreen-ci/poplar/blob/8d03d2bacde0897cedd73ed79ddc167ed1ed4c77/report.go#L52
type poplarTestInfo struct {
	TestName  string           `bson:"test_name" json:"test_name" yaml:"test_name"`
	Trial     int              `bson:"trial" json:"trial" yaml:"trial"`
	Parent    string           `bson:"parent" json:"parent" yaml:"parent"`
	Tags      []string         `bson:"tags" json:"tags" yaml:"tags"`
	Arguments map[string]int32 `bson:"args" json:"args" yaml:"args"`
}

// poplarTestArtifact was copied from
// https://github.com/evergreen-ci/poplar/blob/8d03d2bacde0897cedd73ed79ddc167ed1ed4c77/report.go#L62
type poplarTestArtifact struct {
	Bucket                string    `bson:"bucket" json:"bucket" yaml:"bucket"`
	Prefix                string    `bson:"prefix" json:"prefix" yaml:"prefix"`
	Permissions           string    `bson:"permissions" json:"permissions" yaml:"permissions"`
	Path                  string    `bson:"path" json:"path" yaml:"path"`
	Tags                  []string  `bson:"tags" json:"tags" yaml:"tags"`
	CreatedAt             time.Time `bson:"created_at" json:"created_at" yaml:"created_at"`
	LocalFile             string    `bson:"local_path,omitempty" json:"local_path,omitempty" yaml:"local_path,omitempty"`
	PayloadTEXT           bool      `bson:"is_text,omitempty" json:"is_text,omitempty" yaml:"is_text,omitempty"`
	PayloadFTDC           bool      `bson:"is_ftdc,omitempty" json:"is_ftdc,omitempty" yaml:"is_ftdc,omitempty"`
	PayloadBSON           bool      `bson:"is_bson,omitempty" json:"is_bson,omitempty" yaml:"is_bson,omitempty"`
	PayloadJSON           bool      `bson:"is_json,omitempty" json:"is_json,omitempty" yaml:"is_json,omitempty"`
	PayloadCSV            bool      `bson:"is_csv,omitempty" json:"is_csv,omitempty" yaml:"is_csv,omitempty"`
	DataUncompressed      bool      `bson:"is_uncompressed" json:"is_uncompressed" yaml:"is_uncompressed"`
	DataGzipped           bool      `bson:"is_gzip,omitempty" json:"is_gzip,omitempty" yaml:"is_gzip,omitempty"`
	DataTarball           bool      `bson:"is_tarball,omitempty" json:"is_tarball,omitempty" yaml:"is_tarball,omitempty"`
	EventsRaw             bool      `bson:"events_raw,omitempty" json:"events_raw,omitempty" yaml:"events_raw,omitempty"`
	EventsHistogram       bool      `bson:"events_histogram,omitempty" json:"events_histogram,omitempty" yaml:"events_histogram,omitempty"`
	EventsIntervalSummary bool      `bson:"events_interval_summary,omitempty" json:"events_interval_summary,omitempty" yaml:"events_interval_summary,omitempty"`
	EventsCollapsed       bool      `bson:"events_collapsed,omitempty" json:"events_collapsed,omitempty" yaml:"events_collapsed,omitempty"`
	ConvertGzip           bool      `bson:"convert_gzip,omitempty" json:"convert_gzip,omitempty" yaml:"convert_gzip,omitempty"`
	ConvertBSON2FTDC      bool      `bson:"convert_bson_to_ftdc,omitempty" json:"convert_bson_to_ftdc,omitempty" yaml:"convert_bson_to_ftdc,omitempty"`
	ConvertJSON2FTDC      bool      `bson:"convert_json_to_ftdc" json:"convert_json_to_ftdc" yaml:"convert_json_to_ftdc"`
	ConvertCSV2FTDC       bool      `bson:"convert_csv_to_ftdc" json:"convert_csv_to_ftdc" yaml:"convert_csv_to_ftdc"`
}

// poplarTestMetrics was copied from
// https://github.com/evergreen-ci/poplar/blob/8d03d2bacde0897cedd73ed79ddc167ed1ed4c77/report.go#L124
type poplarTestMetrics struct {
	Name    string      `bson:"name" json:"name" yaml:"name"`
	Version int         `bson:"version,omitempty" json:"version,omitempty" yaml:"version,omitempty"`
	Type    string      `bson:"type" json:"type" yaml:"type"`
	Value   interface{} `bson:"value" json:"value" yaml:"value"`
}
