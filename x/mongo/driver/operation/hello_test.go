package operation

import (
	"os"
	"reflect"
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const bytesToStartDocument = 4

func TestHelloAppend(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		hello *Hello
		fn    func(dst []byte, max int32) ([]byte, error)
		dst   []byte
		max   int32
		want  bson.D
	}{
		{
			name: "appname empty",
			fn:   (&Hello{}).appendClientAppName,
			max:  bytesToStartDocument,
			want: bson.D{},
		},
		{
			name: "appname with 1 less than enough space for name",
			fn:   (&Hello{appname: "foo"}).appendClientAppName,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("application") +
				stringElementSize +
				len("name") +
				len("foo") - 1),
			want: bson.D{{Key: "application", Value: bson.D{}}},
		},
		{
			name: "appname with exact amount of space",
			fn:   (&Hello{appname: "foo"}).appendClientAppName,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("application") +
				stringElementSize +
				len("name") +
				len("foo")),
			want: bson.D{{Key: "application", Value: bson.D{
				{Key: "name", Value: "foo"},
			}}},
		},
		{
			name: "appname with more than enough space",
			fn:   (&Hello{appname: "foo"}).appendClientAppName,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("application") +
				stringElementSize +
				len("name") +
				len("foo") + 1),
			want: bson.D{{Key: "application", Value: bson.D{
				{Key: "name", Value: "foo"},
			}}},
		},
		{
			name: "driver empty",
			fn:   (&Hello{}).appendClientDriver,
			max:  bytesToStartDocument,
			want: bson.D{},
		},
		{
			name: "driver with 1 less than enough space for name",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName) - 1),
			want: bson.D{{Key: "driver", Value: bson.D{}}},
		},
		{
			name: "driver with exact amount of space for name",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName)),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
			}}},
		},
		{
			name: "driver with more than enough space for name",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName) + 1),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
			}}},
		},
		{
			name: "driver with 1 less than enough space for version",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName) +
				stringElementSize +
				len("version") +
				len(version.Driver) - 1),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
			}}},
		},
		{
			name: "driver with exact amount of space for version",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName) +
				stringElementSize +
				len("version") +
				len(version.Driver)),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
				{Key: "version", Value: version.Driver},
			}}},
		},
		{
			name: "driver with more than enough space for version",
			fn:   (&Hello{}).appendClientDriver,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("driver") +
				stringElementSize +
				len("name") +
				len(driverName) +
				stringElementSize +
				len("version") +
				len(version.Driver) + 1),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
				{Key: "version", Value: version.Driver},
			}}},
		},
		{
			name: "driver with more than enough space",
			fn:   (&Hello{}).appendClientDriver,
			max:  512,
			want: bson.D{
				{Key: "driver", Value: bson.D{
					{Key: "name", Value: driverName},
					{Key: "version", Value: version.Driver},
				}},
			},
		},
		{
			name: "env empty",
			fn:   (&Hello{}).appendClientEnv,
			max:  bytesToStartDocument,
			want: bson.D{},
		},
		{
			name: "os empty",
			fn:   (&Hello{}).appendClientOS,
			max:  bytesToStartDocument,
			want: bson.D{},
		},
		{
			name: "os without elements",
			fn:   (&Hello{}).appendClientOS,
			max: int32(docElementSize +
				bytesToStartDocument +
				len("os")),
			want: bson.D{{Key: "os", Value: bson.D{}}},
		},
		{
			name: "os with type",
			fn:   (&Hello{}).appendClientOS,
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("os") +
				stringElementSize +
				len("type") +
				len(runtime.GOOS)),
			want: bson.D{
				{Key: "os", Value: bson.D{
					{Key: "type", Value: runtime.GOOS},
				}},
			},
		},
		{
			name: "os with all elements",
			fn:   (&Hello{}).appendClientOS,
			max:  512,
			want: bson.D{
				{Key: "os", Value: bson.D{
					{Key: "type", Value: runtime.GOOS},
					{Key: "architecture", Value: runtime.GOARCH},
				}},
			},
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			idx, dst := bsoncore.AppendDocumentStart(test.dst)

			// Append the client metadata using the provided
			// function.
			dst, err := test.fn(dst, test.max)
			if err != nil {
				t.Fatalf("error appending client metadata: %v", err)
			}

			dst, err = bsoncore.AppendDocumentEnd(dst, idx)
			if err != nil {
				t.Fatalf("error appending document end: %v", err)
			}

			got, _, ok := bsoncore.ReadDocument(dst)
			if !ok {
				t.Fatalf("error reading document")
			}

			wantBytes, err := bson.Marshal(test.want)
			if err != nil {
				t.Fatalf("error marshaling want document: %v", err)
			}

			want := bsoncore.Document(wantBytes)

			// Get all of the want keys as a set.
			wantKeySet := make(map[string]struct{})

			wantElems, err := want.Elements()
			if err != nil {
				t.Fatalf("error getting elements from want document: %v", err)
			}

			for _, wantElem := range wantElems {
				wantKeySet[wantElem.Key()] = struct{}{}
			}

			// Compare element by element.
			gotElems, err := got.Elements()
			if err != nil {
				t.Fatalf("error getting elements from got document: %v", err)
			}

			if !reflect.DeepEqual(gotElems, wantElems) {
				t.Errorf("got %v, want %v", gotElems, wantElems)
			}
		})
	}

}

func TestHelloAppendClientEnv(t *testing.T) {
	for _, test := range []struct {
		name  string
		hello *Hello
		env   map[string]string
		dst   []byte
		max   int32
		want  bson.D
	}{
		{
			name:  "empty",
			hello: &Hello{},
			env:   map[string]string{},
			max:   bytesToStartDocument,
			want:  bson.D{},
		},
		{
			name:  "aws with 1 less than enough space for name",
			hello: &Hello{},
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			max: int32(bytesToStartDocument +
				docElementSize - 1 +
				len("env") +
				stringElementSize +
				len("name") +
				len(envNameAWSLambda) - 1),
			want: bson.D{},
		},
		{
			name:  "aws with exact space for name",
			hello: &Hello{},
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			max: int32(bytesToStartDocument +
				docElementSize +
				len("env") +
				stringElementSize +
				len("name") +
				len(envNameAWSLambda)),
			want: bson.D{
				{Key: "env", Value: bson.D{
					{Key: "name", Value: envNameAWSLambda},
				}},
			},
		},
		{
			name:  "aws with all elements",
			hello: &Hello{},
			env: map[string]string{
				"AWS_EXECUTION_ENV":               "AWS_Lambda_java8",
				"AWS_REGION":                      "us-east-2",
				"AWS_LAMBDA_FUNCTION_MEMORY_SIZE": "1024",
			},
			max: 512,
			want: bson.D{
				{Key: "env", Value: bson.D{
					{Key: "name", Value: envNameAWSLambda},
					{Key: "region", Value: "us-east-2"},
					{Key: "memory_mb", Value: 1024},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Set the environment variables.
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			idx, dst := bsoncore.AppendDocumentStart(test.dst)

			dst, err := test.hello.appendClientEnv(dst, test.max)
			if err != nil {
				t.Fatalf("error appending client metadata: %v", err)
			}

			dst, err = bsoncore.AppendDocumentEnd(dst, idx)
			if err != nil {
				t.Fatalf("error appending document end: %v", err)
			}

			got, _, ok := bsoncore.ReadDocument(dst)
			if !ok {
				t.Fatalf("error reading document")
			}

			wantBytes, err := bson.Marshal(test.want)
			if err != nil {
				t.Fatalf("error marshaling want document: %v", err)
			}

			want := bsoncore.Document(wantBytes)

			// Get all of the want keys as a set.
			wantElems, err := want.Elements()
			if err != nil {
				t.Fatalf("error getting elements from want document: %v", err)
			}

			// Compare element by element.
			gotElems, err := got.Elements()
			if err != nil {
				t.Fatalf("error getting elements from got document: %v", err)
			}

			if !reflect.DeepEqual(gotElems, wantElems) {
				t.Errorf("got %v, want %v", gotElems, wantElems)
			}
		})
	}
}

func TestParseFaasEnvName(t *testing.T) {
	// Reset the environment variables.
	os.Clearenv()

	for _, test := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "no env",
			want: "",
		},
		{
			name: "one aws",
			env: map[string]string{
				"AWS_EXECUTION_ENV": "hello",
			},
			want: "aws.lambda",
		},
		{
			name: "both aws options",
			env: map[string]string{
				"AWS_EXECUTION_ENV":      "hello",
				"AWS_LAMBDA_RUNTIME_API": "hello",
			},
			want: "aws.lambda",
		},
		{
			name: "multiple variables",
			env: map[string]string{
				"AWS_EXECUTION_ENV":       "hello",
				"FUNCTION_WORKER_RUNTIME": "hello",
			},
			want: "",
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			got := getFaasEnvName()
			if got != test.want {
				t.Errorf("parseFaasEnvName(%s) = %s, want %s",
					test.name, got, test.want)
			}
		})
	}
}
