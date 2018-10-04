package bsonutil

import (
	"github.com/mongodb/mongo-go-driver/bson"
)

// DefaultValueEncoders is a namespace type for the default ValueEncoders used
// when creating a registry.
type DefaultValueEncoders struct{}

// elementEncodeValue is used internally to encode to values
func (dve DefaultValueEncoders) elementEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	elem, ok := i.(*bson.Element)
	if !ok {
		return ValueEncoderError{
			Name:     "elementEncodeValue",
			Types:    []interface{}{(*bson.Element)(nil)},
			Received: i,
		}
	}

	if _, err := elem.Validate(); err != nil {
		return err
	}

	return dve.encodeValue(ectx, vw, elem.Value())
}
