// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
)

func TestDiffHostList(t *testing.T) {
	h1 := "1.0.0.0:27017"
	h2 := "2.0.0.0:27017"
	h3 := "3.0.0.0:27017"
	h4 := "4.0.0.0:27017"
	h5 := "5.0.0.0:27017"
	h6 := "6.0.0.0:27017"
	s1 := description.Server{Addr: "1.0.0.0:27017"}
	s2 := description.Server{Addr: "2.0.0.0:27017"}
	s3 := description.Server{Addr: "3.0.0.0:27017"}
	s6 := description.Server{Addr: "6.0.0.0:27017"}

	topo := description.Topology{
		Servers: []description.Server{s6, s1, s3, s2},
	}
	hostlist := []string{h2, h4, h3, h5}

	diff := diffHostList(topo, hostlist)

	assert.ElementsMatch(t, []string{h4, h5}, diff.Added)
	assert.ElementsMatch(t, []string{h1, h6}, diff.Removed)

	// Ensure that original topology servers and hostlist were not reordered.
	assert.EqualValues(t, []description.Server{s6, s1, s3, s2}, topo.Servers)
	assert.EqualValues(t, []string{h2, h4, h3, h5}, hostlist)
}

func TestDiffTopology(t *testing.T) {
	s1 := description.Server{Addr: "1.0.0.0:27017"}
	s2 := description.Server{Addr: "2.0.0.0:27017"}
	s3 := description.Server{Addr: "3.0.0.0:27017"}
	s4 := description.Server{Addr: "4.0.0.0:27017"}
	s5 := description.Server{Addr: "5.0.0.0:27017"}
	s6 := description.Server{Addr: "6.0.0.0:27017"}

	t1 := description.Topology{
		Servers: []description.Server{s6, s1, s3, s2},
	}
	t2 := description.Topology{
		Servers: []description.Server{s2, s4, s3, s5},
	}

	diff := diffTopology(t1, t2)

	assert.ElementsMatch(t, []description.Server{s4, s5}, diff.Added)
	assert.ElementsMatch(t, []description.Server{s1, s6}, diff.Removed)

	// Ensure that original topology servers were not reordered.
	assert.EqualValues(t, []description.Server{s6, s1, s3, s2}, t1.Servers)
	assert.EqualValues(t, []description.Server{s2, s4, s3, s5}, t2.Servers)
}
