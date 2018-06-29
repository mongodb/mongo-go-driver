package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// must be sent to admin db
// { endSessions: [ {id: uuid}, ... ], $clusterTime: ... }
// only send $clusterTime when gossiping the cluster time
// send 10k sessions at a time

// EndSessions represents an endSessions command.
type EndSessions struct {
	SessionPool *session.Pool
}

// BatchSize is the max number of sessions to be included in 1 endSessions command.
const BatchSize = 10000

func (es *EndSessions) split() [][]*bson.Document {
	batches := [][]*bson.Document{}
	outerIndex := 0
	iter := es.SessionPool.Iterator()

createBatches:
	for {
		currBatch := batches[outerIndex]

		for i := 0; i < BatchSize; i++ {
			if !iter.Next() {
				// out of sessions
				break createBatches
			}

			nextSess := iter.Element()
			currBatch = append(currBatch, bson.NewDocument(
				bson.EC.SubDocument("id", nextSess.SessionID),
			))
		}

		outerIndex++
	}

	return batches
}

func (es *EndSessions) encodeBatch(batch []*bson.Document, desc description.SelectedServer) *Write {
	vals := make([]*bson.Value, 0, len(batch))
	for _, doc := range batch {
		vals = append(vals, bson.VC.Document(doc))
	}

	cmd := bson.NewDocument(bson.EC.ArrayFromElements("endSessions", vals...))
	return &Write{
		DB:      "admin",
		Command: cmd,
	}
}

// Encode will encode this command into a series of wire messages for the given server description.
func (es *EndSessions) Encode(desc description.SelectedServer) ([]wiremessage.WireMessage, error) {
	cmds := es.encode(desc)
	wms := make([]wiremessage.WireMessage, len(cmds))

	for _, cmd := range cmds {
		wm, err := cmd.Encode(desc)
		if err != nil {
			return nil, err
		}

		wms = append(wms, wm)
	}

	return wms, nil
}

func (es *EndSessions) encode(desc description.SelectedServer) []*Write {
	out := []*Write{}
	batches := es.split()

	for _, batch := range batches {
		out = append(out, es.encodeBatch(batch, desc))
	}

	return out
}

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter
func (es *EndSessions) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) error {
	cmds := es.encode(desc)

	for _, cmd := range cmds {
		_, _ = cmd.RoundTrip(ctx, desc, rw) // ignore any errors returned by the command
	}

	return nil
}
