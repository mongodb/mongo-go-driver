package command

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// IsProxyResponse is the response to an isProxy request.
type IsProxyResponse struct {
	Mode string
}

// Encode handles encoding an isProxy request.
func (ipr *IsProxyResponse) Encode(request *IsProxy) (wiremessage.WireMessage, error) {
	var err error
	resp := bson.NewDocument(bson.EC.Double("ok", 1))

	if ipr.Mode != "" {
		resp.Append(bson.EC.String("proxyMode", ipr.Mode))
	}

	var rdr bson.Reader
	rdr, err = resp.MarshalBSON()
	if err != nil {
		return nil, err
	}

	var wm wiremessage.WireMessage
	switch request.reqhdr.OpCode {
	case wiremessage.OpQuery:
		wm = wiremessage.Reply{
			MsgHeader: wiremessage.Header{
				RequestID:  wiremessage.NextRequestID(),
				ResponseTo: request.reqhdr.RequestID,
			},
			NumberReturned: 1,
			Documents:      []bson.Reader{rdr},
		}
	default:
		return nil, fmt.Errorf("Invalid request op code used %s", request.reqhdr.OpCode)
	}

	return wm, nil
}
