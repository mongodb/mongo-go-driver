package session

import "github.com/mongodb/mongo-go-driver/bson"

// Client is a session for clients to run commands.
type Client struct {
	ClusterTime   *bson.Document
	SessionID     *bson.Document
	serverSession *Server
}

// NewClientSession creates a Client.
func NewClientSession(servSess *Server) *Client {
	return &Client{
		SessionID:     servSess.SessionID,
		serverSession: servSess,
	}
}

// AdvanceClusterTime updates the session's cluster time.
func (cs *Client) AdvanceClusterTime(clusterTime *bson.Document) {

}

// EndSession closes the session.
func (cs *Client) EndSession() {

}
