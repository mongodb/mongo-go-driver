package dispatch

import (
	"errors"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
)

// ErrCollation is caused if a collation is given for an invalid server version.
var ErrCollation = errors.New("collation cannot be set for server versions < 3.4")

// ErrArrayFilters is caused if array filters are given for an invalid server version.
var ErrArrayFilters = errors.New("array filters cannot be set for server versions < 3.6")

func interfaceToDocument(val interface{}, registry *bsoncodec.Registry) (*bson.Document, error) {
	if val == nil {
		return bson.NewDocument(), nil
	}

	if registry == nil {
		registry = bson.DefaultRegistry
	}

	if bs, ok := val.([]byte); ok {
		// Slight optimization so we'll just use MarshalBSON and not go through the codec machinery.
		val = bson.Raw(bs)
	}

	// TODO(skriptble): Use a pool of these instead.
	buf := make([]byte, 0, 256)
	b, err := bson.MarshalAppendWithRegistry(registry, buf, val)
	if err != nil {
		return nil, err
	}
	return bson.ReadDocument(b)
}

func interfaceToElement(key string, i interface{}, registry *bsoncodec.Registry) (*bson.Element, error) {
	switch conv := i.(type) {
	case string:
		return bson.EC.String(key, conv), nil
	case *bson.Document:
		return bson.EC.SubDocument(key, conv), nil
	default:
		doc, err := interfaceToDocument(i, registry)
		if err != nil {
			return nil, err
		}

		return bson.EC.SubDocument(key, doc), nil
	}
}
