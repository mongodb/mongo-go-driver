package commandx

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

type Insert struct {
	// Command Parameters
	Collection               bsoncore.Value
	Documents                bsoncore.Value
	Ordered                  bsoncore.Value
	WriteConcern             bsoncore.Value
	BypassDocumentValidation bsoncore.Value

	Database         []byte // a slice of bytes instead of a string to save on allocs.
	LSID             bsoncore.Document
	TxnNumber        bsoncore.Value
	StartTransaction bsoncore.Value
	AutoCommit       bsoncore.Value
	ClusterTime      bsoncore.Document
}

func (i Insert) MarshalWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if err := i.Collection.Validate(); err != nil {
		return dst, err
	}
	if len(i.Database) == 0 {
		return nil, errors.New("Database must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		// OP_QUERY
		flags := i.slaveOK(desc)
		var wmindex int32
		wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
		dst = wiremessagex.AppendQueryFlags(dst, flags)
		// FullCollectionName
		dst = append(dst, i.Database...)
		dst = append(dst, dollarCmd[:]...)
		dst = append(dst, 0x00)
		dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
		dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)
		dst = i.appendCommand(dst, false)
		dst = bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:])))
	} else {
		// OP_MSG
		// TODO(GODRIVER-617): How do we allow users to supply flags? Perhaps we don't and we add
		// functions to allow users to set them themselves.
		var flags wiremessage.MsgFlag
		var wmindex int32
		wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
		dst = wiremessagex.AppendMsgFlags(dst, flags)
		// Body
		dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)
		dst = i.appendCommand(dst, true)
		dst = bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:])))
	}
	return dst, nil
}

func (i Insert) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	return 0
}

func (i Insert) appendCommand(dst []byte, opmsg bool) []byte {
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = bsoncore.AppendValueElement(dst, "insert", i.Collection)

	if opmsg {
		// TODO(GODRIVER-617): Do this without a string conversion.
		dst = bsoncore.AppendStringElement(dst, "$db", string(i.Database))
	}

	if i.Documents.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "documents", i.Documents)
	}
	if i.Ordered.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "ordered", i.Ordered)
	}
	if i.WriteConcern.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "writeConcern", i.WriteConcern)
	}
	if i.BypassDocumentValidation.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "bypassDocumentValidation", i.BypassDocumentValidation)
	}
	if i.LSID != nil {
		dst = bsoncore.AppendDocumentElement(dst, "lsid", i.LSID)
	}
	if i.TxnNumber.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "txnNumber", i.TxnNumber)
	}
	if i.StartTransaction.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "startTransaction", i.StartTransaction)
	}
	if i.AutoCommit.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "autocommit", i.AutoCommit)
	}
	if i.ClusterTime != nil {
		dst = bsoncore.AppendDocumentElement(dst, "$clusterTime", i.ClusterTime)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	return dst
}
