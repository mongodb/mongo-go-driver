package session

import (
	"time"

	"github.com/google/uuid"
	"github.com/mongodb/mongo-go-driver/bson"
)

// Server is an open session with the server.
type Server struct {
	SessionID *bson.Document
	LastUsed  time.Time
}

// returns whether or not a session has expired given a timeout in minutes
// a session is considered expired if it has less than 1 minute left before becoming stale
func (ss *Server) expired(timeoutMinutes uint32) bool {
	timeUnused := time.Since(ss.LastUsed).Minutes()
	return timeUnused > float64(timeoutMinutes-1)
}

func (ss *Server) endSession() {

}

func genUUID() *bson.Document {
	uuidBytes, _ := uuid.New().MarshalBinary()

	return bson.NewDocument(
		bson.EC.BinaryWithSubtype("id", uuidBytes, UUIDSubtype),
	)
}

func newServerSession() *Server {
	return &Server{
		SessionID: genUUID(),
	}
}

// UUIDSubtype is the BSON binary subtype that a UUID should be encoded as
const UUIDSubtype byte = 4
