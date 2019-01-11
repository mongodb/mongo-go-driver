package commandx

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

type Find struct {
	// Command Parameters
	Collection          bsoncore.Value
	Filter              bsoncore.Document
	Sort                bsoncore.Document
	Projection          bsoncore.Document
	Hint                bsoncore.Value
	Skip                bsoncore.Value
	Limit               bsoncore.Value
	BatchSize           bsoncore.Value
	SingleBatch         bsoncore.Value
	Comment             bsoncore.Value
	MaxTimeMS           bsoncore.Value
	ReadConcern         bsoncore.Document
	Max                 bsoncore.Document
	Min                 bsoncore.Document
	ReturnKey           bsoncore.Value
	ShowRecordID        bsoncore.Value
	Tailable            bsoncore.Value
	OplogReplay         bsoncore.Value
	NoCursorTimeout     bsoncore.Value
	AwaitData           bsoncore.Value
	AllowPartialResults bsoncore.Value
	Collation           bsoncore.Document

	ReadPref         bsoncore.Document
	Database         []byte // a slice of bytes instead of a string to save on allocs.
	LSID             bsoncore.Document
	TxnNumber        bsoncore.Value
	StartTransaction bsoncore.Value
	AutoCommit       bsoncore.Value
	ClusterTime      bsoncore.Document
}

var dollarCmd = [...]byte{'.', '$', 'c', 'm', 'd'}

func (f Find) MarshalWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if err := f.Collection.Validate(); err != nil {
		return dst, err
	}
	if len(f.Database) == 0 {
		return nil, errors.New("Database must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		// OP_QUERY
		flags := f.slaveOK(desc)
		var wmindex int32
		wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
		dst = wiremessagex.AppendQueryFlags(dst, flags)
		// FullCollectionName
		dst = append(dst, f.Database...)
		dst = append(dst, dollarCmd[:]...)
		dst = append(dst, 0x00)
		dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
		dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)
		wrapper := int32(-1)
		if f.ReadPref != nil {
			wrapper, dst = bsoncore.AppendDocumentStart(dst)
			dst = bsoncore.AppendHeader(dst, bsontype.EmbeddedDocument, "$query")
		}
		dst = f.appendCommand(dst, false)
		if f.ReadPref != nil {
			var err error
			dst = bsoncore.AppendDocumentElement(dst, "$readPreference", f.ReadPref)
			dst, err = bsoncore.AppendDocumentEnd(dst, wrapper)
			if err != nil {
				return dst, err
			}
		}
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
		dst = f.appendCommand(dst, true)
		dst = bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:])))
	}
	return dst, nil
}

func (f Find) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	if f.ReadPref == nil {
		// assume primary
		return 0
	}

	if mode, ok := f.ReadPref.Lookup("mode").StringValueOK(); ok && mode != "primary" {
		return wiremessage.SlaveOK
	}

	return 0
}

func (f Find) appendCommand(dst []byte, opmsg bool) []byte {
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = bsoncore.AppendValueElement(dst, "find", f.Collection)

	if opmsg {
		// TODO(GODRIVER-617): Do this without a string conversion.
		dst = bsoncore.AppendStringElement(dst, "$db", string(f.Database))
		if f.ReadPref != nil {
			dst = bsoncore.AppendDocumentElement(dst, "$readPreference", f.ReadPref)
		}
	}

	if f.Filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", f.Filter)
	}
	if f.Sort != nil {
		dst = bsoncore.AppendDocumentElement(dst, "sort", f.Sort)
	}
	if f.Projection != nil {
		dst = bsoncore.AppendDocumentElement(dst, "projection", f.Projection)
	}
	if f.Hint.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "hint", f.Hint)
	}
	if f.Skip.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "skip", f.Skip)
	}
	if f.Limit.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "limit", f.Limit)
	}
	if f.BatchSize.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "batchSize", f.BatchSize)
	}
	if f.SingleBatch.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "singleBatch", f.SingleBatch)
	}
	if f.Comment.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "comment", f.Comment)
	}
	if f.MaxTimeMS.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "maxTimeMS", f.MaxTimeMS)
	}
	if f.ReadConcern != nil {
		dst = bsoncore.AppendDocumentElement(dst, "readConcern", f.ReadConcern)
	}
	if f.Max != nil {
		dst = bsoncore.AppendDocumentElement(dst, "max", f.Max)
	}
	if f.Min != nil {
		dst = bsoncore.AppendDocumentElement(dst, "min", f.Min)
	}
	if f.ReturnKey.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "returnKey", f.ReturnKey)
	}
	if f.ShowRecordID.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "showRecordID", f.ShowRecordID)
	}
	if f.Tailable.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "tailable", f.Tailable)
	}
	if f.OplogReplay.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "oplogReplay", f.OplogReplay)
	}
	if f.NoCursorTimeout.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "noCursorTimeout", f.NoCursorTimeout)
	}
	if f.AwaitData.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "awaitData", f.AwaitData)
	}
	if f.AllowPartialResults.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "allowPartialResults", f.AllowPartialResults)
	}
	if f.Collation != nil {
		dst = bsoncore.AppendDocumentElement(dst, "collation", f.Collation)
	}
	if f.LSID != nil {
		dst = bsoncore.AppendDocumentElement(dst, "lsid", f.LSID)
	}
	if f.TxnNumber.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "txnNumber", f.TxnNumber)
	}
	if f.StartTransaction.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "startTransaction", f.StartTransaction)
	}
	if f.AutoCommit.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "autocommit", f.AutoCommit)
	}
	if f.ClusterTime != nil {
		dst = bsoncore.AppendDocumentElement(dst, "$clusterTime", f.ClusterTime)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	return dst
}
