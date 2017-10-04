package ops

import (
	"context"

	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/options"
	"github.com/10gen/mongo-go-driver/yamgo/readconcern"
)

// Count counts how many documents in a collection match a given query.
func Count(ctx context.Context, s *SelectedServer, ns Namespace, readConcern *readconcern.ReadConcern,
	query interface{}, options ...options.CountOption) (int, error) {

	if err := ns.validate(); err != nil {
		return 0, err
	}

	command := bson.D{
		{Name: "count", Value: ns.Collection},
		{Name: "query", Value: query},
	}

	for _, option := range options {
		switch name := option.CountName(); name {
		case "maxTimeMS":
			command.AppendElem(
				name,
				int64(option.CountValue().(time.Duration)/time.Millisecond),
			)
		default:
			command.AppendElem(name, option.CountValue())

		}
	}

	if readConcern != nil {
		command.AppendElem("readConcern", readConcern)
	}

	result := struct{ N int }{}

	err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return 0, internal.WrapError(err, "failed to execute count")
	}

	return result.N, nil
}
