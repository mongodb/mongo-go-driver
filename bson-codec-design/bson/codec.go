package bson

import (
	"fmt"
	"reflect"
)

type Codec interface {
	EncodeValue(*Registry, ValueWriter, interface{}) error
	DecodeValue(*Registry, ValueReader, interface{}) error
}

type CodecZeroer interface {
	Codec
	IsZero(interface{}) bool
}

type BooleanCodec struct{}

var _ Codec = &BooleanCodec{}

func (bc *BooleanCodec) EncodeValue(r *Registry, vw ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Bool {
		return fmt.Errorf("%T can only process structs, but got a %T", bc, val)
	}

	return vw.WriteBoolean(val.Bool())
}

func (bc *BooleanCodec) DecodeValue(*Registry, ValueReader, interface{}) error {
	panic("not implemented")
}
