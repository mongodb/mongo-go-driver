package command

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

// Count represents the count command.
//
// The count command counts how many documents in a collection match the given query.
type Count struct {
	NS       Namespace
	Query    *bson.Document
	Opts     []options.CountOptioner
	ReadPref *readpref.ReadPref

	result int64
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (c *Count) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	if err := c.NS.Validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.EC.String("count", c.NS.Collection), bson.EC.SubDocument("query", c.Query))
	for _, option := range c.Opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	return (&Command{DB: c.NS.DB, ReadPref: c.ReadPref, Command: command}).Encode(desc)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Count) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Count {
	rdr, err := (&Command{}).Decode(desc, wm).Result()
	if err != nil {
		c.err = err
		return c
	}

	val, err := rdr.Lookup("n")
	switch {
	case err == bson.ErrElementNotFound:
		c.err = errors.New("Invalid response from server, no 'n' field")
		return c
	case err != nil:
		c.err = err
		return c
	}

	switch val.Value().Type() {
	case bson.TypeDouble:
		c.result = int64(val.Value().Double())
	case bson.TypeInt32:
		c.result = int64(val.Value().Int32())
	case bson.TypeInt64:
		c.result = val.Value().Int64()
	default:
		c.err = errors.New("Invalid response from server, value field is not a number")
	}

	return c
}

// Result returns the result of a decoded wire message and server description.
func (c *Count) Result() (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.result, nil
}

// Err returns the error set on this command.
func (c *Count) Err() error { return c.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (c *Count) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (int64, error) {
	wm, err := c.Encode(desc)
	if err != nil {
		return 0, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return 0, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return 0, err
	}
	return c.Decode(desc, wm).Result()
}
