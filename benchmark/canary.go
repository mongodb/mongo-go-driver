package benchmark

import (
	"context"
)

func CanaryIncCase(ctx context.Context, iterations int) error {
	var canaryCount int
	for i := 0; i < iterations; i++ {
		canaryCount++
	}
	return nil
}

var globalCanaryCount int

func GlobalCanaryIncCase(ctx context.Context, iters int) error {
	for i := 0; i < iters; i++ {
		globalCanaryCount++
	}

	return nil
}
