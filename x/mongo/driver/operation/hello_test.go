package operation

import (
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const documentSize = 5         // 5 bytes to start and end a document
const embeddedDocumentSize = 7 // 7 bytes to append a document element
const stringElementSize = 7    // 7 bytes to append a string element
const int32ElementSize = 6     // 6 bytes to append an int32 element

func assertAppendClientMaxLen(t *testing.T, got bsoncore.Document, wantD bson.D, maxLen int) {
	t.Helper()

	tooLarge := len(got)-documentSize > maxLen
	require.False(t, tooLarge, "got document is too large: %v", got)

	var gotD bson.D
	err := bson.Unmarshal(got, &gotD)
	require.NoError(t, err, "error unmarshaling got document: %v", err)

	assert.Equal(t, wantD, gotD, "got %v, want %v", gotD, wantD)
}

func encodeWithCallback(t *testing.T, cb func(int, []byte) ([]byte, error)) bsoncore.Document {
	t.Helper()

	var err error
	idx, dst := bsoncore.AppendDocumentStart(nil)

	dst, err = cb(len(dst), dst)
	require.NoError(t, err, "error appending client metadata: %v", err)

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	require.NoError(t, err, "error appending document end: %v", err)

	got, _, ok := bsoncore.ReadDocument(dst)
	require.True(t, ok, "error reading document: %v", got)

	return got
}

func lenStringElem(key, name string) int {
	return stringElementSize + len(key) + len(name)
}

func TestAppendClientAppName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appname string
		maxLen  int
		want    bson.D
	}{
		{
			name:   "empty",
			maxLen: 0,
			want:   bson.D{},
		},
		{
			name: "1 less than enough space",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "application")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{},
		},
		{
			name:    "1 less than enough space for name",
			appname: "foo",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "application")
				dst = bsoncore.AppendStringElement(dst, "name", "foo")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name:    "exact amount of space for name",
			appname: "foo",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "application")
				dst = bsoncore.AppendStringElement(dst, "name", "foo")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "application", Value: bson.D{{Key: "name", Value: "foo"}}}},
		},
		{
			name:    "1 more than enough space for name",
			appname: "foo",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "application")
				dst = bsoncore.AppendStringElement(dst, "name", "foo")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "application", Value: bson.D{{Key: "name", Value: "foo"}}}},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cb := func(n int, dst []byte) ([]byte, error) {
				// Buffer the maxLen by the number of bytes
				// written so far.
				maxLen := test.maxLen + n

				var err error
				dst, err = appendClientAppName(dst, maxLen, test.appname)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertAppendClientMaxLen(t, got, test.want, test.maxLen)
		})
	}
}

func TestAppendClientDriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		hello  *Hello
		maxLen int
		want   bson.D
	}{
		{
			name:   "empty",
			hello:  &Hello{},
			maxLen: 0,
			want:   bson.D{},
		},
		{
			name:  "1 less than enough space",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{},
		},
		{
			name:  "1 less than enough space for name",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name:  "exact amount of space for name",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "driver", Value: bson.D{{Key: "name", Value: driverName}}}},
		},
		{
			name:  "1 more than enough space for name",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "driver", Value: bson.D{{Key: "name", Value: driverName}}}},
		},
		{
			name:  "1 less than enough space for version",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)
				dst = bsoncore.AppendStringElement(dst, "version", version.Driver)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "driver", Value: bson.D{{Key: "name", Value: driverName}}}},
		},
		{
			name:  "exact amount of space for version",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)
				dst = bsoncore.AppendStringElement(dst, "version", version.Driver)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
				{Key: "version", Value: version.Driver},
			}}},
		},
		{
			name:  "1 more than enough space for version",
			hello: &Hello{},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "driver")
				dst = bsoncore.AppendStringElement(dst, "name", driverName)
				dst = bsoncore.AppendStringElement(dst, "version", version.Driver)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "driver", Value: bson.D{
				{Key: "name", Value: driverName},
				{Key: "version", Value: version.Driver},
			}}},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cb := func(n int, dst []byte) ([]byte, error) {
				// Buffer the maxLen by the number of bytes
				// written so far.
				maxLen := test.maxLen + n

				var err error
				dst, err = appendClientDriver(dst, maxLen)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertAppendClientMaxLen(t, got, test.want, test.maxLen)
		})
	}
}

func TestAppendClientEnv(t *testing.T) {
	tests := []struct {
		name   string
		maxLen int
		want   bson.D
		env    map[string]string
	}{
		{
			name:   "empty",
			maxLen: 0,
			want:   bson.D{},
		},
		{
			name: "1 less than enough space",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name: "exact amount of space",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{},
		},
		{
			name: "1 less than enough space for aws name",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_java8",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name: "exact amount of space for aws name",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_java8",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{{Key: "name", Value: envNameAWSLambda}}}},
		},
		{
			name: "1 more than enough space for aws name",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_java8",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{{Key: "name", Value: envNameAWSLambda}}}},
		},
		{
			name: "exact amount of space for aws name but not env",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{},
		},
		{
			name: "1 less than enough space for aws name and memory_mb",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{{Key: "name", Value: envNameAWSLambda}}}},
		},
		{
			name: "exact amount of space for aws name and memory_mb",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAWSLambda},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "1 more than enough space for aws name and memory_mb",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAWSLambda},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "1 less than enough space for aws name, memory_mb, and region",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
				envVarAWSRegion:                   "us-east-1",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)
				dst = bsoncore.AppendStringElement(dst, "region", "us-east-1")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAWSLambda},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "exact amount of space for aws name, memory_mb, and region",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
				envVarAWSRegion:                   "us-east-1",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)
				dst = bsoncore.AppendStringElement(dst, "region", "us-east-1")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAWSLambda},
				{Key: "memory_mb", Value: int32(1024)},
				{Key: "region", Value: "us-east-1"},
			}}},
		},
		{
			name: "1 more than enough space for aws name, memory_mb, and region",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_java8",
				envVarAWSLambdaFunctionMemorySize: "1024",
				envVarAWSRegion:                   "us-east-1",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAWSLambda)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)
				dst = bsoncore.AppendStringElement(dst, "region", "us-east-1")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAWSLambda},
				{Key: "memory_mb", Value: int32(1024)},
				{Key: "region", Value: "us-east-1"},
			}}},
		},
		{
			name: "1 less than enouch for gcp name",
			env: map[string]string{
				envVarKService: "gcp",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name: "exact amount of space for gcp name",
			env: map[string]string{
				envVarKService: "gcp",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
			}}},
		},
		{
			name: "1 more than enough space for gcp name",
			env: map[string]string{
				envVarKService: "gcp",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
			}}},
		},
		{
			name: "1 less than enough space for gcp name and memory_mb",
			env: map[string]string{
				envVarKService:         "gcp",
				envVarFunctionMemoryMB: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
			}}},
		},
		{
			name: "exact amount of space for gcp name and memory_mb",
			env: map[string]string{
				envVarKService:         "gcp",
				envVarFunctionMemoryMB: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "1 more than enough space for gcp name and memory_mb",
			env: map[string]string{
				envVarKService:         "gcp",
				envVarFunctionMemoryMB: "1024",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) + 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "1 less than enough space for gcp name, memory_mb, and region",
			env: map[string]string{
				envVarKService:         "gcp",
				envVarFunctionMemoryMB: "1024",
				envVarFunctionRegion:   "us-east-1",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)
				dst = bsoncore.AppendStringElement(dst, "region", "us-east-1")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
				{Key: "memory_mb", Value: int32(1024)},
			}}},
		},
		{
			name: "exact amount of space for gcp name, memory_mb, and region",
			env: map[string]string{
				envVarKService:         "gcp",
				envVarFunctionMemoryMB: "1024",
				envVarFunctionRegion:   "us-east-1",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameGCPFunc)
				dst = bsoncore.AppendInt32Element(dst, "memory_mb", 1024)
				dst = bsoncore.AppendStringElement(dst, "region", "us-east-1")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameGCPFunc},
				{Key: "memory_mb", Value: int32(1024)},
				{Key: "region", Value: "us-east-1"},
			}}},
		},
		{
			name: "1 less than enough for azure name",
			env: map[string]string{
				envVarFunctionsWorkerRuntime: "node",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAzureFunc)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name: "exact amount of space for azure name",
			env: map[string]string{
				envVarFunctionsWorkerRuntime: "node",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAzureFunc)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAzureFunc},
			}}},
		},
		{
			name: "1 more than enough space for azure name",
			env: map[string]string{
				envVarFunctionsWorkerRuntime: "node",
			},
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "env")
				dst = bsoncore.AppendStringElement(dst, "name", envNameAzureFunc)
				dst = bsoncore.AppendStringElement(dst, "extra", "extra")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "env", Value: bson.D{
				{Key: "name", Value: envNameAzureFunc},
			}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			cb := func(n int, dst []byte) ([]byte, error) {
				// Buffer the maxLen by the number of bytes
				// written so far.
				maxLen := test.maxLen + n

				var err error
				dst, err = appendClientEnv(dst, maxLen)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertAppendClientMaxLen(t, got, test.want, test.maxLen)
		})
	}
}

func TestAppendClientOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		maxLen int
		want   bson.D
	}{
		{
			name:   "empty",
			maxLen: 0,
			want:   bson.D{},
		},
		{
			name: "1 less than enough space for os type",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{},
		},
		{
			name: "exact amount of space for os type",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
			}}},
		},
		{
			name: "1 more than enough space for os type",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)
				dst = bsoncore.AppendStringElement(dst, "extra", "extra")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
			}}},
		},
		{
			name: "1 less than enough space for os architecture",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)
				dst = bsoncore.AppendStringElement(dst, "architecture", runtime.GOARCH)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
			}}},
		},
		{
			name: "exact amount of space for os architecture",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)
				dst = bsoncore.AppendStringElement(dst, "architecture", runtime.GOARCH)

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst)
			}(),
			want: bson.D{{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
				{Key: "architecture", Value: runtime.GOARCH},
			}}},
		},
		{
			name: "1 more than enough space for os architecture",
			maxLen: func() int {
				idx, dst := bsoncore.AppendDocumentElementStart(nil, "os")
				dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)
				dst = bsoncore.AppendStringElement(dst, "architecture", runtime.GOARCH)
				dst = bsoncore.AppendStringElement(dst, "extra", "extra")

				var err error
				dst, err = bsoncore.AppendDocumentEnd(dst, idx)
				require.NoError(t, err, "error appending document end: %v", err)

				return len(dst) - 1
			}(),
			want: bson.D{{Key: "os", Value: bson.D{
				{Key: "type", Value: runtime.GOOS},
				{Key: "architecture", Value: runtime.GOARCH},
			}}},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cb := func(n int, dst []byte) ([]byte, error) {
				maxLen := test.maxLen + n

				var err error
				dst, err = appendClientOS(dst, maxLen)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertAppendClientMaxLen(t, got, test.want, test.maxLen)
		})
	}
}

func TestAppendClientPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		maxLen int
		want   bson.D
	}{
		{
			name:   "empty",
			maxLen: 0,
			want:   bson.D{},
		},
		{
			name:   "1 less than enough space for platform",
			maxLen: lenStringElem("platform", runtime.Version()) - 1,
			want:   bson.D{},
		},
		{
			name:   "exact amount of space for platform",
			maxLen: lenStringElem("platform", runtime.Version()),
			want:   bson.D{{Key: "platform", Value: runtime.Version()}},
		},
		{
			name:   "1 more than enough space for platform",
			maxLen: lenStringElem("platform", runtime.Version()) + 1,
			want:   bson.D{{Key: "platform", Value: runtime.Version()}},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cb := func(n int, dst []byte) ([]byte, error) {
				maxLen := test.maxLen + n

				return appendClientPlatform(dst, maxLen), nil
			}

			got := encodeWithCallback(t, cb)
			assertAppendClientMaxLen(t, got, test.want, test.maxLen)
		})
	}
}

func TestParseFaasEnvName(t *testing.T) {
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
				"AWS_EXECUTION_ENV":        "hello",
				"FUNCTIONS_WORKER_RUNTIME": "hello",
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

func BenchmarkClientMetadata(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := encodeClientMetadata("foo", maxClientMetadataSize)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func FuzzEncodeClientMetadata(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte, appname string) {
		if len(b) > maxClientMetadataSize {
			return
		}

		_, err := encodeClientMetadata(appname, maxClientMetadataSize)
		if err != nil {
			t.Fatalf("error appending client: %v", err)
		}

		_, err = appendClientAppName(b, maxClientMetadataSize, appname)
		if err != nil {
			t.Fatalf("error appending client app name: %v", err)
		}

		_, err = appendClientDriver(b, maxClientMetadataSize)
		if err != nil {
			t.Fatalf("error appending client driver: %v", err)
		}

		_, err = appendClientEnv(b, maxClientMetadataSize)
		if err != nil {
			t.Fatalf("error appending client env: %v", err)
		}

		_, err = appendClientOS(b, maxClientMetadataSize)
		if err != nil {
			t.Fatalf("error appending client os: %v", err)
		}

		appendClientPlatform(b, maxClientMetadataSize)
	})
}
