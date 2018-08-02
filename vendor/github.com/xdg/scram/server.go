package scram

import "sync"

// Server ...
type Server struct {
	sync.RWMutex
	credentialCB CredentialLookup
	nonceGen     NonceGeneratorFcn
	hashGen      HashGeneratorFcn
}

func newServer(cl CredentialLookup, fcn HashGeneratorFcn) (*Server, error) {
	return &Server{
		credentialCB: cl,
		nonceGen:     defaultNonceGenerator,
		hashGen:      fcn,
	}, nil
}

// WithNonceGenerator ...
func (s *Server) WithNonceGenerator(ng NonceGeneratorFcn) *Server {
	s.Lock()
	defer s.Unlock()
	s.nonceGen = ng
	return s
}

// NewConversation ...
func (s *Server) NewConversation() *ServerConversation {
	s.RLock()
	defer s.RUnlock()
	return &ServerConversation{
		nonceGen:     s.nonceGen,
		hashGen:      s.hashGen,
		credentialCB: s.credentialCB,
	}
}
