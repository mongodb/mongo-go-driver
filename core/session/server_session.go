package session

import (
	"time"

	"crypto/rand"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/uuid"
)

var rander = rand.Reader

// Server is an open session with the server.
type Server struct {
	SessionID *bson.Document
	LastUsed  time.Time
}

// returns whether or not a session has expired given a timeout in minutes
// a session is considered expired if it has less than 1 minute left before becoming stale
func (ss *Server) expired(timeoutMinutes uint32) bool {
	if timeoutMinutes <= 0 {
		return true
	}
	timeUnused := time.Since(ss.LastUsed).Minutes()
	return timeUnused > float64(timeoutMinutes-1)
}

// update the last used time for this session.
// must be called whenever this server session is used to send a command to the server.
func (ss *Server) updateUseTime() {
	ss.LastUsed = time.Now()
}

func newServerSession() (*Server, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, err
	}

	idDoc := bson.NewDocument(
		bson.EC.BinaryWithSubtype("id", id[:], UUIDSubtype),
	)

	return &Server{
		SessionID: idDoc,
		LastUsed:  time.Now(),
	}, nil
}

// UUIDSubtype is the BSON binary subtype that a UUID should be encoded as
const UUIDSubtype byte = 4
