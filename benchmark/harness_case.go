package benchmark

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type CaseDefinition struct {
	Bench   BenchCase
	Count   int
	Size    int
	Runtime time.Duration

	startAt time.Time
}

// TimerManager is a subset of the testing.B tool, used to manage
// setup code.
type TimerManager interface {
	ResetTimer()
	StartTimer()
	StopTimer()
}

func (c *CaseDefinition) ResetTimer() {
	c.startAt = time.Now()
	c.Runtime = 0
}

func (c *CaseDefinition) StartTimer() {
	c.startAt = time.Now()
}

func (c *CaseDefinition) StopTimer() {
	if c.startAt.IsZero() {
		return
	}
	c.Runtime += time.Since(c.startAt)
	c.startAt = time.Time{}
}

func (c *CaseDefinition) roundedRuntime() time.Duration {
	return roundDurationMS(c.Runtime)
}

func (c *CaseDefinition) Run(ctx context.Context) *BenchResult {
	out := &BenchResult{
		Trials:     1,
		DataSize:   c.Size,
		Name:       c.Name(),
		Operations: c.Count,
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, ExecutionTimeout)
	defer cancel()

	fmt.Println("=== RUN", out.Name)
	c.startAt = time.Now()

	for {
		if time.Since(c.startAt) > c.Runtime {
			if out.Trials >= MinIterations {
				break
			} else if ctx.Err() != nil {
				break
			}
		}
		res := Result{
			Iterations: c.Count,
		}

		c.StartTimer()
		res.Error = c.Bench(ctx, c, c.Count)
		c.StopTimer()
		res.Duration = c.Runtime

		if res.Error == context.Canceled {
			break
		}

		out.Trials++
		out.Raw = append(out.Raw, res)
	}

	out.Duration = out.totalDuration()
	if out.HasErrors() {
		fmt.Printf("--- FAIL: %s (%s)\n", out.Name, out.roundedRuntime())
	} else {
		fmt.Printf("--- PASS: %s (%s)\n", out.Name, out.roundedRuntime())
	}

	return out

}

func (c *CaseDefinition) String() string {
	return fmt.Sprintf("name=%s, count=%d, runtime=%s timeout=%s",
		c.Name(), c.Count, c.Runtime, ExecutionTimeout)
}

func (c *CaseDefinition) Name() string { return getName(c.Bench) }
func getName(i interface{}) string {
	n := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(n, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return n

}

func getProjectRoot() string { return filepath.Dir(getDirectoryOfFile()) }

func getDirectoryOfFile() string {
	_, file, _, _ := runtime.Caller(1)

	return filepath.Dir(file)
}
