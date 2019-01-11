package driverx

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

// readOperation is the interface for the operaiton specific components of operation execution.
//
// TODO(GODRIVER-617): writeOperation and readOperation are the same except that writeOperation has
// a retryable method, but that will be necessary when retryable reads is implemented. The command
// method behaves slightly different because it does not include the entire command, only the
// non-Type 1 payload components.
type readOperation interface {
	// processResponse handles processing the response document from a write command. It should
	// store any internal state that will be returned later as the result of this operation.
	processResponse(response bsoncore.Document) error

	// selectSever chooses and retruns a server. This is used for both initial server selection and
	// subsequent server selection that occurs during retries.
	selectServer(context.Context) (Server, error)

	// command appends the wiremessage agnostic components of the command to dst. This excluse the
	// $db and $readPef command parameters. This method should not append the length for the BSON
	// document, nor the null byte at the end.
	command(dst []byte, desc description.SelectedServer) ([]byte, error)
}

type readOperationContext struct {
	readOperation

	tkind    description.TopologyKind
	database string

	readPref    *readpref.ReadPref
	readConcern *readconcern.ReadConcern

	client *session.Client
	clock  *session.ClusterClock
}

func (ro readOperationContext) execute(ctx context.Context) error {
	srvr, err := ro.selectServer(ctx)
	if err != nil {
		return err
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: ro.tkind}
	wm, err := ro.createWireMessage(nil, desc)
	if err != nil {
		return err
	}

	res, err := ro.roundTripDecode(ctx, conn, wm)
	if err != nil {
		return err
	}

	// Pull out $clusterTime and operationTime and update session and clock. We handle this before
	// processing the response to ensure we are properly gossiping the cluster time.
	_ = updateClusterTimes(ro.client, ro.clock, res)
	_ = updateOperationTime(ro.client, res)

	return ro.processResponse(res)
}

// roundTripDecode behaves the same as roundTrip, but also decodes the wiremessage response into
// either a result document or an error.
func (readOperationContext) roundTripDecode(ctx context.Context, conn Connection, wm []byte) (bsoncore.Document, error) {
	err := conn.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}

	res, err := conn.ReadWireMessage(ctx, wm[:0])
	if err != nil {
		err = Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}
	return decodeResult(res)
}

func (ro readOperationContext) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return ro.createQueryWireMessage(dst, desc)
	}
	return ro.createMsgWireMessage(dst, desc)
}

func (ro readOperationContext) createQueryWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	flags := slaveOK(desc, nil)
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
	dst = wiremessagex.AppendQueryFlags(dst, flags)
	// FullCollectionName
	dst = append(dst, ro.database...)
	dst = append(dst, dollarCmd[:]...)
	dst = append(dst, 0x00)
	dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
	dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)

	wrapper := int32(-1)
	rp := createReadPref(ro.readPref, desc.Server.Kind, desc.Kind, true)
	if len(rp) > 0 {
		wrapper, dst = bsoncore.AppendDocumentStart(dst)
		dst = bsoncore.AppendHeader(dst, bsontype.EmbeddedDocument, "$query")
	}
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst, err := ro.command(dst, desc)
	if err != nil {
		return dst, err
	}
	dst, err = addReadConcern(dst, ro.readConcern, ro.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, ro.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, ro.client, ro.clock, desc)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	if len(rp) > 0 {
		var err error
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
		dst, err = bsoncore.AppendDocumentEnd(dst, wrapper)
		if err != nil {
			return dst, err
		}
	}

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

func (ro readOperationContext) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): We need to figure out how to include the writeconcern here so that we can
	// set the moreToCome bit.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst, err := ro.command(dst, desc)
	if err != nil {
		return dst, err
	}
	dst, err = addReadConcern(dst, ro.readConcern, ro.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, ro.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, ro.client, ro.clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", ro.database)
	rp := createReadPref(ro.readPref, desc.Server.Kind, desc.Kind, false)
	if len(rp) > 0 {
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}
