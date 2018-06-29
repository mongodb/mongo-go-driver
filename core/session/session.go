package session

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ServerSession is an open session with the server.
type ServerSession struct {
	SessionID *bson.Document
	LastUsed  time.Time
}

// ClientSession is a session for clients to run commands.
type ClientSession struct {
	ClusterTime   *bson.Document
	SessionID     *bson.Document
	serverSession *ServerSession
}

// NewClientSession creates a ClientSession.
func NewClientSession() (*ClientSession, error) {
	return nil, nil
}

// AdvanceClusterTime updates the session's cluster time.
func (cs *ClientSession) AdvanceClusterTime(clusterTime *bson.Document) {

}

// EndSession closes the session.
func (cs *ClientSession) EndSession() {

}
