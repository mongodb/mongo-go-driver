package session

import "github.com/mongodb/mongo-go-driver/bson"

// Client is a session for clients to run commands.
type Client struct {
	ClusterTime *bson.Document
	SessionID   *bson.Document
	SessionType Type

	pool          *Pool
	terminated    bool
	serverSession *Server
}

func getClusterTime(clusterTime *bson.Document) (uint32, uint32) {
	if clusterTime == nil {
		return 0, 0
	}

	clusterTimeVal, err := clusterTime.LookupErr("$clusterTime")
	if err != nil {
		return 0, 0
	}

	timestampVal, err := clusterTimeVal.MutableDocument().LookupErr("clusterTime")
	if err != nil {
		return 0, 0
	}

	return timestampVal.Timestamp()
}

// MaxClusterTime compares 2 clusterTime documents and returns the document representing the highest cluster time.
func MaxClusterTime(ct1 *bson.Document, ct2 *bson.Document) *bson.Document {
	epoch1, ord1 := getClusterTime(ct1)
	epoch2, ord2 := getClusterTime(ct2)

	if epoch1 > epoch2 {
		return ct1
	} else if epoch1 < epoch2 {
		return ct2
	} else if ord1 > ord2 {
		return ct1
	} else if ord1 < ord2 {
		return ct2
	}

	return ct1
}

// NewClientSession creates a Client.
func NewClientSession(pool *Pool, sessionType Type) (*Client, error) {
	servSess, err := pool.GetSession()
	if err != nil {
		return nil, err
	}

	return &Client{
		SessionID:     servSess.SessionID,
		SessionType:   sessionType,
		pool:          pool,
		serverSession: servSess,
	}, nil
}

// AdvanceClusterTime updates the session's cluster time.
func (c *Client) AdvanceClusterTime(clusterTime *bson.Document) {
	c.ClusterTime = MaxClusterTime(c.ClusterTime, clusterTime)
}

// UpdateUseTime updates the session's last used time.
// Must be called whenver this session is used to send a command to the server.
func (c *Client) UpdateUseTime() {
	c.serverSession.updateUseTime()
}

// EndSession ends the session.
func (c *Client) EndSession() {
	if c.terminated {
		return
	}

	c.terminated = true
	c.pool.ReturnSession(c.serverSession)
	return
}

// Type describes the type of the session
type Type uint8

// These constants are the valid types for a client session.
const (
	Explicit Type = iota
	Implicit
)

// MinWireVersion is the minimum wire version that supports $clusterTime in messages
const MinWireVersion = 6
