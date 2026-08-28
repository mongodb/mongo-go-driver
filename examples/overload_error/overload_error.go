// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	defaultMaxAttempts = 2

	baseBackoff = 100 * time.Millisecond
	maxBackoff  = 10_000 * time.Millisecond

	errSystemOverloadedError = "SystemOverloadedError"
	errRetryableError        = "RetryableError"
)

// isSystemOverloadedError detects overload errors
func isSystemOverloadedError(err error) bool {
	var lerr mongo.LabeledError
	return errors.As(err, &lerr) && lerr.HasErrorLabel(errSystemOverloadedError)
}

// serverBaseBackoff returns the base backoff that the server attached to the
// error as "baseBackoffMS", or 0 if the server did not supply one. A positive
// value replaces the client's default base backoff.
func serverBaseBackoff(err error) time.Duration {
	// For command errors, "baseBackoffMS" is a top-level field of the server
	// response.
	var cerr mongo.CommandError
	if errors.As(err, &cerr) {
		if ms, ok := cerr.Raw.Lookup("baseBackoffMS").AsInt64OK(); ok {
			return time.Duration(ms) * time.Millisecond
		}
	}

	// For write errors, "baseBackoffMS" is a field of the "writeConcernError"
	// subdocument, not of the top-level response.
	var wex mongo.WriteException
	if errors.As(err, &wex) && wex.WriteConcernError != nil {
		if ms, ok := wex.WriteConcernError.Raw.Lookup("baseBackoffMS").AsInt64OK(); ok {
			return time.Duration(ms) * time.Millisecond
		}
	}

	return 0
}

// overloadBackoff returns the backoff duration for the given retry attempt by
// doubling the base backoff once per attempt, capped at maxBackoff.
func overloadBackoff(base time.Duration, attempt int) time.Duration {
	d := base
	for i := 0; i < attempt && d < maxBackoff; i++ {
		d *= 2
	}
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

// jitterDuration returns the input duration weighted by a pseudo-random ratio
// in [0.0, 1.0).
func jitterDuration(d time.Duration) time.Duration {
	return time.Duration(float64(d) * rand.Float64())
}

// executeWithRetries executes the given function with retries if it returns a
// SystemOverloadedError.
func executeWithRetries[T any](
	ctx context.Context, maxAttempts int,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T
	var err error
	for attempts := 0; attempts < maxAttempts; attempts++ {
		isRetry := attempts > 0

		// The first attempt runs immediately. Every subsequent attempt waits
		// for an exponentially increasing backoff based on the error returned
		// by the previous attempt, with jitter.
		if isRetry {
			// Prefer the base backoff supplied by the server over the client's
			// default, if there is one.
			base := baseBackoff
			if serverBase := serverBaseBackoff(err); serverBase > 0 {
				base = serverBase
			}

			sleep := time.NewTimer(jitterDuration(overloadBackoff(base, attempts)))
			select {
			case <-ctx.Done():
				sleep.Stop()
				if err == nil {
					err = ctx.Err()
				}
				return result, err
			case <-sleep.C:
			}
		}

		result, err = fn(ctx)
		if err == nil {
			break
		}
		if !isSystemOverloadedError(err) {
			break
		}
		var lerr mongo.LabeledError
		if !errors.As(err, &lerr) || !lerr.HasErrorLabel(errRetryableError) {
			break
		}
	}
	return result, err
}

func main() {
	uri := os.Getenv("MONGODB_URI")
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	defer client.Disconnect(context.Background())

	ctx := context.Background()
	result, err := executeWithRetries(ctx, defaultMaxAttempts, func(ctx context.Context) ([]bson.D, error) {
		cursor, err := coll.Find(ctx, bson.D{})
		if err != nil {
			return nil, err
		}
		var res []bson.D
		err = cursor.All(ctx, &res)
		return res, err
	})
	if err != nil {
		log.Fatalf("Unhandled error: %v", err)
	}
	fmt.Printf("found %v\n", result)
}
