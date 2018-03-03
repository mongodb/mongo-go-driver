package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Insert represents the insert command.
//
// The insert command inserts a set of documents into the database.
//
// Since the Insert command does not return any value other than ok or
// an error, this type has no Err method.
type Insert struct {
	NS   Namespace
	Docs []*bson.Document
	Opts []options.InsertOptioner

	result result.Insert
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (i *Insert) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	command := bson.NewDocument(bson.EC.String("insert", i.NS.Collection))
	vals := make([]*bson.Value, 0, len(i.Docs))
	for _, doc := range i.Docs {
		vals = append(vals, bson.VC.Document(doc))
	}
	command.Append(bson.EC.ArrayFromElements("documents", vals...))

	for _, option := range i.Opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	return (&Command{DB: i.NS.DB, Command: command, isWrite: true}).Encode(desc)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (i *Insert) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Insert {
	rdr, err := (&Command{}).Decode(desc, wm).Result()
	if err != nil {
		i.err = err
		return i
	}

	i.err = bson.Unmarshal(rdr, &i.result)
	return i
}

// Result returns the result of a decoded wire message and server description.
func (i *Insert) Result() (result.Insert, error) {
	if i.err != nil {
		return result.Insert{}, i.err
	}
	return i.result, nil
}

// Err returns the error set on this command.
func (i *Insert) Err() error { return i.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (i *Insert) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.Insert, error) {
	wm, err := i.Encode(desc)
	if err != nil {
		return result.Insert{}, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.Insert{}, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return result.Insert{}, err
	}
	return i.Decode(desc, wm).Result()
}
