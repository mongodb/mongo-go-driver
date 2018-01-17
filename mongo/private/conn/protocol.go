// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"context"
	"fmt"

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
	"github.com/skriptble/wilson/bson"
)

// ExecuteCommand executes the message on the channel.
func ExecuteCommand(ctx context.Context, c Connection, request msg.Request, out interface{}) (bson.Reader, error) {
	readers, err := ExecuteCommands(ctx, c, []msg.Request{request}, []interface{}{out})
	if err != nil {
		return nil, err
	}
	if len(readers) < 1 {
		return nil, nil
	}
	return readers[0], nil
}

// ExecuteCommands executes the messages on the connection.
func ExecuteCommands(ctx context.Context, c Connection, requests []msg.Request, out []interface{}) ([]bson.Reader, error) {
	if len(requests) != len(out) {
		panic("invalid arguments. 'out' length must equal 'msgs' length")
	}

	err := c.Write(ctx, requests...)
	if err != nil {
		return nil, internal.WrapErrorf(err, "failed sending commands(%d)", len(requests))
	}

	var errors []error
	var readers = make([]bson.Reader, len(requests))
	for i, req := range requests {
		resp, err := c.Read(ctx, req.RequestID())
		if err != nil {
			return nil, internal.WrapErrorf(err, "failed receiving command response for %d", req.RequestID())
		}

		r, err := readCommandResponse(resp, out[i])
		if err != nil {
			errors = append(errors, err)
			continue
		}
		readers[i] = r
	}

	return readers, internal.MultiError(errors...)
}

func readCommandResponse(resp msg.Response, out interface{}) (bson.Reader, error) {
	switch typedResp := resp.(type) {
	case *msg.Reply:
		if typedResp.NumberReturned == 0 {
			return nil, ErrNoDocCommandResponse
		}
		if typedResp.NumberReturned > 1 {
			return nil, ErrMultiDocCommandResponse
		}

		if typedResp.ResponseFlags&msg.QueryFailure != 0 {
			// read first document as error
			r, err := typedResp.Iter().DecodeBytes()
			switch {
			case err != nil:
				msg := fmt.Sprintf("failed to read command failure document: %v", err)
				return nil, NewCommandResponseError(msg)
			case r == nil && err == nil:
				return nil, ErrUnknownCommandFailure
			}
			return nil, &CommandFailureError{
				Msg:      "command failure",
				Response: r,
			}
		}

		// read into raw first
		r, err := typedResp.Iter().DecodeBytes()
		if err != nil {
			msg := fmt.Sprintf("failed to read command response document: %v", err)
			return nil, NewCommandResponseError(msg)
		}
		if r == nil {
			return nil, ErrNoCommandResponse
		}

		// check the raw command response for ok field.
		ok := false
		var errmsg, codeName string
		var code int32
		itr, err := r.Iterator()
		if err != nil {
			return nil, err
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
				switch elem.Value().Type() {
				case bson.TypeString:
					errmsg = elem.Value().StringValue()
				}
			case "codeName":
				switch elem.Value().Type() {
				case bson.TypeString:
					codeName = elem.Value().StringValue()
				}
			case "code":
				switch elem.Value().Type() {
				case bson.TypeInt32:
					code = elem.Value().Int32()
				}
			}
		}

		if !ok {
			if errmsg == "" {
				errmsg = "command failed"
			}
			return nil, &CommandError{
				Code:    code,
				Message: errmsg,
				Name:    codeName,
			}
		}

		// 	// check the raw command response for ok field.
		// 	ok := false
		// 	var errmsg, codeName string
		// 	var code int32
		// 	iter, err := r.Iterator()
		// 	if err != nil {
		// 		// TODO(skriptble): This should probably be something like a
		// 		// malformed response.
		// 		return nil, ErrNoCommandResponse
		// 	}
		// loop:
		// 	for iter.Next() {
		// 		elem := iter.Element()
		// 		val := elem.Value()
		// 		switch elem.Key() {
		// 		case "ok":
		// 			switch val.Type() {
		// 			case bson.TypeInt32:
		// 				if val.Int32() == 1 {
		// 					ok = true
		// 					break loop
		// 				}
		// 			case bson.TypeInt64:
		// 				if val.Int64() == 1 {
		// 					ok = true
		// 					break loop
		// 				}
		// 			}
		// 		case "errmsg":
		// 			// Ignore any error that occurs since we're handling malformed documents below.
		// 			if val.Type() != bson.TypeString {
		// 				continue
		// 			}
		// 			errmsg = val.StringValue()
		// 		case "codeName":
		// 			// Ignore any error that occurs since we're handling malformed documents below.
		// 			if val.Type() != bson.TypeString {
		// 				continue
		// 			}
		// 			codeName = val.StringValue()
		// 		case "code":
		// 			// Ignore any error that occurs since we're handling malformed documents below.
		// 			if val.Type() != bson.TypeInt32 {
		// 				continue
		// 			}
		// 			code = val.Int32()
		// 		}
		// 	}
		//
		// 	if !ok {
		// 		if errmsg == "" {
		// 			errmsg = "command failed"
		// 		}
		// 		return nil, &CommandError{
		// 			Code:    code,
		// 			Message: errmsg,
		// 			Name:    codeName,
		// 		}
		// 	}
		// re-decode the response into the user provided structure...
		if out == nil {
			return r, nil
		}
		ok, err = typedResp.Iter().One(out)
		if err != nil {
			msg := fmt.Sprintf("failed to read command response document: %v", err)
			return nil, NewCommandResponseError(msg)
		}
		if !ok {
			return nil, ErrNoCommandResponse
		}

		return r, nil
	default:
		return nil, fmt.Errorf("unsupported response message type: %T", typedResp)
	}

	return nil, nil
}
