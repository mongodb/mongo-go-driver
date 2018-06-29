package session

// Pool is a pool of server sessions that can be reused.
type Pool struct {
	Head *ServerSession
	Tail *ServerSession
}

// GetSession retrieves an unexpired session from the pool.
func (sp *Pool) GetSession() *ServerSession {
	return nil
}

// ReturnSession returns a session to the pool if it has not expired.
func (sp *Pool) ReturnSession() *ServerSession {
	return nil
}
