package session

import "github.com/mongodb/mongo-go-driver/core/description"

// Pool is a pool of server sessions that can be reused.
type Pool struct {
	DescChannel    chan description.Topology // channel to read topology descriptions to update the session timeout
	Head           *ServerSession
	Tail           *ServerSession
	SessionTimeout int
}

// GetSession retrieves an unexpired session from the pool.
func (sp *Pool) GetSession() *ServerSession {
	return nil
}

// ReturnSession returns a session to the pool if it has not expired.
func (sp *Pool) ReturnSession() *ServerSession {
	return nil
}
