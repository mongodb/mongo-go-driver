package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	DefaultHTTPGetAddress = "http://127.0.0.1:12345"

	_, err := handler(events.APIGatewayProxyRequest{})
	if err != nil {
		t.Fatal(err)
	}
}
