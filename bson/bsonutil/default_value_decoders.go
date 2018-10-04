package bsonutil

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
)

// DefaultValueDecoders is a namespace type for the default ValueDecoders used
// when creating a registry.
type DefaultValueDecoders struct{}

func (dvd DefaultValueDecoders) decodeElement(dc DecodeContext, ar ArrayReader) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)
	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		var elem *bson.Element
		err = dvd.elementDecodeValue(dc, vr, "", &elem)
		if err != nil {
			return nil, err
		}
		elems = append(elems, reflect.ValueOf(elem))
	}

	return elems, nil
}
