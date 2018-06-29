package session

import (
	"sync"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ClusterClock represents a logical clock for keeping track of cluster time.
type ClusterClock struct {
	clusterTime *bson.Document
	lock        sync.Mutex
}

// GetClusterTime returns the cluster's current time.
func (cc *ClusterClock) GetClusterTime() *bson.Document {
	var ct *bson.Document
	cc.lock.Lock()
	ct = cc.clusterTime
	cc.lock.Unlock()

	return ct
}

// AdvanceClusterTime updates the cluster's current time.
func (cc *ClusterClock) AdvanceClusterTime(clusterTime *bson.Document) {
	cc.lock.Lock()
	cc.clusterTime = MaxClusterTime(cc.clusterTime, clusterTime)
	cc.lock.Unlock()
}
