// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
)

// Server represents a logical connection to a server.
type Server interface {
	// Connection gets a connection to the server.
	Connection(context.Context) (conn.Connection, error)
	// Model gets a description of the server.
	Model() *model.Server
}
