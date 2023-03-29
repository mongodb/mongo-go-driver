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

const maxHelloCommandSize = 512 //  maximum size (bytes) of a hello command
const docElementSize = 7        // 7 bytes to append a document element
const stringElementSize = 7     // 7 bytes to append a string element
const int32ElementSize = 6      // 6 bytes to append an int32 element
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
	// FaaS supprint environment variable names
	envVarAWSRegion                   = "AWS_REGION"
	envVarAwsLambdaFunctionMemorySize = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
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

func appendStringElement(dst []byte, key, value string, maxLen int32) []byte {
	if int32(len(dst)+len(key)+len(value))+stringElementSize > maxLen {
		return dst
	}

	return bsoncore.AppendStringElement(dst, key, value)
}

func appendInt32Element(dst []byte, key string, value int32, maxLen int32) []byte {
	if int32(len(dst)+len(key))+int32ElementSize > maxLen {
		return dst
	}

	return bsoncore.AppendInt32Element(dst, key, value)
}

// appendClientAppName appends the application name to dst. It builds the
// application sub-document key-by-key, checking the length of the document
// after each key. If the document exceeds the maximum length, the key is
// omitted and the document is ended and returned.
func (h *Hello) appendClientAppName(dst []byte, maxLen int32) ([]byte, error) {
	const key = "application"

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document and the "application" key.
	if int32(len(dst)+len(key))+docElementSize > maxLen {
		return dst, nil
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)
	dst = appendStringElement(dst, "name", h.appname, maxLen)

	return bsoncore.AppendDocumentEnd(dst, idx)
}

// appendClientDriver appends the driver metadata to dst. It builds the
// driver sub-document key-by-key, checking the length of the document
// after each key. If the document exceeds the maximum length, the key is
// omitted and the document is ended and returned.
func (*Hello) appendClientDriver(dst []byte, maxLen int32) ([]byte, error) {
	const key = "driver"

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document and the "driver" key.
	if int32(len(dst)+len(key))+docElementSize > maxLen {
		return dst, nil
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	dst = appendStringElement(dst, "name", driverName, maxLen)
	dst = appendStringElement(dst, "version", version.Driver, maxLen)

	return bsoncore.AppendDocumentEnd(dst, idx)
}

// appendClientEnv appends the enviroment metadata to dst. It builds the
// environment sub-document key-by-key, checking the length of the document
// after each key. If the document exceeds the maximum length, the key is
// omitted and the document is ended and returned. If there is no FaaS
// environment, the env sub-document is omitted.
func (*Hello) appendClientEnv(dst []byte, maxLen int32) ([]byte, error) {
	const key = "env"
	const nameKey = "name"

	name := getFaasEnvName()

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document, the "env" key, the "name" key, and
	// the value of the "name" key.
	bufNeeded := int32(len(dst) + len(key) + docElementSize +
		stringElementSize + len(nameKey) + len(name))

	if bufNeeded > maxLen {
		return dst, nil
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)
	dst = appendStringElement(dst, nameKey, name, maxLen)

	switch name {
	case envNameAWSLambda:
		region := os.Getenv(envVarAWSRegion)
		dst = appendStringElement(dst, "region", region, maxLen)

		memSize := os.Getenv(envVarAwsLambdaFunctionMemorySize)
		memSizeInt, _ := strconv.Atoi(memSize)

		dst = appendInt32Element(dst, "memory_mb", int32(memSizeInt), maxLen)
	}

	return bsoncore.AppendDocumentEnd(dst, idx)
}

// appendClientOS appends the OS metadata to dst. It builds the OS
// sub-document key-by-key, checking the length of the document after each key.
// If the document exceeds the maximum length, the key is omitted and the
// document is ended and returned.
func (*Hello) appendClientOS(dst []byte, maxLen int32) ([]byte, error) {
	const key = "os"

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document and the "os" key.
	if int32(len(dst)+len(key))+docElementSize > maxLen {
		return dst, nil
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	dst = appendStringElement(dst, "type", runtime.GOOS, maxLen)
	dst = appendStringElement(dst, "architecture", runtime.GOARCH, maxLen)

	return bsoncore.AppendDocumentEnd(dst, idx)
}

// appendClient appends the client metadata to dst. It builds the client
// sub-document key-by-key, checking the length of the document after each key.
// If the document exceeds the maximum length, the key is omitted and the
// document is ended and returned.
func (h *Hello) appendClient(dst []byte, maxLen int32) ([]byte, error) {
	const key = "client"

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document and the "client" key.
	if int32(len(dst)+len(key))+docElementSize > maxLen {
		return dst, nil
	}

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	var err error
	dst, err = h.appendClientAppName(dst, maxLen)
	if err != nil {
		return dst, err
	}

	dst, err = h.appendClientDriver(dst, maxLen)
	if err != nil {
		return dst, err
	}

	// Collect data required to calculate the osMaxLen.

	faasName := getFaasEnvName()
	faasNameLen := int32(len(faasName))

	// osMaxLen is the maximum number of bytes that can be used to construct
	// the os sub-document. Per the specifications, the priority of "os" and
	// "env" are interleaved such that os.type > env.name > os.* > env.*.
	// Therefore, if "faasName" is not empty, the maximum length for which
	// the os sub-document can be constructed must account for the bytes
	// required to construct an env sub-doc with "name: <faasName>".
	osMaxLen := maxLen
	if faasNameLen > 0 {
		osMaxLen -= docElementSize + stringElementSize + int32(len("name")) + faasNameLen
	}

	dst, err = h.appendClientOS(dst, osMaxLen)
	if err != nil {
		return dst, err
	}

	dst, err = h.appendClientEnv(dst, maxLen)
	if err != nil {
		return dst, err
	}

	dst = appendStringElement(dst, "platform", runtime.Version(), maxLen)

	return bsoncore.AppendDocumentEnd(dst, idx)
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

	dst, err = h.appendClient(dst, maxHelloCommandSize)
	if err != nil {
		return dst, err
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
