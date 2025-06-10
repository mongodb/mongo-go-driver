// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
)

type changeStreamDeployment struct {
	topologyKind description.TopologyKind
	server       driver.Server
	conn         *mnet.Connection
}

var _ driver.Deployment = (*changeStreamDeployment)(nil)
var _ driver.Server = (*changeStreamDeployment)(nil)
var _ driver.ErrorProcessor = (*changeStreamDeployment)(nil)

func (c *changeStreamDeployment) SelectServer(context.Context, description.ServerSelector) (driver.Server, error) {

	return c, nil
}

func (c *changeStreamDeployment) Kind() description.TopologyKind {
	return c.topologyKind
}

func (c *changeStreamDeployment) Connection(context.Context) (*mnet.Connection, error) {
	return c.conn, nil
}

func (c *changeStreamDeployment) RTTMonitor() driver.RTTMonitor {
	return c.server.RTTMonitor()
}

func (c *changeStreamDeployment) ProcessError(err error, describer mnet.Describer) driver.ProcessErrorResult {
	ep, ok := c.server.(driver.ErrorProcessor)
	if !ok {
		return driver.NoChange
	}

	return ep.ProcessError(err, describer)
}

// GetServerSelectionTimeout returns zero as a server selection timeout is not
// applicable for change stream deployments.
func (*changeStreamDeployment) GetServerSelectionTimeout() time.Duration {
	return 0
}
