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

func (bc *BooleanCodec) DecodeValue(r *Registry, vr ValueReader, i interface{}) error {
	target, ok := i.(*bool)
	if !ok {
		return fmt.Errorf("%T can only be used to decode *bool", bc)
	}

	if vr.Type() != TypeBoolean {
		return fmt.Errorf("cannot decode %v into a boolean", vr.Type())
	}

	var err error
	*target, err = vr.ReadBoolean()
	return err
}
