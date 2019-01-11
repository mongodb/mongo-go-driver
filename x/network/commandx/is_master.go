package commandx

import (
	"runtime"

	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/version"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessagex"
)

type IsMaster struct {
	Client             bsoncore.Document
	Compressors        bsoncore.Value
	SASLSupportedMechs bsoncore.Value
}

var isMasterNS = [...]byte{'a', 'd', 'm', 'i', 'n', '.', '$', 'c', 'm', 'd', 0x00}

func (im IsMaster) MarshalWireMessage(dst []byte, _ description.SelectedServer) ([]byte, error) {
	// isMaster always uses OP_QUERY
	flags := wiremessage.SlaveOK
	wmindex, dst := wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
	dst = wiremessagex.AppendQueryFlags(dst, flags)
	// FullCollectionName
	dst = append(dst, isMasterNS[:]...)
	dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
	dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)

	// command
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = bsoncore.AppendInt32Element(dst, "isMaster", 1)

	if im.Client != nil {
		dst = bsoncore.AppendDocumentElement(dst, "client", im.Client)
	}
	if im.SASLSupportedMechs.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "saslSupportedMechs", im.SASLSupportedMechs)
	}
	if im.Compressors.Type != bsontype.Type(0) {
		dst = bsoncore.AppendValueElement(dst, "compression", im.Compressors)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	dst = bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:])))
	return dst, nil
}

// ClientDoc creates a client information document for use in an isMaster
// command.
func ClientDoc(app string) bsoncore.Document {
	cidx, doc := bsoncore.AppendDocumentStart(nil)
	driverIdx, doc := bsoncore.AppendDocumentElementStart(doc, "driver")
	doc = bsoncore.AppendStringElement(doc, "name", "mongo-go-driver")
	doc = bsoncore.AppendStringElement(doc, "version", version.Driver)
	doc, _ = bsoncore.AppendDocumentEnd(doc, driverIdx)
	osIdx, doc := bsoncore.AppendDocumentElementStart(doc, "os")
	doc = bsoncore.AppendStringElement(doc, "type", runtime.GOOS)
	doc = bsoncore.AppendStringElement(doc, "architecture", runtime.GOARCH)
	doc, _ = bsoncore.AppendDocumentEnd(doc, osIdx)
	doc = bsoncore.AppendStringElement(doc, "platform", runtime.Version())

	if app != "" {
		var appIdx int32
		appIdx, doc = bsoncore.AppendDocumentElementStart(doc, "application")
		doc = bsoncore.AppendStringElement(doc, "name", app)
		doc, _ = bsoncore.AppendDocumentEnd(doc, appIdx)
	}

	doc, _ = bsoncore.AppendDocumentEnd(doc, cidx)
	return doc
}
