package ops

import (
	"context"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/10gen/mongo-go-driver/mongo/writeconcern"
)

// FindOneAndDelete modifies and returns a single document.
func FindOneAndDelete(ctx context.Context, s *SelectedServer, ns Namespace,
	writeConcern *writeconcern.WriteConcern, query interface{}, result interface{},
	opts ...options.FindOneAndDeleteOption) (bool, error) {

	if err := ns.validate(); err != nil {
		return false, err
	}

	command := bson.D{
		{Name: "findAndModify", Value: ns.Collection},
		{Name: "query", Value: query},
		{Name: "remove", Value: true},
	}

	for _, option := range opts {
		switch name := option.FindOneAndDeleteName(); name {
		case "maxTimeMS":
			command.AppendElem(
				name,
				int64(option.FindOneAndDeleteValue().(time.Duration)/time.Millisecond),
			)
		default:
			command.AppendElem(name, option.FindOneAndDeleteValue())
		}
	}

	if writeConcern != nil {
		command.AppendElem("writeConcern", writeConcern)
	}

	returned := struct{ Value *bson.Raw }{}

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
