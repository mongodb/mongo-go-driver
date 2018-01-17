package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
	"github.com/skriptble/wilson/bson"
)

// FindOneAndReplace modifies and returns a single document.
func FindOneAndReplace(ctx context.Context, s *SelectedServer, ns Namespace,
	writeConcern *writeconcern.WriteConcern, query *bson.Document, replacement *bson.Document,
	result interface{}, opts ...options.FindOneAndReplaceOption) (bool, error) {

	if err := ns.validate(); err != nil {
		return false, err
	}

	command := bson.NewDocument()
	command.Append(
		bson.C.String("findAndModify", ns.Collection),
		bson.C.SubDocument("query", query),
		bson.C.SubDocument("update", replacement),
	)
	// command := oldbson.D{
	// 	{Name: "findAndModify", Value: ns.Collection},
	// 	{Name: "query", Value: query},
	// 	{Name: "update", Value: replacement},
	// }

	for _, option := range opts {
		option.Option(command)
		// switch name := option.FindOneAndReplaceName(); name {
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.FindOneAndReplaceValue().(time.Duration)/time.Millisecond),
		// 	)
		// default:
		// 	command.AppendElem(name, option.FindOneAndReplaceValue())
		// }
	}

	if writeConcern != nil {
		elem, err := writeConcern.MarshalBSONElement()
		if err != nil {
			return false, err
		}
		command.Append(elem)
		// command.AppendElem("writeConcern", writeConcern)
	}

	rdr, err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return false, internal.WrapError(err, "failed to execute count")
	}

	if elem, err := rdr.Lookup("value"); err == nil {
		if elem.Value().Type() == bson.TypeNull {
			return false, nil
		}
	}

	if result == nil {
		return true, nil
	}

	return true, nil
}
