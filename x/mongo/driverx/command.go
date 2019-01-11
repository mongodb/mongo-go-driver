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

//go:generate drivergen -use-pointers CommandOperation command.generated.go

type CommandOperation struct {
	_ struct{} `drivergen:"-"`
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

	result bsoncore.Document `drivergen:"-"`
}

func Command(cmd bsoncore.Document) *CommandOperation { return &CommandOperation{cmd: cmd} }

func (co *CommandOperation) Result() bsoncore.Document { return co.result }

// Select retrieves a server to be used when executing an operatcon.
func (co *CommandOperation) Select(ctx context.Context) (Server, error) {
	if co == nil || co.d == nil {
		return nil, errors.New("CommandOperation must be non-nil and have a Deployment set before Select can be called.")
	}
	return co.d.SelectServer(ctx, createReadPrefSelector(readpref.Primary(), co.selector))
}

// SelectAndExecute selects a server and runs this operation against it.
func (co *CommandOperation) SelectAndExecute(ctx context.Context) error {
	srvr, err := co.Select(ctx)
	if err != nil {
		return err
	}

	return co.Execute(ctx, srvr)
}

func (co *CommandOperation) Execute(ctx context.Context, srvr Server) error {
	if co == nil || co.d == nil {
		return errors.New("CommandOperation must be non-nil and have a Deployment set before Execute can be called.")
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: co.d.Description().Kind}
	wm, err := co.createWireMessage(nil, desc)
	if err != nil {
		return err
	}

	res, err := roundTripDecode(ctx, conn, wm)
	if err != nil {
		return err
	}

	// pull out $clusterTime and operationTime and update session and clock
	_ = updateClusterTimes(co.client, co.clock, res)
	_ = updateOperationTime(co.client, res)

	co.result = res
	return err
}

func (co *CommandOperation) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if len(co.database) == 0 {
		return nil, errors.New("Database must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return co.createQueryWireMessage(dst, desc)
	}
	return co.createMsgWireMessage(dst, desc)
}

func (co *CommandOperation) createQueryWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	flags := slaveOK(desc, co.readPref)
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
	dst = wiremessagex.AppendQueryFlags(dst, flags)
	// FullCollectionName
	dst = append(dst, co.database...)
	dst = append(dst, dollarCmd[:]...)
	dst = append(dst, 0x00)
	dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
	dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)
	wrapper := int32(-1)
	rp := createReadPref(co.readPref, desc.Server.Kind, desc.Kind, true)
	if len(rp) > 0 {
		wrapper, dst = bsoncore.AppendDocumentStart(dst)
		dst = bsoncore.AppendHeader(dst, bsontype.EmbeddedDocument, "$query")
	}
	// Append Command
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = append(dst, co.cmd[4:len(co.cmd)-1]...) // Just append the elements
	dst, err := addReadConcern(dst, co.rc, co.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, co.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, co.client, co.clock, desc)

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

func (co *CommandOperation) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
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
	dst = append(dst, co.cmd[4:len(co.cmd)-1]...) // Just append the elements
	dst, err := addReadConcern(dst, co.rc, co.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, co.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, co.client, co.clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", co.database)
	rp := createReadPref(co.readPref, desc.Server.Kind, desc.Kind, false)
	if len(rp) > 0 {
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}
