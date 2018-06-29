package session

type SessionPool struct {
	Head *ServerSession
	Tail *ServerSession
}

func (sp *SessionPool) GetSession() *ServerSession {
	return nil
}

func (sp *SessionPool) ReturnSession() *ServerSession {
	return nil
}
