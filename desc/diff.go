package desc

import (
	"sort"
	"strings"
)

// DiffCluster returns the difference of two clusters.
func DiffCluster(old, new *Cluster) *ClusterDiff {
	var diff ClusterDiff

	// TODO: do this without sorting...
	oldServers := serverDescSorter(old.Servers)
	newServers := serverDescSorter(new.Servers)

	sort.Sort(oldServers)
	sort.Sort(newServers)

	i := 0
	j := 0
	for {
		if i < len(oldServers) && j < len(newServers) {
			comp := strings.Compare(string(oldServers[i].Endpoint), string(newServers[j].Endpoint))
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

type serverDescSorter []*Server

func (x serverDescSorter) Len() int      { return len(x) }
func (x serverDescSorter) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x serverDescSorter) Less(i, j int) bool {
	return strings.Compare(string(x[i].Endpoint), string(x[j].Endpoint)) < 0
}
