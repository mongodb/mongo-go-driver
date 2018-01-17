package ops

import (
	"context"

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
	"github.com/skriptble/wilson/bson"
)

// FindOneAndDelete modifies and returns a single document.
func FindOneAndDelete(ctx context.Context, s *SelectedServer, ns Namespace,
	writeConcern *writeconcern.WriteConcern, query *bson.Document, result interface{},
	opts ...options.FindOneAndDeleteOption) (bool, error) {

	if err := ns.validate(); err != nil {
		return false, err
	}

	command := bson.NewDocument(3 + uint(len(opts)))
	command.Append(
		bson.C.String("findAndModify", ns.Collection),
		bson.C.SubDocument("query", query),
		bson.C.Boolean("remove", true),
	)
	// command := oldbson.D{
	// 	{Name: "findAndModify", Value: ns.Collection},
	// 	{Name: "query", Value: query},
	// 	{Name: "remove", Value: true},
	// }

	for _, option := range opts {
		option.Option(command)
		// switch name := option.FindOneAndDeleteName(); name {
		// case "maxTimeMS":
		// 	command.AppendElem(
		// 		name,
		// 		int64(option.FindOneAndDeleteValue().(time.Duration)/time.Millisecond),
		// 	)
		// default:
		// 	command.AppendElem(name, option.FindOneAndDeleteValue())
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

	returned := struct{ Value *oldbson.Raw }{}

	err := runMustUsePrimary(ctx, s, ns.DB, command, &returned)
	if err != nil {
		return false, internal.WrapError(err, "failed to execute count")
	}

	if returned.Value == nil {
		return false, nil
	}

	if result == nil {
		return true, nil
	}

	return true, returned.Value.Unmarshal(result)
}
