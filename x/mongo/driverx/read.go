package driverx

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

//go:generate drivergen ReadOperation read.generated.go

type ReadOperation struct {
	// Command sets the command that will be run.
	cmd bsoncore.Document `drivergen:"Command"`
	// ReadConcern sets the read concern to use when running the command.
	rc *readconcern.ReadConcern `drivergen:"ReadConcern,pointerExempt"`

	// Database sets the database to run the command against.
	database string
	// Deployment sets the Deployment to run the command against.
	d Deployment `drivergen:"Deployment"`

	selector description.ServerSelector `drivergen:"ServerSelector"`
	readPref *readpref.ReadPref         `drivergen:"ReadPreference,pointerExempt"`
	clock    *session.ClusterClock      `drivergen:"Clock,pointerExempt"`
	client   *session.Client            `drivergen:"Session,pointerExempt"`
}

func Read(cmd bsoncore.Document) ReadOperation { return ReadOperation{cmd: cmd} }

func (ro ReadOperation) Execute(ctx context.Context) (bsoncore.Document, error) {
	if ro.d == nil {
		return nil, errors.New("ReadOperation must have a Deployment set before Execute can be called.")
	}
	srvr, err := ro.d.SelectServer(ctx, createReadPrefSelector(ro.selector, ro.readPref))
	if err != nil {
		return nil, err
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: ro.d.Description().Kind}
	wm, err := ro.createWireMessage(nil, desc)
	if err != nil {
		return nil, err
	}

	res, err := roundTripDecode(ctx, conn, wm)
	if err != nil {
		return nil, err
	}

	// pull out $clusterTime and operationTime and update session and clock
	_ = updateClusterTimes(ro.client, ro.clock, res)
	_ = updateOperationTime(ro.client, res)

	return res, err
}

func (ro ReadOperation) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if len(ro.database) == 0 {
		return nil, errors.New("Database must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return ro.createQueryWireMessage(dst, desc)
	}
	return ro.createMsgWireMessage(dst, desc)
}

func (ro ReadOperation) createQueryWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	flags := slaveOK(desc, ro.readPref)
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
	// Append Command
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = append(dst, ro.cmd[4:len(ro.cmd)-1]...) // Just append the elements
	dst, err := addReadConcern(dst, ro.rc, ro.client, desc)
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

func (ro ReadOperation) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): How do we allow users to supply flags? Perhaps we don't and we add
	// functions to allow users to set them themselves.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	// Append Command
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = append(dst, ro.cmd[4:len(ro.cmd)-1]...) // Just append the elements
	dst, err := addReadConcern(dst, ro.rc, ro.client, desc)
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
