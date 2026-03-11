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
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	tokenBucketCap = 10_000
	retryToken     = 10
	refreshToken   = 1

	maxAttempts = 5

	baseBackoff = 100 * time.Millisecond
	maxBackoff  = 10_000 * time.Millisecond

	errSystemOverloadedError = "SystemOverloadedError"
	errRetryableError        = "RetryableError"
)

// isSystemOverloadedError detects overload errors
func isSystemOverloadedError(err error) bool {
	if lerr, ok := err.(mongo.LabeledError); ok && lerr.HasErrorLabel(errSystemOverloadedError) {
		return true
	}
	return false
}

// tokenBucket is used to limit retries
type tokenBucket struct {
	capacity int
	tokens   int
	locker   *sync.Mutex
}

func newTokenBucket(capacity int) *tokenBucket {
	return &tokenBucket{
		capacity: capacity,
		tokens:   capacity,
		locker:   new(sync.Mutex),
	}
}

func (tb *tokenBucket) Consume(amount int) bool {
	tb.locker.Lock()
	defer tb.locker.Unlock()
	if amount > tb.tokens {
		return false
	}
	tb.tokens -= amount
	return true
}

func (tb *tokenBucket) Deposit(amount int) {
	tb.locker.Lock()
	defer tb.locker.Unlock()
	tb.tokens += amount
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
}

// executeWithRetries executes the given function with retries if it returns a
// SystemOverloadedError.
func executeWithRetries(
	ctx context.Context, tb *tokenBucket,
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
			tb.Deposit(refreshToken)
			if isRetry {
				tb.Deposit(retryToken)
			}
			break
		}
		if !isSystemOverloadedError(err) {
			tb.Deposit(retryToken)
			break
		}
		if lerr, ok := err.(mongo.LabeledError); !ok || !lerr.HasErrorLabel(errRetryableError) {
			break
		}
		if !tb.Consume(retryToken) {
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
	tb := newTokenBucket(tokenBucketCap)
	result, err := executeWithRetries(ctx, tb, func(ctx context.Context) (any, error) {
		cursor, err := coll.Find(ctx, bson.D{})
		if err != nil {
			return nil, err
		}
		var res []bson.D
		err = cursor.All(ctx, &res)
		return res, err
	})
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		log.Panic(err)
	}
	fmt.Printf("found %v\n", result)
}
