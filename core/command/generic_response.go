package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// GenericResponse is a generic response.
type GenericResponse struct {
	OK    bool
	Error *Error
	Resp  *bson.Document
}

// Encode encodes this generic response.
func (gr *GenericResponse) Encode(reqhdr wiremessage.Header) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode decodes this generic response.
func (gr *GenericResponse) Decode(wm wiremessage.WireMessage) *GenericResponse {
	return gr
}

// Result returns the result of this response.
func (gr *GenericResponse) Result() (bson.Reader, error) {
	return nil, nil
}

// Err returns the error from this response.
func (gr *GenericResponse) Err() error { return nil }

func (gr *GenericResponse) encodeOpMsg() (wiremessage.Msg, error)                  { return wiremessage.Msg{}, nil }
func (gr *GenericResponse) encodeOpReply() (wiremessage.Reply, error)              { return wiremessage.Reply{}, nil }
func (gr *GenericResponse) decodeOpMsg(msg wiremessage.Msg) *GenericResponse       { return gr }
func (gr *GenericResponse) decodeOpReply(reply wiremessage.Reply) *GenericResponse { return gr }
