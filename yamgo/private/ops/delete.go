package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// Delete executes an delete command with a given set of delete documents and options.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Delete(ctx context.Context, s *SelectedServer, ns Namespace, deleteDocs []bson.D,
	result interface{}, options ...options.DeleteOption) error {

	if err := ns.validate(); err != nil {
		return err
	}

	command := bson.D{
		{Name: "delete", Value: ns.Collection},
	}

	for _, option := range options {
		switch name := option.DeleteName(); name {
		case "collation":
			for i := range deleteDocs {
				deleteDocs[i].AppendElem("collation", option.DeleteValue())
			}
		default:
			command.AppendElem(option.DeleteName(), option.DeleteValue())
		}
	}

	command.AppendElem("deletes", deleteDocs)

	// TODO GODRIVER-27: write concern

	err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute delete")
	}

	return nil
}
