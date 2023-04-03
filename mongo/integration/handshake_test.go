package integration

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestHandshakeProse(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	opts := mtest.NewOptions().
		CreateCollection(false).
		ClientType(mtest.Proxy)

	clientMetadata := func(env bson.D) bson.D {
		elems := bson.D{
			{Key: "application", Value: bson.D{
				{Key: "name", Value: ""},
			}},
			{Key: "driver", Value: bson.D{
				{Key: "name", Value: "mongo-go-driver"},
				{Key: "version", Value: version.Driver},
			}},
			{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
				{Key: "architecture", Value: runtime.GOARCH},
			}},
		}

		// If env is empty, don't include it in the metadata.
		if env != nil && !reflect.DeepEqual(env, bson.D{}) {
			elems = append(elems, bson.E{Key: "env", Value: env})
		}

		elems = append(elems, bson.E{Key: "platform", Value: runtime.Version()})

		return elems
	}

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
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
				{Key: "memory_mb", Value: 1024},
				{Key: "region", Value: "us-east-2"},
			}),
		},
		{
			name: "valid Azure",
			env: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "azure.func"},
			}),
		},
		{
			name: "valid GCP",
			env: map[string]string{
				"K_SERVICE":            "servicename",
				"FUNCTION_MEMORY_MB":   "1024",
				"FUNCTION_TIMEOUT_SEC": "60",
				"FUNCTION_REGION":      "us-central1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "gcp.func"},
				{Key: "memory_mb", Value: 1024},
				{Key: "region", Value: "us-central1"},
				{Key: "timeout_sec", Value: 60},
			}),
		},
		{
			name: "valid Vercel",
			env: map[string]string{
				"VERCEL":        "1",
				"VERCEL_URL":    "*.vercel.app",
				"VERCEL_REGION": "cdg1",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "vercel"},
				{Key: "region", Value: "cdg1"},
				{Key: "url", Value: "*.vercel.app"},
			}),
		},
		{
			name: "invalid multiple providers",
			env: map[string]string{
				"AWS_EXECUTION_ENV":        "AWS_Lambda_java8",
				"FUNCTIONS_WORKER_RUNTIME": "node",
			},
			want: clientMetadata(nil),
		},
		{
			name: "invalid long string",
			env: map[string]string{
				"AWS_EXECUTION_ENV": "AWS_Lambda_java8",
				"AWS_REGION": func() string {
					var s string
					for i := 0; i < 512; i++ {
						s += "a"
					}
					return s
				}(),
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
		},
		{
			name: "invalid wrong types",
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "big",
			},
			want: clientMetadata(bson.D{
				{Key: "name", Value: "aws.lambda"},
			}),
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

				got, ok := clientVal.DocumentOK()
				assert.True(mt, ok, "expected client field to be a document, got %T", clientVal)

				wantBytes, err := bson.Marshal(test.want)
				assert.Nil(mt, err, "Marshal error for %v: %v", test.want, err)

				want := bsoncore.Document(wantBytes)

				wantElems, err := want.Elements()
				assert.Nil(mt, err, "error getting elements from want document: %v", err)

				gotElems, err := got.Elements()
				assert.Nil(mt, err, "error getting elements from got document: %v", err)

				if !reflect.DeepEqual(wantElems, gotElems) {
					mt.Errorf("expected %v, got %v", wantElems, gotElems)
				}
			}
		})
	}
}
