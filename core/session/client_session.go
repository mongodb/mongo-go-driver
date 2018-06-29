package session

import "github.com/mongodb/mongo-go-driver/bson"

// Client is a session for clients to run commands.
type Client struct {
	ClusterTime *bson.Document
	SessionID   *bson.Document

	pool          *Pool
	terminated    bool
	serverSession *Server
}

// NewClientSession creates a Client.
func NewClientSession(pool *Pool) (*Client, error) {
	servSess, err := pool.GetSession()
	if err != nil {
		return nil, err
	}

	return &Client{
		SessionID:     servSess.SessionID,
		serverSession: servSess,
	}, nil
}

func getClusterTime(clusterTime *bson.Document) (uint32, uint32) {
	if clusterTime == nil {
		return 0, 0
	}

	return clusterTime.Lookup("clusterTime").Timestamp()
}

// AdvanceClusterTime updates the session's cluster time.
func (c *Client) AdvanceClusterTime(clusterTime *bson.Document) {
	currEpochTime, _ := getClusterTime(c.ClusterTime)
	newEpochTime, _ := getClusterTime(clusterTime)

	if newEpochTime > currEpochTime {
		c.ClusterTime = clusterTime
	}
}

// EndSession ends the session.
func (c *Client) EndSession() {
	if c.terminated {
		return
	}

	c.pool.ReturnSession(c.serverSession)
	return
}
