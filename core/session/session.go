package session

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

type ServerSession struct {
	SessionID *bson.Document
	LastUsed  time.Time
}

type ClientSession struct {
	ClusterTime   *bson.Document
	SessionID     *bson.Document
	serverSession *ServerSession
}

func NewClientSession() (*ClientSession, error) {
	// TODO: session pool logic
	return nil, nil
}

func (cs *ClientSession) AdvanceClusterTime(clusterTime *bson.Document) {

}

func (cs *ClientSession) EndSession() {

}
