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

const maxClientMetadataSize = 512 //  maximum size (bytes) of a hello command
const documentSize = 5            // 5 bytes to start and end a document
const embeddedDocumentSize = 7    // 7 bytes to append a document element
const stringElementSize = 7       // 7 bytes to append a string element
const int32ElementSize = 6        // 6 bytes to append an int32 element
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

// appendStringElement appends the key and value to dst. maxLen is the maximum
// number of bytes that can be appended to dst.
func appendStringElement(dst []byte, key, value string, maxLen int32) (int32, []byte) {
	originalLen := int32(len(dst))
	if int32(len(key)+len(value))+stringElementSize > maxLen {
		return 0, dst
	}

	dst = bsoncore.AppendStringElement(dst, key, value)

	bytesWritten := int32(len(dst)) - originalLen
	return bytesWritten, dst
}

// appendInt32Element appends the key and value to dst. maxLen is the maximum
// number of bytes that can be appended to dst.
func appendInt32Element(dst []byte, key string, value int32, maxLen int32) (int32, []byte) {
	originalLen := int32(len(dst))
	if int32(len(key))+int32ElementSize > maxLen {
		return 0, dst
	}

	dst = bsoncore.AppendInt32Element(dst, key, value)

	bytesWritten := int32(len(dst)) - originalLen
	return bytesWritten, dst
}

// appendClientAppName appends the application name to dst. It builds the
// application sub-document key-by-key, checking the length of the document
// after each key. If the document exceeds the maximum length, the key is
// omitted and the document is ended and returned. MaxLen is the maximum number
// of bytes that can be appended to dst.
func appendClientAppName(dst []byte, name string, maxLen int32) (int32, []byte, error) {
	const key = "application"

	// Validate input parameters.
	if dst == nil || maxLen <= 0 {
		return 0, dst, nil
	}

	// If the size of the embedded document with the "application" key is
	// greater than the maximum length, then the document is too long and
	// we should terminate.
	if embeddedDocumentSize+int32(len(key)) > maxLen {
		return 0, dst, nil
	}

	// Account for the null byte at the end of the document.
	maxLen--

	// Save the original length of dst so we can calculate the number of
	// bytes written.
	originalLen := len(dst)
	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	// Decrease the maximum length to account for the bytes appended so far.
	maxLen -= int32(len(dst) - originalLen)

	_, dst = appendStringElement(dst, "name", name, maxLen)

	var err error

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return 0, dst, err
	}

	bytesWritten := int32(len(dst) - originalLen)
	return bytesWritten, dst, nil
}

// appendClientDriver appends the driver metadata to dst. It builds the
// driver sub-document key-by-key, checking the length of the document after
// each key. If the document exceeds the maximum length, the driver is omitted
// and the document is ended and returned. MaxLen is the maximum number of
// bytes that can be appended to dst.
func appendClientDriver(dst []byte, maxLen int32) (int32, []byte, error) {
	const key = "driver"

	// Validate input parameters.
	if dst == nil || maxLen <= 0 {
		return 0, dst, nil
	}

	if embeddedDocumentSize+int32(len(key)) > maxLen {
		return 0, dst, nil
	}

	// Account for the null byte at the end of the document.
	maxLen--

	// Save the original length of dst so we can calculate the number of
	// bytes written.
	originalLen := len(dst)

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	// Decrease the maximum length to account for the bytes appended so far.
	maxLen -= int32(len(dst) - originalLen)

	var n int32

	n, dst = appendStringElement(dst, "name", driverName, maxLen)
	maxLen -= n

	_, dst = appendStringElement(dst, "version", version.Driver, maxLen)

	var err error

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return 0, dst, err
	}

	bytesWritten := int32(len(dst) - originalLen)
	return bytesWritten, dst, nil
}

// appendClientEnv appends the enviroment metadata to dst. It builds the
// environment sub-document key-by-key, checking the length of the document
// after each key. If the document exceeds the maximum length, the key is
// omitted and the document is ended and returned. If there is no FaaS
// environment, the env sub-document is omitted.
func appendClientEnv(dst []byte, maxLen int32) (int32, []byte, error) {
	const key = "env"
	const nameKey = "name"

	name := getFaasEnvName()

	// If there is no FaaS environment, the env sub-document is omitted.
	if name == "" {
		return 0, dst, nil
	}

	if embeddedDocumentSize+int32(len(key)+len(nameKey)+len(name)+stringElementSize) > maxLen {
		return 0, dst, nil
	}

	// Account for the null byte at the end of the document.
	maxLen--

	// Save the original length of dst so we can calculate the number of
	// bytes written.
	originalLen := len(dst)

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	// Decrease the maximum length to account for the bytes appended so far.
	maxLen -= int32(len(dst) - originalLen)

	var n int32
	n, dst = appendStringElement(dst, nameKey, name, maxLen)

	maxLen -= n

	addMem := func(envVar string) (n int32) {
		mem := os.Getenv(envVar)
		if mem == "" {
			return n
		}

		memInt, err := strconv.Atoi(mem)
		if err != nil {
			return n
		}

		n, dst = appendInt32Element(dst, "memory_mb", int32(memInt), maxLen)
		return n
	}

	addRegion := func(envVar string) (n int32) {
		region := os.Getenv(envVar)
		if region == "" {
			return n
		}

		n, dst = appendStringElement(dst, "region", region, maxLen)

		return n
	}

	switch name {
	case envNameAWSLambda:
		maxLen -= addMem(envVarAWSLambdaFunctionMemorySize)
		addRegion(envVarAWSRegion)
	case envNameGCPFunc:
		maxLen -= addMem(envVarFunctionMemoryMB)
		maxLen -= addRegion(envVarFunctionRegion)

		timeout := os.Getenv(envVarFunctionTimeoutSec)
		if timeout != "" {
			timeoutInt, _ := strconv.Atoi(timeout)
			_, dst = appendInt32Element(dst, "timeout_sec", int32(timeoutInt), maxLen)
		}
	case envNameVercel:
		maxLen -= addRegion(envVarVercelRegion)

		vurl := os.Getenv(envVarVercelURL)
		if vurl != "" {
			_, dst = appendStringElement(dst, "url", vurl, maxLen)
		}
	}

	var err error

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return 0, dst, err
	}

	bytesWritten := int32(len(dst) - originalLen)
	return bytesWritten, dst, nil
}

// appendClientOS appends the OS metadata to dst. It builds the OS
// sub-document key-by-key, checking the length of the document after each key.
// If the document exceeds the maximum length, the key is omitted and the
// document is ended and returned.
func appendClientOS(dst []byte, maxLen int32) (int32, []byte, error) {
	const key = "os"

	// Do nothing if the dst slice is already too long, taking into account
	// the size of the embedded document and the "os" key.
	if int32(len(key))+embeddedDocumentSize > maxLen {
		return 0, dst, nil
	}

	// Account for the null byte at the end of the document.
	maxLen--

	// Save the original length of dst so we can calculate the number of
	// bytes written.
	originalLen := len(dst)

	idx, dst := bsoncore.AppendDocumentElementStart(dst, key)

	// Decrease the maximum length to account for the bytes appended so far.
	maxLen -= int32(len(dst) - originalLen)

	var n int32

	n, dst = appendStringElement(dst, "type", runtime.GOOS, maxLen)
	maxLen -= n

	_, dst = appendStringElement(dst, "architecture", runtime.GOARCH, maxLen)

	var err error

	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return 0, dst, err
	}

	bytesWritten := int32(len(dst) - originalLen)
	return bytesWritten, dst, nil
}

// appendClientPlatform appends the platform metadata to dst. If the document
// exceeds the maximum length, the platform is omitted.
func appendClientPlatform(dst []byte, maxLen int32) (int32, []byte, error) {
	const key = "platform"
	originalLen := len(dst)

	if int32(len(key))+stringElementSize > maxLen {
		return 0, dst, nil
	}

	_, dst = appendStringElement(dst, key, runtime.Version(), maxLen)

	bytesWritten := len(dst) - originalLen
	return int32(bytesWritten), dst, nil
}

// encodeClientMetadata encodes the client metadata into a BSON document. maxLen
// is the maximum length the document can be. If the document exceeds maxLen,
// then an empty byte slice is returned. If there is not enough space to encode
// a document, the document is truncated and returned.
func encodeClientMetadata(h *Hello, maxLen int32) ([]byte, error) {
	originalMaxLen := maxLen

	dst := make([]byte, 0, maxLen)
	if documentSize > maxLen {
		return dst[:0], nil
	}

	maxLen -= 1 // Account for the null byte at the end of the document.

	idx, dst := bsoncore.AppendDocumentStart(dst)
	maxLen -= int32(len(dst))

	var err error
	var n int32

	n, dst, err = appendClientAppName(dst, h.appname, maxLen)
	if err != nil {
		return dst, err
	}

	maxLen -= n

	n, dst, err = appendClientDriver(dst, maxLen)
	if err != nil {
		return dst, err
	}

	maxLen -= n

	n, dst, err = appendClientOS(dst, maxLen)
	if err != nil {
		return dst, err
	}

	maxLen -= n

	n, dst, err = appendClientEnv(dst, maxLen)
	if err != nil {
		return dst, err
	}

	maxLen -= n

	n, dst, err = appendClientPlatform(dst, maxLen)
	if err != nil {
		return dst, err
	}

	//dst = appendStringElement(dst, "platform", runtime.Version(), maxLen)
	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	if err != nil {
		return dst, err
	}

	// If the final "dst" slice is somehow greater than the maximum length,
	// then return an empty slice. This should never happen, but the server
	// may error if the document exceeds the maximum length.
	if int32(len(dst)) > originalMaxLen {
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

	clientMetadata, err := encodeClientMetadata(h, maxClientMetadataSize)
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
