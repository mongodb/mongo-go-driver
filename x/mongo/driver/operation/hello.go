// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// maxClientMetadataSize is the maximum size of the client metadata document
// that can be sent to the server. Note that the maximum document size on
// standalone and replica servers is 1024, but the maximum document size on
// sharded clusters is 512.
const maxClientMetadataSize = 512

const driverName = "mongo-go-driver"

// Hello is used to run the handshake operation.
type Hello struct {
	appname            string
	compressors        []string
	saslSupportedMechs string
	d                  driver.Deployment
	clock              *session.ClusterClock
	speculativeAuth    bsoncore.Document
	topologyVersion    *description.TopologyVersion
	maxAwaitTimeMS     *int64
	serverAPI          *driver.ServerAPIOptions
	loadBalanced       bool

	res bsoncore.Document
}

var _ driver.Handshaker = (*Hello)(nil)

// NewHello constructs a Hello.
func NewHello() *Hello { return &Hello{} }

// AppName sets the application name in the client metadata sent in this operation.
func (h *Hello) AppName(appname string) *Hello {
	h.appname = appname
	return h
}

// ClusterClock sets the cluster clock for this operation.
func (h *Hello) ClusterClock(clock *session.ClusterClock) *Hello {
	if h == nil {
		h = new(Hello)
	}

	h.clock = clock
	return h
}

// Compressors sets the compressors that can be used.
func (h *Hello) Compressors(compressors []string) *Hello {
	h.compressors = compressors
	return h
}

// SASLSupportedMechs retrieves the supported SASL mechanism for the given user when this operation
// is run.
func (h *Hello) SASLSupportedMechs(username string) *Hello {
	h.saslSupportedMechs = username
	return h
}

// Deployment sets the Deployment for this operation.
func (h *Hello) Deployment(d driver.Deployment) *Hello {
	h.d = d
	return h
}

// SpeculativeAuthenticate sets the document to be used for speculative authentication.
func (h *Hello) SpeculativeAuthenticate(doc bsoncore.Document) *Hello {
	h.speculativeAuth = doc
	return h
}

// TopologyVersion sets the TopologyVersion to be used for heartbeats.
func (h *Hello) TopologyVersion(tv *description.TopologyVersion) *Hello {
	h.topologyVersion = tv
	return h
}

// MaxAwaitTimeMS sets the maximum time for the server to wait for topology changes during a heartbeat.
func (h *Hello) MaxAwaitTimeMS(awaitTime int64) *Hello {
	h.maxAwaitTimeMS = &awaitTime
	return h
}

// ServerAPI sets the server API version for this operation.
func (h *Hello) ServerAPI(serverAPI *driver.ServerAPIOptions) *Hello {
	h.serverAPI = serverAPI
	return h
}

// LoadBalanced specifies whether or not this operation is being sent over a connection to a load balanced cluster.
func (h *Hello) LoadBalanced(lb bool) *Hello {
	h.loadBalanced = lb
	return h
}

// Result returns the result of executing this operation.
func (h *Hello) Result(addr address.Address) description.Server {
	return description.NewServer(addr, bson.Raw(h.res))
}

const (
	// FaaS environment variable names
	envVarAWSExecutionEnv        = "AWS_EXECUTION_ENV"
	envVarAWSLambdaRuntimeAPI    = "AWS_LAMBDA_RUNTIME_API"
	envVarFunctionsWorkerRuntime = "FUNCTIONS_WORKER_RUNTIME"
	envVarKService               = "K_SERVICE"
	envVarFunctionName           = "FUNCTION_NAME"
	envVarVercel                 = "VERCEL"
)

const (
	// FaaS environment variable names
	envVarAWSRegion                   = "AWS_REGION"
	envVarAWSLambdaFunctionMemorySize = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
	envVarFunctionMemoryMB            = "FUNCTION_MEMORY_MB"
	envVarFunctionTimeoutSec          = "FUNCTION_TIMEOUT_SEC"
	envVarFunctionRegion              = "FUNCTION_REGION"
	envVarVercelURL                   = "VERCEL_URL"
	envVarVercelRegion                = "VERCEL_REGION"
)

const (
	// FaaS environment names used by the client
	envNameAWSLambda = "aws.lambda"
	envNameAzureFunc = "azure.func"
	envNameGCPFunc   = "gcp.func"
	envNameVercel    = "vercel"
)

// getFaasEnvName parses the FaaS environment variable name and returns the
// corresponding name used by the client. If none of the variables or variables
// for multiple names are populated the client.env value MUST be entirely
// omitted.
func getFaasEnvName() string {
	envVars := []string{
		envVarAWSExecutionEnv,
		envVarAWSLambdaRuntimeAPI,
		envVarFunctionsWorkerRuntime,
		envVarKService,
		envVarFunctionName,
		envVarVercel,
	}

	// If none of the variables are populated the client.env value MUST be
	// entirely omitted.
	names := make(map[string]struct{})

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			continue
		}

		var name string

		switch envVar {
		case envVarAWSExecutionEnv, envVarAWSLambdaRuntimeAPI:
			name = envNameAWSLambda
		case envVarFunctionsWorkerRuntime:
			name = envNameAzureFunc
		case envVarKService, envVarFunctionName:
			name = envNameGCPFunc
		case envVarVercel:
			name = envNameVercel
		}

		names[name] = struct{}{}
		if len(names) > 1 {
			// If multiple names are populated the client.env value
			// MUST be entirely omitted.
			names = nil

			break
		}
	}

	for name := range names {
		return name
	}

	return ""
}

// appendWithMaxLen appends the "dst" slice using the provided function. If the
// length of the resulting slice is greater than maxLen, the original slice is
// returned. Otherwise, the new slice is returned.
func appendWithMaxLen(dst []byte, maxLen int, fn func([]byte) []byte) []byte {
	origLen := len(dst)

	dst = fn(dst)
	if len(dst) > maxLen {
		return dst[:origLen]
	}

	return dst
}

func appendSubDocument(dst []byte, maxLen int, key string, fns []func([]byte) []byte) ([]byte, error) {
	if dst == nil || maxLen <= 0 || len(fns) == 0 {
		return dst, nil
	}

	bytesLeft := maxLen - 1 // -1 for the null byte to end the sub doc.
	originalLen := len(dst)

	var idx int32
	dst = appendWithMaxLen(dst, bytesLeft, func(dst []byte) []byte {
		idx, dst = bsoncore.AppendDocumentElementStart(dst, key)

		return dst
	})

	// If the sub-document is too large then continuing will corrupt the
	// document so return the original slice.
	if len(dst) == originalLen {
		return dst, nil
	}

	emptySize := len(dst) // Size of the sub-document if it is empty.

	for _, fn := range fns {
		if fn == nil {
			continue
		}

		dst = appendWithMaxLen(dst, bytesLeft, fn)
	}

	// If the driver sub doc is empty, truncate and return the original
	// slice.
	if len(dst) == emptySize {
		return dst[:originalLen], nil
	}

	var err error

	// End the driver sub-document.
	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return dst, err
	}

	return dst, nil
}

// appendClientAppName appends the application name to dst. If appending the
// application name would cause the length of the document to exceed maxLen,
// the original dst slice is returned.
func appendClientAppName(dst []byte, maxLen int, name string) ([]byte, error) {
	if dst == nil || maxLen <= 0 {
		return dst, nil
	}

	const appNameKey = "application"
	var err error

	dst, err = appendSubDocument(dst, maxLen, appNameKey, []func([]byte) []byte{
		func(dst []byte) []byte {
			const nameKey = "name"

			return bsoncore.AppendStringElement(dst, nameKey, name)
		},
	})

	return dst, err
}

// appendClientDriver appends the driver metadata to dst. If appending the
// driver metadata would cause the length of the document to exceed maxLen,
// the original dst slice is returned. If only an empty driver sub-document
// could be appended, the original dst slice is returned.
func appendClientDriver(dst []byte, maxLen int) ([]byte, error) {
	if dst == nil || maxLen <= 0 {
		return dst, nil
	}

	const appNameKey = "driver"
	var err error

	dst, err = appendSubDocument(dst, maxLen, appNameKey, []func([]byte) []byte{
		func(dst []byte) []byte {
			const key = "name"

			return bsoncore.AppendStringElement(dst, key, driverName)
		},
		func(dst []byte) []byte {
			const key = "version"

			return bsoncore.AppendStringElement(dst, key, version.Driver)
		},
	})

	return dst, err
}

// appendClientEnv appends the environment metadata to dst. If appending the
// environment metadata would cause the length of the document to exceed
// maxLen, the original dst slice is returned. If only an empty environment
// sub-document could be appended, the original dst slice is returned. The
// "env" sub-document must contain a non-empty "name" field, so if the
// environment variable responsible for setting the name is not set, the
// original dst slice is returned.
func appendClientEnv(dst []byte, maxLen int) ([]byte, error) {
	if dst == nil || maxLen <= 0 {
		return dst, nil
	}

	name := getFaasEnvName()
	if name == "" {
		return dst, nil
	}

	addMem := func(envVar string) func([]byte) []byte {
		mem := os.Getenv(envVar)
		if mem == "" {
			return nil
		}

		memInt64, err := strconv.ParseInt(mem, 10, 32)
		if err != nil {
			return nil
		}

		memInt32 := int32(memInt64)

		return func(dst []byte) []byte {
			const key = "memory_mb"

			return bsoncore.AppendInt32Element(dst, key, memInt32)
		}
	}

	addRegion := func(envVar string) func([]byte) []byte {
		region := os.Getenv(envVar)
		if region == "" {
			return nil
		}

		return func(dst []byte) []byte {
			key := "region"

			return bsoncore.AppendStringElement(dst, key, region)
		}
	}

	addTimeout := func(envVar string) func([]byte) []byte {
		timeout := os.Getenv(envVar)
		if timeout == "" {
			return nil
		}

		timeoutInt64, err := strconv.ParseInt(timeout, 10, 32)
		if err != nil {
			return nil
		}

		timeoutInt32 := int32(timeoutInt64)
		return func(dst []byte) []byte {
			key := "timeout_sec"

			return bsoncore.AppendInt32Element(dst, key, timeoutInt32)
		}
	}

	addURL := func(envVar string) func([]byte) []byte {
		url := os.Getenv(envVar)
		if url == "" {
			return nil
		}

		return func(dst []byte) []byte {
			return bsoncore.AppendStringElement(dst, "url", url)
		}
	}

	fns := []func([]byte) []byte{
		func(dst []byte) []byte {
			return bsoncore.AppendStringElement(dst, "name", name)
		},
	}

	switch name {
	case envNameAWSLambda:
		fns = append(fns, addMem(envVarAWSLambdaFunctionMemorySize))
		fns = append(fns, addRegion(envVarAWSRegion))
	case envNameGCPFunc:
		fns = append(fns, addMem(envVarFunctionMemoryMB))
		fns = append(fns, addRegion(envVarFunctionRegion))
		fns = append(fns, addTimeout(envVarFunctionTimeoutSec))
	case envNameVercel:
		fns = append(fns, addRegion(envVarVercelRegion))
		fns = append(fns, addURL(envVarVercelURL))
	}

	const envKey = "env"
	return appendSubDocument(dst, maxLen, envKey, fns)
}

// appendClientOS appends the OS metadata to dst. If the document exceeds the
// maximum length, the OS is omitted.
func appendClientOS(dst []byte, maxLen int) ([]byte, error) {
	if dst == nil || maxLen <= 0 {
		return dst, nil
	}

	const osKey = "os"
	var err error

	dst, err = appendSubDocument(dst, maxLen, osKey, []func([]byte) []byte{
		func(dst []byte) []byte {
			const key = "type"

			return bsoncore.AppendStringElement(dst, key, runtime.GOOS)
		},
		func(dst []byte) []byte {
			const key = "architecture"

			return bsoncore.AppendStringElement(dst, key, runtime.GOARCH)
		},
	})

	return dst, err
}

// appendClientPlatform appends the platform metadata to dst. If the document
// exceeds the maximum length, the platform is omitted.
func appendClientPlatform(dst []byte, maxLen int) []byte {
	if dst == nil || maxLen <= 0 {
		return dst
	}

	const platformKey = "platform"
	return appendWithMaxLen(dst, maxLen, func(dst []byte) []byte {
		return bsoncore.AppendStringElement(dst, platformKey, runtime.Version())
	})
}

// encodeClientMetadata encodes the client metadata into a BSON document. maxLen
// is the maximum length the document can be. If the document exceeds maxLen,
// then an empty byte slice is returned. If there is not enough space to encode
// a document, the document is truncated and returned.
//
// This function attempts to build the following document, prioritizing upto the
// givien order:
//
//	{
//		application: {
//			name: "<string>"
//		},
//		driver: {
//		      	name: "<string>",
//		        version: "<string>"
//		},
//		platform: "<string>",
//		os: {
//		        type: "<string>",
//		        name: "<string>",
//		        architecture: "<string>",
//		        version: "<string>"
//		},
//		env: {
//		        name: "<string>",
//		        timeout_sec: 42,
//		        memory_mb: 1024,
//		        region: "<string>",
//		        url: "<string>"
//		}
//	}
func encodeClientMetadata(appname string, maxLen int) ([]byte, error) {
	dst := make([]byte, 0, maxLen)

	idx, dst := bsoncore.AppendDocumentStart(dst)
	bytesLeft := maxLen - 1 // -1 for the null byte at the end of the document

	var err error
	dst, err = appendClientAppName(dst, bytesLeft, appname)
	if err != nil {
		return dst, err
	}

	dst, err = appendClientDriver(dst, bytesLeft)
	if err != nil {
		return dst, err
	}

	dst, err = appendClientOS(dst, bytesLeft)
	if err != nil {
		return dst, err
	}

	dst = appendClientPlatform(dst, bytesLeft)

	dst, err = appendClientEnv(dst, bytesLeft)
	if err != nil {
		return dst, err
	}

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return dst, err
	}

	// If the final "dst" slice is somehow greater than the maximum length,
	// then return an empty slice. This should never happen, but the server
	// may error if the document exceeds the maximum length.
	if len(dst) > maxLen {
		return dst[:0], nil
	}

	return dst, nil
}

// handshakeCommand appends all necessary command fields as well as client metadata, SASL supported mechs, and compression.
func (h *Hello) handshakeCommand(dst []byte, desc description.SelectedServer) ([]byte, error) {
	dst, err := h.command(dst, desc)
	if err != nil {
		return dst, err
	}

	if h.saslSupportedMechs != "" {
		dst = bsoncore.AppendStringElement(dst, "saslSupportedMechs", h.saslSupportedMechs)
	}
	if h.speculativeAuth != nil {
		dst = bsoncore.AppendDocumentElement(dst, "speculativeAuthenticate", h.speculativeAuth)
	}
	var idx int32
	idx, dst = bsoncore.AppendArrayElementStart(dst, "compression")
	for i, compressor := range h.compressors {
		dst = bsoncore.AppendStringElement(dst, strconv.Itoa(i), compressor)
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, idx)

	clientMetadata, err := encodeClientMetadata(h.appname, maxClientMetadataSize)
	if err != nil {
		return dst, err
	}

	// If the client metadata is empty, do not append it to the command.
	if len(clientMetadata) > 0 {
		dst = bsoncore.AppendDocumentElement(dst, "client", clientMetadata)
	}

	return dst, nil
}

// command appends all necessary command fields.
func (h *Hello) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// Use "hello" if topology is LoadBalanced, API version is declared or server
	// has responded with "helloOk". Otherwise, use legacy hello.
	if desc.Kind == description.LoadBalanced || h.serverAPI != nil || desc.Server.HelloOK {
		dst = bsoncore.AppendInt32Element(dst, "hello", 1)
	} else {
		dst = bsoncore.AppendInt32Element(dst, internal.LegacyHello, 1)
	}
	dst = bsoncore.AppendBooleanElement(dst, "helloOk", true)

	if tv := h.topologyVersion; tv != nil {
		var tvIdx int32

		tvIdx, dst = bsoncore.AppendDocumentElementStart(dst, "topologyVersion")
		dst = bsoncore.AppendObjectIDElement(dst, "processId", tv.ProcessID)
		dst = bsoncore.AppendInt64Element(dst, "counter", tv.Counter)
		dst, _ = bsoncore.AppendDocumentEnd(dst, tvIdx)
	}
	if h.maxAwaitTimeMS != nil {
		dst = bsoncore.AppendInt64Element(dst, "maxAwaitTimeMS", *h.maxAwaitTimeMS)
	}
	if h.loadBalanced {
		// The loadBalanced parameter should only be added if it's true. We should never explicitly send
		// loadBalanced=false per the load balancing spec.
		dst = bsoncore.AppendBooleanElement(dst, "loadBalanced", true)
	}

	return dst, nil
}

// Execute runs this operation.
func (h *Hello) Execute(ctx context.Context) error {
	if h.d == nil {
		return errors.New("a Hello must have a Deployment set before Execute can be called")
	}

	return h.createOperation().Execute(ctx)
}

// StreamResponse gets the next streaming Hello response from the server.
func (h *Hello) StreamResponse(ctx context.Context, conn driver.StreamerConnection) error {
	return h.createOperation().ExecuteExhaust(ctx, conn)
}

func (h *Hello) createOperation() driver.Operation {
	return driver.Operation{
		Clock:      h.clock,
		CommandFn:  h.command,
		Database:   "admin",
		Deployment: h.d,
		ProcessResponseFn: func(info driver.ResponseInfo) error {
			h.res = info.ServerResponse
			return nil
		},
		ServerAPI: h.serverAPI,
	}
}

// GetHandshakeInformation performs the MongoDB handshake for the provided connection and returns the relevant
// information about the server. This function implements the driver.Handshaker interface.
func (h *Hello) GetHandshakeInformation(ctx context.Context, _ address.Address, c driver.Connection) (driver.HandshakeInformation, error) {
	err := driver.Operation{
		Clock:      h.clock,
		CommandFn:  h.handshakeCommand,
		Deployment: driver.SingleConnectionDeployment{C: c},
		Database:   "admin",
		ProcessResponseFn: func(info driver.ResponseInfo) error {
			h.res = info.ServerResponse
			return nil
		},
		ServerAPI: h.serverAPI,
	}.Execute(ctx)
	if err != nil {
		return driver.HandshakeInformation{}, err
	}

	info := driver.HandshakeInformation{
		Description: h.Result(c.Address()),
	}
	if speculativeAuthenticate, ok := h.res.Lookup("speculativeAuthenticate").DocumentOK(); ok {
		info.SpeculativeAuthenticate = speculativeAuthenticate
	}
	if serverConnectionID, ok := h.res.Lookup("connectionId").Int32OK(); ok {
		info.ServerConnectionID = &serverConnectionID
	}
	// Cast to bson.Raw to lookup saslSupportedMechs to avoid converting from bsoncore.Value to bson.RawValue for the
	// StringSliceFromRawValue call.
	if saslSupportedMechs, lookupErr := bson.Raw(h.res).LookupErr("saslSupportedMechs"); lookupErr == nil {
		info.SaslSupportedMechs, err = internal.StringSliceFromRawValue("saslSupportedMechs", saslSupportedMechs)
	}
	return info, err
}

// FinishHandshake implements the Handshaker interface. This is a no-op function because a non-authenticated connection
// does not do anything besides the initial Hello for a handshake.
func (h *Hello) FinishHandshake(context.Context, driver.Connection) error {
	return nil
}
