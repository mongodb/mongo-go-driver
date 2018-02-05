package ops

import (
	"context"
	"errors"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
)

// FindOneAndUpdate modifies and returns a single document.
func FindOneAndUpdate(ctx context.Context, s *SelectedServer, ns Namespace,
	writeConcern *writeconcern.WriteConcern, query *bson.Document, update *bson.Document,
	opts ...options.FindOneAndUpdateOptioner) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(
		bson.C.String("findAndModify", ns.Collection),
		bson.C.SubDocument("query", query),
		bson.C.SubDocument("update", update),
	)

	for _, option := range opts {
		if option == nil {
			continue
		}
		option.Option(command)
	}

	if writeConcern != nil {
		elem, err := writeConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}
		command.Append(elem)
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute count")
	}

	val, err := rdr.Lookup("value")
	switch {
	case err == bson.ErrElementNotFound:
		return nil, errors.New("Invalid response from server, no value field")
	case err != nil:
		return nil, err
	}

	switch val.Value().Type() {
	case bson.TypeNull:
		return &singleResultCursor{}, nil
	case bson.TypeEmbeddedDocument:
		return &singleResultCursor{rdr: val.Value().ReaderDocument()}, nil
	default:
		return nil, errors.New("Invalid response from server, value field is not a document")
	}
}
