package main

import (
	"os"

	"github.com/mongodb/mongo-go-driver/benchmark"
)

func main() {
	os.Exit(benchmark.DriverBenchmarkMain())
}
