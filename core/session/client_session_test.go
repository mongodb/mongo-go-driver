package session

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/uuid"
	"github.com/stretchr/testify/require"
)

func TestClientSession(t *testing.T) {
	var clusterTime1 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 10, 5))))
	var clusterTime2 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 5, 5))))
	var clusterTime3 = bson.NewDocument(bson.EC.SubDocument("$clusterTime",
		bson.NewDocument(bson.EC.Timestamp("clusterTime", 5, 0))))

	t.Run("TestMaxClusterTime", func(t *testing.T) {
		maxTime := MaxClusterTime(clusterTime1, clusterTime2)
		if maxTime != clusterTime1 {
			t.Errorf("Wrong max time")
		}

		maxTime = MaxClusterTime(clusterTime3, clusterTime2)
		if maxTime != clusterTime2 {
			t.Errorf("Wrong max time")
		}
	})

	t.Run("TestAdvanceClusterTime", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit)
		require.Nil(t, err, "Unexpected error")
		err = sess.AdvanceClusterTime(clusterTime2)
		require.Nil(t, err, "Unexpected error")
		if sess.ClusterTime != clusterTime2 {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime2, sess.ClusterTime)
		}
		err = sess.AdvanceClusterTime(clusterTime3)
		require.Nil(t, err, "Unexpected error")
		if sess.ClusterTime != clusterTime2 {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime2, sess.ClusterTime)
		}
		err = sess.AdvanceClusterTime(clusterTime1)
		require.Nil(t, err, "Unexpected error")
		if sess.ClusterTime != clusterTime1 {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime1, sess.ClusterTime)
		}
		sess.EndSession()
	})

	t.Run("TestEndSession", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit)
		require.Nil(t, err, "Unexpected error")
		sess.EndSession()
		err = sess.UpdateUseTime()
		require.NotNil(t, err, "Expected error, received nil")
	})
}
