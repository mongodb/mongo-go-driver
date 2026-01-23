// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse

package mongocrypt

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mongocrypt/options"
)

// load JSON for benchmark
func resourceToDocumentForBench(b *testing.B, filename string) bsoncore.Document {
	b.Helper()

	content, err := os.ReadFile(path.Join(resourcesDir, filename))
	require.NoError(b, err)

	var doc bsoncore.Document

	err = bson.UnmarshalExtJSON(content, false, &doc)
	require.NoError(b, err)

	return doc
}

// Add encryption key to the encryption context.
func addMongoKeysForBench(b *testing.B, encryptCtx *Context) {
	b.Helper()

	if encryptCtx.State() != NeedMongoKeys {
		return
	}

	_, err := encryptCtx.NextOperation()
	require.NoError(b, err)

	// feed result and finish op
	err = encryptCtx.AddOperationResult(resourceToDocumentForBench(b, "local-key-document.json"))
	require.NoError(b, err)

	err = encryptCtx.CompleteOperation()
	require.NoError(b, err)
}

// encrypt a document for benchmarking
func createEncryptedDocForBench(b *testing.B, crypt *MongoCrypt, iter int) bsoncore.Document {
	b.Helper()

	const algorithm = "AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic"

	// create explicit encryption context and check initial state
	keyID := bson.Binary{
		Subtype: 0x04, // 0x04 is UUID subtype
		Data:    []byte("aaaaaaaaaaaaaaaa"),
	}

	encryptOpts := &options.ExplicitEncryptionOptions{Algorithm: algorithm, KeyID: &keyID}
	doc := bsoncore.NewDocumentBuilder().AppendString("v", fmt.Sprintf("value %04v", iter)).Build()

	encryptCtx, err := crypt.CreateExplicitEncryptionContext(doc, encryptOpts)
	require.NoError(b, err)

	defer encryptCtx.Close()

	addMongoKeysForBench(b, encryptCtx)

	// perform final encryption
	encryptedDoc, err := encryptCtx.Finish()
	require.NoError(b, err)

	return encryptedDoc
}

// decrypt an encrypted document for benchmarking.
func decryptDocForBench(b *testing.B, crypt *MongoCrypt, encryptedDoc bsoncore.Document) bsoncore.Document {
	b.Helper()

	// create explicit decryption context and check initial state
	decryptCtx, err := crypt.CreateDecryptionContext(encryptedDoc)
	require.NoError(b, err)

	defer decryptCtx.Close()

	// perform final decryption
	decryptedDoc, err := decryptCtx.Finish()
	require.NoError(b, err)

	return decryptedDoc
}

// create a document of the form:
// { "key0001": <encrypted "value 0001">, "key0002": <encrypted "value 0002">, ... }
func createEncryptedDocForBulkDecryptionBench(b *testing.B, crypt *MongoCrypt, count int) bsoncore.Document {
	bldr := bsoncore.NewDocumentBuilder()

	for i := 0; i < count; i++ {
		encDoc := createEncryptedDocForBench(b, crypt, i)
		bldr.AppendValue(fmt.Sprintf("key%04v", i), encDoc.Lookup("v"))
	}

	return bldr.Build()
}

// Create a MongoCrypt object for benchmarking bulk decryption.
func newCryptForBench(b *testing.B) *MongoCrypt {
	key := []byte{
		0x9d, 0x94, 0x4b, 0x0d, 0x93, 0xd0, 0xc5, 0x44,
		0xa5, 0x72, 0xfd, 0x32, 0x1b, 0x94, 0x30, 0x90,
		0x23, 0x35, 0x73, 0x7c, 0xf0, 0xf6, 0xc2, 0xf4,
		0xda, 0x23, 0x56, 0xe7, 0x8f, 0x04, 0xcc, 0xfa,
		0xde, 0x75, 0xb4, 0x51, 0x87, 0xf3, 0x8b, 0x97,
		0xd7, 0x4b, 0x44, 0x3b, 0xac, 0x39, 0xa2, 0xc6,
		0x4d, 0x91, 0x00, 0x3e, 0xd1, 0xfa, 0x4a, 0x30,
		0xc1, 0xd2, 0xc6, 0x5e, 0xfb, 0xac, 0x41, 0xf2,
		0x48, 0x13, 0x3c, 0x9b, 0x50, 0xfc, 0xa7, 0x24,
		0x7a, 0x2e, 0x02, 0x63, 0xa3, 0xc6, 0x16, 0x25,
		0x51, 0x50, 0x78, 0x3e, 0x0f, 0xd8, 0x6e, 0x84,
		0xa6, 0xec, 0x8d, 0x2d, 0x24, 0x47, 0xe5, 0xaf,
	}

	localProvider := bsoncore.NewDocumentBuilder().
		AppendBinary("key", 0, key).
		Build()

	kmsProviders := bsoncore.NewDocumentBuilder().
		AppendDocument("local", localProvider).
		Build()

	cryptOpts := &options.MongoCryptOptions{
		KmsProviders: kmsProviders,
	}

	crypt, err := NewMongoCrypt(cryptOpts)

	require.NoError(b, err)
	assert.NotNil(b, crypt)

	return crypt
}

func calcMedian(values []float64) float64 {
	sort.Float64s(values)

	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2.0
	}

	return values[(n-1)/2]
}

func BenchmarkBulkDecryption(b *testing.B) {
	const numKeys = 1500
	const repeatCount = 10

	// Create crypt that uses a data key with the "local" KMS provider.
	crypt := newCryptForBench(b)
	defer crypt.Close()

	// Encrypt 1500 string values of the form value 0001, value 0002, value 0003,
	// ... with the algorithm AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic.
	encryptedDoc := createEncryptedDocForBulkDecryptionBench(b, crypt, numKeys)

	// decrypt the document repeatedly for one second to warm up the benchmark.
	for start := time.Now(); time.Since(start) < time.Second; {
		decryptDocForBench(b, crypt, encryptedDoc)
	}

	benchmarks := []struct {
		threads int
	}{
		{threads: 1},
		{threads: 2},
		{threads: 8},
		{threads: 64},
	}

	// Run the benchmark. Repeat benchmark for thread counts: (1, 2, 8, 64).
	// Repeat 10 times.
	for _, bench := range benchmarks {
		var opsPerSec []float64

		b.Run(fmt.Sprintf("threadCount=%v", bench.threads), func(b *testing.B) {
			runtime.GOMAXPROCS(bench.threads)

			for i := 0; i < repeatCount; i++ {
				b.ResetTimer()
				b.ReportAllocs()

				startTime := time.Now()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						decryptDocForBench(b, crypt, encryptedDoc)
					}
				})

				opsPerSec = append(opsPerSec, float64(b.N)/time.Now().Sub(startTime).Seconds())
			}
		})

		b.Logf("thread count: %v, median ops/sec: %.2f\n", bench.threads, calcMedian(opsPerSec))
	}
}
