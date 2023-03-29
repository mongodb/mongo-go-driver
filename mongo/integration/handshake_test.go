package integration

import (
	"context"
	"os"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	opts := mtest.NewOptions().
		CreateCollection(false).
		ClientType(mtest.Proxy)

	for _, test := range []struct {
		name string
		env  map[string]string
		want bson.D
	}{
		{
			name: "valid AWS",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			want: bson.D{
				{Key: "name", Value: "aws.lambda"},
				{Key: "region", Value: "us-east-2"},
				{Key: "memory_mb", Value: 1024},
			},
		},
		{
			name: "valid Azure",
			env: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			want: bson.D{
				{Key: "name", Value: "azure.func"},
			},
		},
	} {
		test := test

		mt.RunOpts(test.name, opts, func(mt *mtest.T) {
			for k, v := range test.env {
				mt.Setenv(k, v)
			}

			// Ping the server to ensure the handshake has completed.
			err := mt.Client.Ping(context.Background(), nil)
			assert.Nil(mt, err, "Ping error: %v", err)

			messages := mt.GetProxiedMessages()

			// First two messages are handshake messages
			for idx, pair := range messages[:2] {
				hello := internal.LegacyHello
				//  Expect "hello" command name with API version.
				if os.Getenv("REQUIRE_API_VERSION") == "true" {
					hello = "hello"
				}

				assert.Equal(mt, pair.CommandName, hello, "expected command name %s at index %d, got %s", hello, idx,
					pair.CommandName)

				sent := pair.Sent

				// Lookup the "client" field in the command document.
				clientVal, err := sent.Command.LookupErr("client")
				assert.Nil(mt, err, "expected command %s at index %d to contain client field", sent.Command, idx)

				// Lookup the "env" field in the "client" document.
				envVal, err := clientVal.Document().LookupErr("env")
				assert.Nil(mt, err, "expected command %s at index %d to contain env field", sent.Command, idx)

				got, ok := envVal.DocumentOK()
				assert.True(mt, ok, "expected env field to be a document, got %T", envVal)

				wantBytes, err := bson.Marshal(test.want)
				assert.Nil(mt, err, "Marshal error for %v: %v", test.want, err)

				want := bsoncore.Document(wantBytes)

				// Get all of the want keys as a set.
				wantElems, err := want.Elements()
				assert.Nil(mt, err, "error getting elements from want document: %v", err)

				// Compare element by element.
				gotElems, err := got.Elements()
				assert.Nil(mt, err, "error getting elements from got document: %v", err)

				if !reflect.DeepEqual(gotElems, wantElems) {
					mt.Fatalf("expected env document %v, got %v", wantElems, gotElems)
				}
			}
		})
	}
}
