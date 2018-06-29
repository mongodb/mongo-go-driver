package session

import (
	"sync"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ClusterClock represents a logical clock for keeping track of cluster time.
type ClusterClock struct {
	clusterTime *bson.Document
	rwLock      sync.RWMutex
}

// GetClusterTime returns the cluster's current time.
func (cc *ClusterClock) GetClusterTime() *bson.Document {
	var ct *bson.Document
	cc.rwLock.RLock()
	ct = cc.clusterTime
	cc.rwLock.RUnlock()

	return ct
}

// AdvanceClusterTime updates the cluster's current time.
func (cc *ClusterClock) AdvanceClusterTime(clusterTime *bson.Document) {
	cc.rwLock.Lock()
	cc.clusterTime = MaxClusterTime(cc.clusterTime, clusterTime)
	cc.rwLock.Unlock()
}
