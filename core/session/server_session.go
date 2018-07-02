package session

import (
	"time"

	"crypto/rand"
	"io"

	"github.com/mongodb/mongo-go-driver/bson"
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
	timeUnused := time.Since(ss.LastUsed).Minutes()
	return timeUnused > float64(timeoutMinutes-1)
}

func (ss *Server) endSession() {

}

func genUUID() (*bson.Document, error) {
	var uuid [16]byte

	_, err := io.ReadFull(rander, uuid[:])
	if err != nil {
		return nil, err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return bson.NewDocument(
		bson.EC.BinaryWithSubtype("id", uuid[:], UUIDSubtype),
	), nil
}

func newServerSession() (*Server, error) {
	uuid, err := genUUID()
	if err != nil {
		return nil, err
	}

	return &Server{
		SessionID: uuid,
	}, nil
}

// UUIDSubtype is the BSON binary subtype that a UUID should be encoded as
const UUIDSubtype byte = 4
