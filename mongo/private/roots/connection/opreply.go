package connection

import (
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

func decodeOpReply(reply wiremessage.Reply) (bson.Reader, error) {
	if reply.NumberReturned == 0 {
		return nil, errors.New("unknown command failure")
	}
	if reply.NumberReturned > 1 {
		return nil, errors.New("no command response document")
	}
	if len(reply.Documents) != 1 {
		return nil, errors.New("malformed OP_REPLY: NumberReturned does not match number of documents returned")
	}
	rdr := reply.Documents[0]
	_, err := rdr.Validate()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("malformed OP_REPLY: invalid document: %s", err))
	}
	if reply.ResponseFlags&wiremessage.QueryFailure == wiremessage.QueryFailure {
		return nil, errors.New(fmt.Sprintf("command failure: %v", rdr))
	}

	ok := false
	var errmsg, codeName string
	var code int32
	itr, err := rdr.Iterator()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("malformed OP_REPLY: cannot iterate document: %s", err))
	}
	for itr.Next() {
		elem := itr.Element()
		switch elem.Key() {
		case "ok":
			switch elem.Value().Type() {
			case bson.TypeInt32:
				if elem.Value().Int32() == 1 {
					ok = true
				}
			case bson.TypeInt64:
				if elem.Value().Int64() == 1 {
					ok = true
				}
			case bson.TypeDouble:
				if elem.Value().Double() == 1 {
					ok = true
				}
			}
		case "errmsg":
			if str, okay := elem.Value().StringValueOK(); okay {
				errmsg = str
			}
		case "codeName":
			if str, okay := elem.Value().StringValueOK(); okay {
				codeName = str
			}
		case "code":
			if c, okay := elem.Value().Int32OK(); okay {
				code = c
			}
		}
	}

	if !ok {
		if errmsg == "" {
			errmsg = "command failed"
		}
		if codeName != "" {
			return nil, errors.New(fmt.Sprintf("code[%d]: (%s) %s", code, codeName, errmsg))
		}
		return nil, errors.New(errmsg)
	}

	return rdr, nil
}
