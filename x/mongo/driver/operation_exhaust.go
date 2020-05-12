// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

// ExecuteExhaust gets a connection from the provided deployment and reads a response from it. This will error if the
// connection CurrentlyStreaming function returns false.
func (op Operation) ExecuteExhaust(ctx context.Context, conn Streamer, scratch []byte) error {
	if !conn.CurrentlyStreaming() {
		return errors.New("exhaust read must be done with a Streamer that is currently streaming")
	}

	scratch = scratch[:0]
	res, err := op.readWireMessage(ctx, conn, scratch)
	if err != nil {
		return err
	}
	if op.ProcessResponseFn != nil {
		if err = op.ProcessResponseFn(res, nil, description.Server{}); err != nil {
			return err
		}
	}

	return nil
}
