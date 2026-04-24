// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package examples

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

// executeWithRetries executes the given function with retries if it returns a
// SystemOverloadedError.
func executeWithRetries(
	ctx context.Context, maxAttempts int,
	fn func(ctx context.Context) (any, error),
) (any, error) {
	var result any
	var err error
	expDur := baseBackoff
	for attempts := 0; attempts < maxAttempts; attempts++ {
		isRetry := attempts > 0

		if isRetry {
			if expDur > maxBackoff {
				expDur = maxBackoff
			}
			delay := expDur * time.Duration(rand.Int63n(512)) / 512
			sleep := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				sleep.Stop()
				if err == nil {
					err = ctx.Err()
				}
				return result, err
			case <-sleep.C:
			}
			if expDur < maxBackoff {
				expDur = expDur * 2
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

// ExampleOverloadError_Find demonstrates how to use executeWithRetries to retry a Find operation
// if it returns a SystemOverloadedError.
func ExampleOverloadError_Find() {
	var coll *mongo.Collection

	ctx := context.Background()
	result, err := executeWithRetries(ctx, defaultMaxAttempts, func(ctx context.Context) (any, error) {
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
