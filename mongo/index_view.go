package mongo

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/ops"
)

// ErrInvalidIndexValue indicates that the index Keys document has a value that isn't either a number or a string.
var ErrInvalidIndexValue = errors.New("invalid index value")

// ErrNonStringIndexName indicates that the index name specified in the options is not a string.
var ErrNonStringIndexName = errors.New("index name must be a string")

// ErrMultipleIndexDrop indicates that multiple indexes would be dropped from a call to IndexView.DropOne.
var ErrMultipleIndexDrop = errors.New("multiple indexes would be dropped")

// IndexView is used to create, drop, and list indexes on a given collection.
type IndexView struct {
	coll *Collection
}

// IndexModel contains information about an index.
type IndexModel struct {
	Keys    *bson.Document
	Options *bson.Document
}

// List returns a cursor iterating over all the indexes in the collection.
func (iv IndexView) List(ctx context.Context) (Cursor, error) {
	s, err := iv.coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	return ops.ListIndexes(ctx, s, iv.coll.namespace(), ops.ListIndexesOptions{})
}

// CreateOne creates a single index in the collection specified by the model.
func (iv IndexView) CreateOne(ctx context.Context, model IndexModel) (string, error) {
	names, err := iv.CreateMany(ctx, model)
	if err != nil {
		return "", err
	}

	return names[0], nil
}

// CreateMany creates multiple indexes in the collection specified by the models. The names of the
// creates indexes are returned.
func (iv IndexView) CreateMany(ctx context.Context, models ...IndexModel) ([]string, error) {
	names := make([]string, 0, len(models))
	indexes := bson.NewArray()

	for _, model := range models {
		name, err := getOrGenerateIndexName(model)
		if err != nil {
			return nil, err
		}

		names = append(names, name)

		index := bson.NewDocument(
			bson.EC.SubDocument("key", model.Keys),
		)
		if model.Options != nil {
			err = index.Concat(model.Options)
			if err != nil {
				return nil, err
			}
		}
		index.Set(bson.EC.String("name", name))

		indexes.Append(bson.VC.Document(index))
	}

	s, err := iv.coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	err = ops.CreateIndexes(ctx, s, iv.coll.namespace(), indexes)
	if err != nil {
		return nil, err
	}

	return names, nil
}

// DropOne drops the index with the given name from the collection.
func (iv IndexView) DropOne(ctx context.Context, name string) (bson.Reader, error) {
	if name == "*" {
		return nil, ErrMultipleIndexDrop
	}

	s, err := iv.coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	return ops.DropIndexes(ctx, s, iv.coll.namespace(), name)
}

// DropAll drops all indexes in the collection.
func (iv IndexView) DropAll(ctx context.Context) (bson.Reader, error) {
	s, err := iv.coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	return ops.DropIndexes(ctx, s, iv.coll.namespace(), "*")
}

func getOrGenerateIndexName(model IndexModel) (string, error) {
	if model.Options != nil {
		nameVal, err := model.Options.Lookup("name")

		switch err {
		case bson.ErrElementNotFound:
			break
		case nil:
			if nameVal.Value().Type() != bson.TypeString {
				return "", ErrNonStringIndexName
			}

			return nameVal.Value().StringValue(), nil
		default:
			return "", err
		}
	}

	name := bytes.NewBufferString("")
	itr := model.Keys.Iterator()
	first := true

	for itr.Next() {
		if !first {
			_, err := name.WriteRune('_')
			if err != nil {
				return "", err
			}
		}

		elem := itr.Element()
		_, err := name.WriteString(elem.Key())
		if err != nil {
			return "", err
		}

		_, err = name.WriteRune('_')
		if err != nil {
			return "", err
		}

		var value string

		switch elem.Value().Type() {
		case bson.TypeInt32:
			value = fmt.Sprintf("%d", elem.Value().Int32())
		case bson.TypeInt64:
			value = fmt.Sprintf("%d", elem.Value().Int64())
		case bson.TypeString:
			value = elem.Value().StringValue()
		default:
			return "", ErrInvalidIndexValue
		}

		_, err = name.WriteString(value)
		if err != nil {
			return "", err
		}

		first = false
	}
	if err := itr.Err(); err != nil {
		return "", err
	}

	return name.String(), nil
}
