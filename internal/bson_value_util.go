package internal

import (
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type ErrNilValue struct{}

func (e ErrNilValue) Error() string {
	return "value is nil"
}

// ErrMapForOrderedArgument is returned when a map with multiple keys is passed
// to a CRUD method for an ordered parameter
type ErrMapForOrderedArgument struct {
	ParamName string
}

// Error implements the error interface.
func (e ErrMapForOrderedArgument) Error() string {
	return fmt.Sprintf("multi-key map passed in for ordered parameter %v",
		e.ParamName)
}

// ErrMarshal is returned when attempting to transform a value into a document
// results in an error.
type ErrMarshal struct {
	Value interface{}
	Err   error
}

// Error implements the error interface.
func (e ErrMarshal) Error() string {
	return fmt.Sprintf("cannot transform type %s to a BSON Document: %v",
		reflect.TypeOf(e.Value), e.Err)
}

// NewBSONValue will attempt to convert a value from an "any" type to a
// bsoncore.Value.
func NewBSONValue(registry *bsoncodec.Registry, val interface{},
	mapAllowed bool, paramName string) (bsoncore.Value, error) {
	if val == nil {
		return bsoncore.Value{}, ErrNilValue{}
	}

	// If the value is already a bsoncore.Value, then do nothing.
	if value, ok := val.(bsoncore.Value); ok {
		return value, nil
	}

	if !mapAllowed {
		refValue := reflect.ValueOf(val)
		if refValue.Kind() == reflect.Map && refValue.Len() > 1 {
			return bsoncore.Value{}, ErrMapForOrderedArgument{paramName}
		}
	}

	if registry == nil {
		registry = bson.DefaultRegistry
	}

	buf := make([]byte, 0, 256)
	bsonType, bsonValue, err := bson.MarshalValueAppendWithRegistry(registry, buf[:0], val)
	if err != nil {
		return bsoncore.Value{}, ErrMarshal{Value: val, Err: err}
	}

	return bsoncore.Value{Type: bsonType, Data: bsonValue}, nil
}
