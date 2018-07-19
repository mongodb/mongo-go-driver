package command

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// IsProxy is a proxy command.
type IsProxy struct {
	err error
	res bson.Reader

	reqhdr wiremessage.Header
}

// Encode encodes this proxy request.
func (ip *IsProxy) Encode() (wiremessage.WireMessage, error) {
	cmd := bson.NewDocument(bson.EC.Int32("isProxy", 1))

	rdr, err := cmd.MarshalBSON()
	if err != nil {
		return nil, err
	}

	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: "$proxy.$cmd",
		Flags:              wiremessage.SlaveOK,
		NumberToReturn:     -1,
		Query:              rdr,
	}
	return query, nil
}

// Decode decodes this proxy request.
func (ip *IsProxy) Decode(wm wiremessage.WireMessage) *IsProxy {
	switch t := wm.(type) {
	case wiremessage.Query:
		cmd, err := t.Query.ElementAt(0)
		if err != nil {
			ip.err = err
			return ip
		}
		if num, ok := cmd.Value().Int32OK(); cmd.Key() != "isProxy" || !ok || num != 1 {
			ip.err = errors.New("command is not an 'isProxy' command")
			return ip
		}
		ip.reqhdr = t.MsgHeader
		return ip
	case wiremessage.Reply:
		rdr, err := decodeCommandOpReply(t)
		if err != nil {
			ip.err = err
			return ip
		}
		ip.res = rdr
		return ip
	case wiremessage.Msg:
	default:
		ip.err = errors.New("Can only decode OP_QUERY, OP_REPLY, and OP_MSG")
	}

	return ip
}

// Result returns the result of this proxy request.
func (ip *IsProxy) Result() (bson.Reader, error) {
	if ip.err != nil {
		return nil, ip.err
	}

	return ip.res, nil
}

// Err returns the error from this proxy request.
func (ip *IsProxy) Err() error { return ip.err }

// RoundTrip handles roundtripping this proxy request and decoding the response.
func (ip *IsProxy) RoundTrip(ctx context.Context, rw wiremessage.ReadWriter) (bson.Reader, error) {
	wm, err := ip.Encode()
	if err != nil {
		return nil, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return nil, err
	}
	return ip.Decode(wm).Result()
}
