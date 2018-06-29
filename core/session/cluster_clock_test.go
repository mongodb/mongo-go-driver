package session

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
)

func TestClusterClock(t *testing.T) {
	var clusterTime1 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 10, 5))))
	var clusterTime2 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 5, 5))))
	var clusterTime3 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 5, 0))))

	t.Run("ClusterTime", func(t *testing.T) {
		clock := ClusterClock{}
		clock.AdvanceClusterTime(clusterTime3)
		done := make(chan struct{})
		go func() {
			clock.AdvanceClusterTime(clusterTime1)
			done <- struct{}{}
		}()
		clock.AdvanceClusterTime(clusterTime2)

		<-done
		if clock.GetClusterTime() != clusterTime1 {
			t.Errorf("Expected cluster time %v, received %v", clusterTime1, clock.GetClusterTime())
		}
	})
}
