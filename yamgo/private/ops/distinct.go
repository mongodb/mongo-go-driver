package ops

import (
	"context"

	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// Distinct returns the distinct values for a specified field across a single collection.
func Distinct(ctx context.Context, s *SelectedServer, ns Namespace, field string, query interface{},
	options ...options.DistinctOption) ([]interface{}, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.D{
		{Name: "distinct", Value: ns.Collection},
		{Name: "key", Value: field},
	}

	if query != nil {
		command.AppendElem("query", query)
	}

	for _, option := range options {
		switch name := option.DistinctName(); name {
		case "maxTimeMS":
			command.AppendElem(
				name,
				int64(option.DistinctValue().(time.Duration)/time.Millisecond),
			)
		default:
			command.AppendElem(name, option.DistinctValue())

		}
	}

	// TODO GODRIVER-27: read concern

	result := struct{ Values []interface{} }{}

	err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute count")
	}

	return result.Values, nil
}
