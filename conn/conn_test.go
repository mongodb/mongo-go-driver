package conn_test

import (
	"fmt"
	"testing"

	. "github.com/10gen/mongo-go-driver/conn"
	"github.com/kr/pretty"
)

func createIntegrationTestConnection() (Connection, error) {
	c, err := Dial(Endpoint(*host),
		WithAppName("mongo-go-driver-test"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err)
	}
	return c, nil
}

func TestBinaryTcpConnectionInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConnection()
	if err != nil {
		t.Error(err)
	}

	if subject.Desc() == nil {
		t.Error("connection's ServerInfo was nil")
	}

	t.Logf("Connection %s: %# v", subject, pretty.Formatter(subject.Desc()))
}
