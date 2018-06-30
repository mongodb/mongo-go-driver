package session

import "github.com/mongodb/mongo-go-driver/bson"

// ClientSession is a session for clients to run commands.
type ClientSession struct {
	ClusterTime   *bson.Document
	SessionID     *bson.Document
	serverSession *ServerSession
}

// NewClientSession creates a ClientSession.
func NewClientSession(servSess *ServerSession) *ClientSession {
	return &ClientSession{
		SessionID:     servSess.SessionID,
		serverSession: servSess,
	}
}

// AdvanceClusterTime updates the session's cluster time.
func (cs *ClientSession) AdvanceClusterTime(clusterTime *bson.Document) {

}

// EndSession closes the session.
func (cs *ClientSession) EndSession() {

}
