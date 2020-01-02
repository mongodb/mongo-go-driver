package zstd

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	ec := m.Run()
	if ec == 0 && runtime.NumGoroutine() > 1 {
		n := 0
		for n < 60 {
			n++
			time.Sleep(time.Second)
			if runtime.NumGoroutine() == 1 {
				os.Exit(0)
			}
		}
		fmt.Println("goroutines:", runtime.NumGoroutine())
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		os.Exit(1)
	}
	os.Exit(ec)
}
