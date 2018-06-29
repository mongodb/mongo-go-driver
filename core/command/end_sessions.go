package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// must be sent to admin db
// { endSessions: [ {id: uuid}, ... ], $clusterTime: ... }
// only send $clusterTime when gossiping the cluster time
// send 10k sessions at a time

// EndSessions represents an endSessions command.
type EndSessions struct {
	Clock      *session.ClusterClock
	SessionIDs []*bson.Document

	results []result.EndSessions
	errors  []error
}

// BatchSize is the max number of sessions to be included in 1 endSessions command.
const BatchSize = 10000

func (es *EndSessions) split() [][]*bson.Document {
	batches := [][]*bson.Document{}
	docIndex := 0
	totalNumDocs := len(es.SessionIDs)

createBatches:
	for {
		batch := []*bson.Document{}

		for i := 0; i < BatchSize; i++ {
			if docIndex == totalNumDocs {
				break createBatches
			}

			batch = append(batch, es.SessionIDs[docIndex])
			docIndex++
		}

		batches = append(batches, batch)
	}

	return batches
}

func (es *EndSessions) encodeBatch(batch []*bson.Document, desc description.SelectedServer) *Write {
	vals := make([]*bson.Value, 0, len(batch))
	for _, doc := range batch {
		vals = append(vals, bson.VC.Document(doc))
	}

	cmd := bson.NewDocument(
		bson.EC.ArrayFromElements("endSessions", vals...),
	)

	return &Write{
		Clock:   es.Clock,
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

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (es *EndSessions) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *EndSessions {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		es.errors = append(es.errors, err)
		return es
	}

	return es.decode(desc, rdr)
}

func (es *EndSessions) decode(desc description.SelectedServer, rdr bson.Reader) *EndSessions {
	var res result.EndSessions
	es.errors = append(es.errors, bson.Unmarshal(rdr, res))
	es.results = append(es.results, res)
	return es
}

// Result returns the results of the decoded wire messages.
func (es *EndSessions) Result() ([]result.EndSessions, []error) {
	return es.results, es.errors
}

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter
func (es *EndSessions) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) ([]result.EndSessions, []error) {
	cmds := es.encode(desc)

	for _, cmd := range cmds {
		rdr, _ := cmd.RoundTrip(ctx, desc, rw) // ignore any errors returned by the command
		es.decode(desc, rdr)
	}

	return es.Result()
}
