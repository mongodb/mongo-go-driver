package benchmark

import (
	"context"
)

func CanaryIncCase(ctx context.Context, tm TimerManager, iters int) error {
	var canaryCount int
	for i := 0; i < iters; i++ {
		canaryCount++
	}
	return nil
}

var globalCanaryCount int

func GlobalCanaryIncCase(ctx context.Context, tm TimerManager, iters int) error {
	for i := 0; i < iters; i++ {
		globalCanaryCount++
	}

	return nil
}
