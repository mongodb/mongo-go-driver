package bson

import (
	"fmt"
	"reflect"
)

var defaultSliceCodec = &SliceCodec{}

type SliceCodec struct{}

func (sc *SliceCodec) EncodeValue(*Registry, ValueWriter, interface{}) error {
	panic("not implemented")
}

func (sc *SliceCodec) DecodeValue(r *Registry, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() && !val.CanSet() {
			return fmt.Errorf("cannot set value of field %v and it is nil", val)
		}
		return sc.decodeSlice(r, vr, val)
	case reflect.Array:
		return sc.decodeArray(r, vr, val)
	default:
		return fmt.Errorf("%T can only processes arrays and slices", sc)
	}
}

func (sc *SliceCodec) decodeSlice(r *Registry, vr ValueReader, val reflect.Value) error {
	elems := make([]reflect.Value, 0)
	eType := val.Type().Elem()
	if eType.Kind() == reflect.Ptr {
		eType = eType.Elem()
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	for {
		vr, err := ar.ReadValue()
		if err == EOA {
			break
		}
		if err != nil {
			return err
		}

		ptr := reflect.New(eType)

		codec, err := r.Lookup(ptr.Type())
		if err != nil {
			return fmt.Errorf("unable to find codec for type %v for field '%s': %v", ptr.Type(), val, err)
		}

		err = codec.DecodeValue(r, vr, ptr.Interface())
		if err != nil {
			return err
		}
		elems = append(elems, ptr.Elem())
	}

	slc := reflect.MakeSlice(val.Type(), len(elems), len(elems))

	for idx, elem := range elems {
		slc.Index(idx).Set(elem)
	}

	val.Set(slc)

	return nil
}

func (sc *SliceCodec) decodeArray(r *Registry, vr ValueReader, val reflect.Value) error {
	return nil
}
