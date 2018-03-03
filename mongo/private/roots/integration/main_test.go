package integration

import (
	"flag"
	"os"
	"testing"
)

var host = flag.String("host", "127.0.0.1:27017", "specify the location of a running mongodb server.")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
