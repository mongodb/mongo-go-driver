// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// decodeCommandOpReply handles decoding the OP_REPLY response to an OP_QUERY
// command.
func decodeCommandOpReply(reply wiremessage.Reply) (bson.Reader, error) {
	if reply.NumberReturned == 0 {
		return nil, ErrNoDocCommandResponse
	}
	if reply.NumberReturned > 1 {
		return nil, ErrMultiDocCommandResponse
	}
	if len(reply.Documents) != 1 {
		return nil, NewCommandResponseError("malformed OP_REPLY: NumberReturned does not match number of documents returned", nil)
	}
	rdr := reply.Documents[0]
	_, err := rdr.Validate()
	if err != nil {
		return nil, NewCommandResponseError("malformed OP_REPLY: invalid document", err)
	}
	if reply.ResponseFlags&wiremessage.QueryFailure == wiremessage.QueryFailure {
		return nil, QueryFailureError{
			Message:  "command failure",
			Response: reply.Documents[0],
		}
	}

	ok := false
	var errmsg, codeName string
	var code int32
	itr, err := rdr.Iterator()
	if err != nil {
		return nil, NewCommandResponseError("malformed OP_REPLY: cannot iterate document", err)
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
		return nil, Error{
			Code:    code,
			Message: errmsg,
			Name:    codeName,
		}
	}

	return rdr, nil
}
