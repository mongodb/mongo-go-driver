// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

import (
	"sort"
	"strings"
)

// DiffCluster returns the difference of two clusters.
func DiffCluster(old, new *Cluster) *ClusterDiff {
	var diff ClusterDiff

	// TODO: do this without sorting...
	oldServers := serverSorter(old.Servers)
	newServers := serverSorter(new.Servers)

	sort.Sort(oldServers)
	sort.Sort(newServers)

	i := 0
	j := 0
	for {
		if i < len(oldServers) && j < len(newServers) {
			comp := strings.Compare(oldServers[i].Addr.String(), newServers[j].Addr.String())
			switch comp {
			case 1:
				//left is bigger than
				diff.AddedServers = append(diff.AddedServers, newServers[j])
				j++
			case -1:
				// right is bigger
				diff.RemovedServers = append(diff.RemovedServers, oldServers[i])
				i++
			case 0:
				i++
				j++
			}
		} else if i < len(oldServers) {
			diff.RemovedServers = append(diff.RemovedServers, oldServers[i])
			i++
		} else if j < len(newServers) {
			diff.AddedServers = append(diff.AddedServers, newServers[j])
			j++
		} else {
			break
		}
	}

	return &diff
}

// ClusterDiff is the difference between two clusters.
type ClusterDiff struct {
	AddedServers   []*Server
	RemovedServers []*Server
}

type serverSorter []*Server

func (x serverSorter) Len() int      { return len(x) }
func (x serverSorter) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x serverSorter) Less(i, j int) bool {
	return strings.Compare(x[i].Addr.String(), x[j].Addr.String()) < 0
}
