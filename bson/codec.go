package bson

import (
	"fmt"
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
	b, ok := i.(bool)
	if !ok {
		return fmt.Errorf("%T can only process bools, but got a %T", bc, i)
	}

	return vw.WriteBoolean(b)
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
