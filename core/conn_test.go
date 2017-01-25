package core_test

import (
	"fmt"
	"testing"

	. "github.com/craiggwilson/mongo-go-driver/core"
	"github.com/craiggwilson/mongo-go-driver/core/msg"
	"github.com/kr/pretty"
)

func createIntegrationTestConnection() (Connection, error) {

	c, err := DialConnection(ConnectionOptions{
		AppName:        "mongo-go-driver-test",
		Codec:          msg.NewWireProtocolCodec(),
		Endpoint:       Endpoint(*host),
		EndpointDialer: DialEndpoint,
	})
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
