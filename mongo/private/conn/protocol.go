// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"context"
	"errors"
	"fmt"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
)

// ExecuteCommand executes the message on the channel.
func ExecuteCommand(ctx context.Context, c Connection, request msg.Request, out interface{}) error {
	return ExecuteCommands(ctx, c, []msg.Request{request}, []interface{}{out})
}

// ExecuteCommands executes the messages on the connection.
func ExecuteCommands(ctx context.Context, c Connection, requests []msg.Request, out []interface{}) error {
	if len(requests) != len(out) {
		return errors.New("invalid arguments. 'out' length must equal 'msgs' length")
	}

	err := c.Write(ctx, requests...)
	if err != nil {
		return internal.WrapErrorf(err, "failed sending commands(%d)", len(requests))
	}

	var errors []error
	for i, req := range requests {
		resp, err := c.Read(ctx, req.RequestID())
		if err != nil {
			return internal.WrapErrorf(err, "failed receiving command response for %d", req.RequestID())
		}

		err = readCommandResponse(resp, out[i])
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}

	return internal.MultiError(errors...)
}

func readCommandResponse(resp msg.Response, out interface{}) error {
	switch typedResp := resp.(type) {
	case *msg.Reply:
		if typedResp.NumberReturned == 0 {
			return ErrNoDocCommandResponse
		}
		if typedResp.NumberReturned > 1 {
			return ErrMultiDocCommandResponse
		}

		if typedResp.ResponseFlags&msg.QueryFailure != 0 {
			// read first document as error
			var doc bson.D
			ok, err := typedResp.Iter().One(&doc)
			if err != nil {
				msg := fmt.Sprintf("failed to read command failure document: %v", err)
				return NewCommandResponseError(msg)
			}
			if !ok {
				return ErrUnknownCommandFailure
			}
			return &CommandFailureError{
				Msg:      "command failure",
				Response: doc,
			}
		}

		// read into raw first
		var raw bson.RawD
		ok, err := typedResp.Iter().One(&raw)
		if err != nil {
			msg := fmt.Sprintf("failed to read command response document: %v", err)
			return NewCommandResponseError(msg)
		}
		if !ok {
			return ErrNoCommandResponse
		}

		// check the raw command response for ok field.
		ok = false
		var errmsg, codeName string
		var code int32
		for _, rawElem := range raw {
			switch rawElem.Name {
			case "ok":
				var v int32
				err := rawElem.Value.Unmarshal(&v)
				if err == nil && v == 1 {
					ok = true
					break
				}
			case "errmsg":
				// Ignore any error that occurs since we're handling malformed documents below.
				_ = rawElem.Value.Unmarshal(&errmsg)
			case "codeName":
				// Ignore any error that occurs since we're handling malformed documents below.
				_ = rawElem.Value.Unmarshal(&codeName)
			case "code":
				// Ignore any error that occurs since we're handling malformed documents below.
				_ = rawElem.Value.Unmarshal(&code)
			}
		}

		if !ok {
			if errmsg == "" {
				errmsg = "command failed"
			}
			return &CommandError{
				Code:    code,
				Message: errmsg,
				Name:    codeName,
			}
		}

		// re-decode the response into the user provided structure...
		ok, err = typedResp.Iter().One(out)
		if err != nil {
			msg := fmt.Sprintf("failed to read command response document: %v", err)
			return NewCommandResponseError(msg)
		}
		if !ok {
			return ErrNoCommandResponse
		}
	default:
		return fmt.Errorf("unsupported response message type: %T", typedResp)
	}

	return nil
}
