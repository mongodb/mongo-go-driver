// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

// Cluster is a description of a cluster.
type Cluster struct {
	Servers []*Server
	Kind    ClusterKind
}

// Server returns the model.Server with the specified address.
func (i *Cluster) Server(addr Addr) (*Server, bool) {
	for _, server := range i.Servers {
		if server.Addr.String() == addr.String() {
			return server, true
		}
	}
	return nil, false
}
