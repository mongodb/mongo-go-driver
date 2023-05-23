// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"fmt"
	"runtime"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func assertDocsEqual(t *testing.T, got bsoncore.Document, want []byte) {
	t.Helper()

	var gotD bson.D
	err := bson.Unmarshal(got, &gotD)
	require.NoError(t, err, "error unmarshaling got document: %v", err)

	var wantD bson.D
	err = bson.UnmarshalExtJSON(want, true, &wantD)
	require.NoError(t, err, "error unmarshaling want byte slice: %v", err)

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

// clearTestEnv will clear the test environment created by tests. This will
// ensure that the local environment does not effect the outcome of a unit
// test.
func clearTestEnv(t *testing.T) {
	t.Setenv(envVarAWSExecutionEnv, "")
	t.Setenv(envVarAWSLambdaRuntimeAPI, "")
	t.Setenv(envVarFunctionsWorkerRuntime, "")
	t.Setenv(envVarKService, "")
	t.Setenv(envVarFunctionName, "")
	t.Setenv(envVarVercel, "")
	t.Setenv(envVarAWSRegion, "")
	t.Setenv(envVarAWSLambdaFunctionMemorySize, "")
	t.Setenv(envVarFunctionMemoryMB, "")
	t.Setenv(envVarFunctionTimeoutSec, "")
	t.Setenv(envVarFunctionRegion, "")
	t.Setenv(envVarVercelRegion, "")
}

func TestAppendClientName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appname string
		want    []byte // Extend JSON
	}{
		{
			name: "empty",
			want: []byte(`{}`),
		},
		{
			name:    "non-empty",
			appname: "foo",
			want:    []byte(`{"application":{"name":"foo"}}`),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cb := func(_ int, dst []byte) ([]byte, error) {
				var err error
				dst, err = appendClientAppName(dst, test.appname)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertDocsEqual(t, got, test.want)
		})
	}
}

func TestAppendClientDriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want []byte // Extend JSON
	}{
		{
			name: "full",
			want: []byte(fmt.Sprintf(`{"driver":{"name": %q, "version": %q}}`, driverName, version.Driver)),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cb := func(_ int, dst []byte) ([]byte, error) {
				var err error
				dst, err = appendClientDriver(dst)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertDocsEqual(t, got, test.want)
		})
	}
}

func TestAppendClientEnv(t *testing.T) {
	clearTestEnv(t)

	tests := []struct {
		name          string
		omitEnvFields bool
		env           map[string]string
		want          []byte // Extended JSON
	}{
		{
			name: "empty",
			want: []byte(`{}`),
		},
		{
			name:          "empty with omit",
			omitEnvFields: true,
			want:          []byte(`{}`),
		},
		{
			name: "aws only",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_foo",
			},
			want: []byte(`{"env":{"name":"aws.lambda"}}`),
		},
		{
			name: "aws mem only",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_foo",
				envVarAWSLambdaFunctionMemorySize: "1024",
			},
			want: []byte(`{"env":{"name":"aws.lambda","memory_mb":1024}}`),
		},
		{
			name: "aws region only",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_foo",
				envVarAWSRegion:       "us-east-2",
			},
			want: []byte(`{"env":{"name":"aws.lambda","region":"us-east-2"}}`),
		},
		{
			name: "aws mem and region",
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_foo",
				envVarAWSLambdaFunctionMemorySize: "1024",
				envVarAWSRegion:                   "us-east-2",
			},
			want: []byte(`{"env":{"name":"aws.lambda","memory_mb":1024,"region":"us-east-2"}}`),
		},
		{
			name:          "aws mem and region with omit fields",
			omitEnvFields: true,
			env: map[string]string{
				envVarAWSExecutionEnv:             "AWS_Lambda_foo",
				envVarAWSLambdaFunctionMemorySize: "1024",
				envVarAWSRegion:                   "us-east-2",
			},
			want: []byte(`{"env":{"name":"aws.lambda"}}`),
		},
		{
			name: "gcp only",
			env: map[string]string{
				envVarKService: "servicename",
			},
			want: []byte(`{"env":{"name":"gcp.func"}}`),
		},
		{
			name: "gcp mem",
			env: map[string]string{
				envVarKService:         "servicename",
				envVarFunctionMemoryMB: "1024",
			},
			want: []byte(`{"env":{"name":"gcp.func","memory_mb":1024}}`),
		},
		{
			name: "gcp region",
			env: map[string]string{
				envVarKService:       "servicename",
				envVarFunctionRegion: "us-east-2",
			},
			want: []byte(`{"env":{"name":"gcp.func","region":"us-east-2"}}`),
		},
		{
			name: "gcp timeout",
			env: map[string]string{
				envVarKService:           "servicename",
				envVarFunctionTimeoutSec: "1",
			},
			want: []byte(`{"env":{"name":"gcp.func","timeout_sec":1}}`),
		},
		{
			name: "gcp mem, region, and timeout",
			env: map[string]string{
				envVarKService:           "servicename",
				envVarFunctionTimeoutSec: "1",
				envVarFunctionRegion:     "us-east-2",
				envVarFunctionMemoryMB:   "1024",
			},
			want: []byte(`{"env":{"name":"gcp.func","memory_mb":1024,"region":"us-east-2","timeout_sec":1}}`),
		},
		{
			name:          "gcp mem, region, and timeout with omit fields",
			omitEnvFields: true,
			env: map[string]string{
				envVarKService:           "servicename",
				envVarFunctionTimeoutSec: "1",
				envVarFunctionRegion:     "us-east-2",
				envVarFunctionMemoryMB:   "1024",
			},
			want: []byte(`{"env":{"name":"gcp.func"}}`),
		},
		{
			name: "vercel only",
			env: map[string]string{
				envVarVercel: "1",
			},
			want: []byte(`{"env":{"name":"vercel"}}`),
		},
		{
			name: "vercel region",
			env: map[string]string{
				envVarVercel:       "1",
				envVarVercelRegion: "us-east-2",
			},
			want: []byte(`{"env":{"name":"vercel","region":"us-east-2"}}`),
		},
		{
			name: "azure only",
			env: map[string]string{
				envVarFunctionsWorkerRuntime: "go1.x",
			},
			want: []byte(`{"env":{"name":"azure.func"}}`),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			for key, val := range test.env {
				t.Setenv(key, val)
			}

			cb := func(_ int, dst []byte) ([]byte, error) {
				var err error
				dst, err = appendClientEnv(dst, test.omitEnvFields, false)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertDocsEqual(t, got, test.want)
		})
	}
}

func TestAppendClientOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		omitNonType bool
		want        []byte // Extended JSON
	}{
		{
			name: "full",
			want: []byte(fmt.Sprintf(`{"os":{"type":%q,"architecture":%q}}`, runtime.GOOS, runtime.GOARCH)),
		},
		{
			name:        "partial",
			omitNonType: true,
			want:        []byte(fmt.Sprintf(`{"os":{"type":%q}}`, runtime.GOOS)),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cb := func(_ int, dst []byte) ([]byte, error) {
				var err error
				dst, err = appendClientOS(dst, test.omitNonType)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertDocsEqual(t, got, test.want)
		})
	}
}

func TestAppendClientPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want []byte // Extended JSON
	}{
		{
			name: "full",
			want: []byte(fmt.Sprintf(`{"platform":%q}`, runtime.Version())),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cb := func(_ int, dst []byte) ([]byte, error) {
				var err error
				dst = appendClientPlatform(dst)

				return dst, err
			}

			got := encodeWithCallback(t, cb)
			assertDocsEqual(t, got, test.want)
		})
	}
}

func TestEncodeClientMetadata(t *testing.T) {
	clearTestEnv(t)

	type application struct {
		Name string `bson:"name"`
	}

	type driver struct {
		Name    string `bson:"name"`
		Version string `bson:"version"`
	}

	type dist struct {
		Type         string `bson:"type,omitempty"`
		Architecture string `bson:"architecture,omitempty"`
	}

	type env struct {
		Name       string `bson:"name,omitempty"`
		TimeoutSec int64  `bson:"timeout_sec,omitempty"`
		MemoryMB   int32  `bson:"memory_mb,omitempty"`
		Region     string `bson:"region,omitempty"`
	}

	type clientMetadata struct {
		Application *application `bson:"application"`
		Driver      *driver      `bson:"driver"`
		OS          *dist        `bson:"os"`
		Platform    string       `bson:"platform,omitempty"`
		Env         *env         `bson:"env,omitempty"`
	}

	formatJSON := func(client *clientMetadata) []byte {
		bytes, err := bson.MarshalExtJSON(client, true, false)
		require.NoError(t, err, "error encoding client metadata for test: %v", err)

		return bytes
	}

	// Set environment variables to add `env` field to handshake.
	t.Setenv(envVarAWSLambdaRuntimeAPI, "lambda")
	t.Setenv(envVarAWSLambdaFunctionMemorySize, "123")
	t.Setenv(envVarAWSRegion, "us-east-2")

	t.Run("nothing is omitted", func(t *testing.T) {
		got, err := encodeClientMetadata("foo", maxClientMetadataSize)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)

		want := formatJSON(&clientMetadata{
			Application: &application{Name: "foo"},
			Driver:      &driver{Name: driverName, Version: version.Driver},
			OS:          &dist{Type: runtime.GOOS, Architecture: runtime.GOARCH},
			Platform:    runtime.Version(),
			Env:         &env{Name: envNameAWSLambda, MemoryMB: 123, Region: "us-east-2"},
		})

		assertDocsEqual(t, got, want)
	})

	t.Run("env is omitted sub env.name", func(t *testing.T) {
		// Calculate the full length of a bsoncore.Document.
		temp, err := encodeClientMetadata("foo", maxClientMetadataSize)
		require.NoError(t, err, "error constructing template: %v", err)

		got, err := encodeClientMetadata("foo", len(temp)-1)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)

		want := formatJSON(&clientMetadata{
			Application: &application{Name: "foo"},
			Driver:      &driver{Name: driverName, Version: version.Driver},
			OS:          &dist{Type: runtime.GOOS, Architecture: runtime.GOARCH},
			Platform:    runtime.Version(),
			Env:         &env{Name: envNameAWSLambda},
		})

		assertDocsEqual(t, got, want)
	})

	t.Run("os is omitted sub os.type", func(t *testing.T) {
		// Calculate the full length of a bsoncore.Document.
		temp, err := encodeClientMetadata("foo", maxClientMetadataSize)
		require.NoError(t, err, "error constructing template: %v", err)

		// Calculate what the environment costs.
		edst, err := appendClientEnv(nil, false, false)
		require.NoError(t, err, "error constructing env template: %v", err)

		// Calculate what the env.name costs.
		ndst := bsoncore.AppendStringElement(nil, "name", envNameAWSLambda)

		// Environment sub name.
		envSubName := len(edst) - len(ndst)

		got, err := encodeClientMetadata("foo", len(temp)-envSubName-1)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)

		want := formatJSON(&clientMetadata{
			Application: &application{Name: "foo"},
			Driver:      &driver{Name: driverName, Version: version.Driver},
			OS:          &dist{Type: runtime.GOOS},
			Platform:    runtime.Version(),
			Env:         &env{Name: envNameAWSLambda},
		})

		assertDocsEqual(t, got, want)
	})

	t.Run("omit the env doc entirely", func(t *testing.T) {
		// Calculate the full length of a bsoncore.Document.
		temp, err := encodeClientMetadata("foo", maxClientMetadataSize)
		require.NoError(t, err, "error constructing template: %v", err)

		// Calculate what the environment costs.
		edst, err := appendClientEnv(nil, false, false)
		require.NoError(t, err, "error constructing env template: %v", err)

		// Calculate what the os.type costs.
		odst := bsoncore.AppendStringElement(nil, "type", runtime.GOOS)

		// Calculate what the environment plus the os.type costs.
		envAndOSType := len(edst) + len(odst)

		got, err := encodeClientMetadata("foo", len(temp)-envAndOSType-1)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)

		want := formatJSON(&clientMetadata{
			Application: &application{Name: "foo"},
			Driver:      &driver{Name: driverName, Version: version.Driver},
			OS:          &dist{Type: runtime.GOOS},
			Platform:    runtime.Version(),
		})

		assertDocsEqual(t, got, want)
	})

	t.Run("omit the platform", func(t *testing.T) {
		// Calculate the full length of a bsoncore.Document.
		temp, err := encodeClientMetadata("foo", maxClientMetadataSize)
		require.NoError(t, err, "error constructing template: %v", err)

		// Calculate what the environment costs.
		edst, err := appendClientEnv(nil, false, false)
		require.NoError(t, err, "error constructing env template: %v", err)

		// Calculate what the os.type costs.
		odst := bsoncore.AppendStringElement(nil, "type", runtime.GOOS)

		// Calculate what the platform costs
		pdst := appendClientPlatform(nil)

		// Calculate what the environment plus the os.type costs.
		envAndOSTypeAndPlatform := len(edst) + len(odst) + len(pdst)

		got, err := encodeClientMetadata("foo", len(temp)-envAndOSTypeAndPlatform)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)

		want := formatJSON(&clientMetadata{
			Application: &application{Name: "foo"},
			Driver:      &driver{Name: driverName, Version: version.Driver},
			OS:          &dist{Type: runtime.GOOS},
		})

		assertDocsEqual(t, got, want)
	})

	t.Run("0 max len", func(t *testing.T) {
		got, err := encodeClientMetadata("foo", 0)
		assert.Nil(t, err, "error in encodeClientMetadata: %v", err)
		assert.Len(t, got, 0)
	})
}

func TestParseFaasEnvName(t *testing.T) {
	clearTestEnv(t)

	tests := []struct {
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
				envVarAWSExecutionEnv: "AWS_Lambda_foo",
			},
			want: envNameAWSLambda,
		},
		{
			name: "both aws options",
			env: map[string]string{
				envVarAWSExecutionEnv:     "AWS_Lambda_foo",
				envVarAWSLambdaRuntimeAPI: "hello",
			},
			want: envNameAWSLambda,
		},
		{
			name: "multiple variables",
			env: map[string]string{
				envVarAWSExecutionEnv:        "AWS_Lambda_foo",
				envVarFunctionsWorkerRuntime: "hello",
			},
			want: "",
		},
		{
			name: "vercel and aws lambda",
			env: map[string]string{
				envVarAWSExecutionEnv: "AWS_Lambda_foo",
				envVarVercel:          "hello",
			},
			want: envNameVercel,
		},
		{
			name: "invalid aws prefix",
			env: map[string]string{
				envVarAWSExecutionEnv: "foo",
			},
			want: "",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			got := getFaasEnvName()
			if got != test.want {
				t.Errorf("parseFaasEnvName(%s) = %s, want %s", test.name, got, test.want)
			}
		})
	}
}

func BenchmarkClientMetadata(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := encodeClientMetadata("foo", maxClientMetadataSize)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkClientMetadtaLargeEnv(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.Setenv(envNameAWSLambda, "foo")

	str := ""
	for i := 0; i < 512; i++ {
		str += "a"
	}

	b.Setenv(envVarAWSLambdaRuntimeAPI, str)

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

		_, err = appendClientAppName(b, appname)
		if err != nil {
			t.Fatalf("error appending client app name: %v", err)
		}

		_, err = appendClientDriver(b)
		if err != nil {
			t.Fatalf("error appending client driver: %v", err)
		}

		_, err = appendClientEnv(b, false, false)
		if err != nil {
			t.Fatalf("error appending client env ff: %v", err)
		}

		_, err = appendClientEnv(b, false, true)
		if err != nil {
			t.Fatalf("error appending client env ft: %v", err)
		}

		_, err = appendClientEnv(b, true, false)
		if err != nil {
			t.Fatalf("error appending client env tf: %v", err)
		}

		_, err = appendClientEnv(b, true, true)
		if err != nil {
			t.Fatalf("error appending client env tt: %v", err)
		}

		_, err = appendClientOS(b, false)
		if err != nil {
			t.Fatalf("error appending client os f: %v", err)
		}

		_, err = appendClientOS(b, true)
		if err != nil {
			t.Fatalf("error appending client os t: %v", err)
		}

		appendClientPlatform(b)
	})
}
