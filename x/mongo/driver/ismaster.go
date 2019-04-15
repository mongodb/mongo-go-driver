package driver

import (
	"context"
	"errors"
	"runtime"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/result"
)

// IsMasterOperation is used to run the isMaster handshake operation.
type IsMasterOperation struct {
	appname            string
	compressors        []string
	saslSupportedMechs string

	server Server
	conn   Connection
	tkind  description.TopologyKind

	res result.IsMaster
}

// IsMaster constructs an IsMasterOperation.
func IsMaster() *IsMasterOperation { return &IsMasterOperation{} }

// AppName sets the application name in the client metadata sent in this operation.
func (imo *IsMasterOperation) AppName(appname string) *IsMasterOperation {
	imo.appname = appname
	return imo
}

// Compressors sets the compressors that can be used.
func (imo *IsMasterOperation) Compressors(compressors []string) *IsMasterOperation {
	imo.compressors = compressors
	return imo
}

// SASLSupportedMechs retrieves the supported SASL mechanism for the given user when this operation
// is run.
func (imo *IsMasterOperation) SASLSupportedMechs(username string) *IsMasterOperation {
	imo.saslSupportedMechs = username
	return imo
}

// Server sets the server for this operation.
func (imo *IsMasterOperation) Server(server Server) *IsMasterOperation {
	imo.server = server
	return imo
}

// Connection sets the connection for this operation.
func (imo *IsMasterOperation) Connection(conn Connection) *IsMasterOperation {
	imo.conn = conn
	return imo
}

// Result returns the result of executing this operaiton.
func (imo *IsMasterOperation) Result() result.IsMaster { return imo.res }

func (imo *IsMasterOperation) processResponse(response bsoncore.Document, _ Server) error {
	// Replace this with direct unmarshaling.
	err := bson.Unmarshal(response, &imo.res)
	if err != nil {
		return err
	}

	// Reconstructs the $clusterTime doc after decode
	if imo.res.ClusterTime != nil {
		imo.res.ClusterTime = bsoncore.BuildDocument(nil, bsoncore.AppendDocumentElement(nil, "$clusterTime", imo.res.ClusterTime))
	}
	return nil
}

func (imo *IsMasterOperation) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt32Element(dst, "isMaster", 1)

	idx, dst := bsoncore.AppendDocumentElementStart(dst, "client")

	didx, dst := bsoncore.AppendDocumentElementStart(dst, "driver")
	dst = bsoncore.AppendStringElement(dst, "name", "mongo-go-driver")
	dst = bsoncore.AppendStringElement(dst, "version", version.Driver)
	dst, _ = bsoncore.AppendDocumentEnd(dst, didx)

	didx, dst = bsoncore.AppendDocumentElementStart(dst, "os")
	dst = bsoncore.AppendStringElement(dst, "type", runtime.GOOS)
	dst = bsoncore.AppendStringElement(dst, "architecture", runtime.GOARCH)
	dst, _ = bsoncore.AppendDocumentEnd(dst, didx)

	dst = bsoncore.AppendStringElement(dst, "platform", runtime.Version())
	if imo.appname != "" {
		didx, dst = bsoncore.AppendDocumentElementStart(dst, "application")
		dst = bsoncore.AppendStringElement(dst, "name", imo.appname)
		dst, _ = bsoncore.AppendDocumentEnd(dst, didx)
	}
	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	if imo.saslSupportedMechs != "" {
		dst = bsoncore.AppendStringElement(dst, "saslSupportedMechs", imo.saslSupportedMechs)
	}

	idx, dst = bsoncore.AppendArrayElementStart(dst, "compression")
	for i, compressor := range imo.compressors {
		dst = bsoncore.AppendStringElement(dst, strconv.Itoa(i), compressor)
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, idx)

	return dst, nil
}

// Execute runs this operation.
func (imo *IsMasterOperation) Execute(ctx context.Context) error {
	if imo.server == nil && imo.conn == nil {
		return errors.New("an IsMasterOperation must have a Server or Connection set before Execute can be called")
	}

	server := imo.server
	if imo.conn != nil {
		server = connectionServer{imo.conn}
	}
	return OperationContext{
		CommandFn:         imo.command,
		Server:            server,
		Database:          "admin",
		ProcessResponseFn: imo.processResponse,
	}.Execute(ctx)
}

// Handshake implements the Handshaker interface.
func (imo *IsMasterOperation) Handshake(ctx context.Context, _ address.Address, c Connection) (description.Server, error) {
	err := OperationContext{
		CommandFn:         imo.command,
		Server:            connectionServer{c},
		Database:          "admin",
		ProcessResponseFn: imo.processResponse,
	}.Execute(ctx)
	if err != nil {
		return description.Server{}, err
	}
	return description.NewServer(c.Address(), imo.res), nil
}

type connectionServer struct{ c Connection }

var _ Server = connectionServer{}

func (cs connectionServer) Connection(context.Context) (Connection, error) { return cs.c, nil }
