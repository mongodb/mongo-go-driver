// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

// Server represents a server.
type Server interface {
	// Connection gets a connection to use.
	Connection(context.Context) (conn.Connection, error)
	// Model gets the description of the server.
	Model() *model.Server
}

// SelectedServer represents a binding to a server. It contains a
// read preference in the case that needs to be passed on to the
// server during communication.
type SelectedServer struct {
	Server
	// ClusterKind indicates the kind of the cluster the
	// server was selected from.
	ClusterKind model.ClusterKind
	// ReadPref indicates the read preference that should
	// be passed to MongoS. This can be nil.
	ReadPref *readpref.ReadPref
}
