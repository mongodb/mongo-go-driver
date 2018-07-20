package bson

import (
	"fmt"
	"reflect"
)

var defaultSliceCodec = &SliceCodec{}

// SliceCodec is the Codec used for slice and array values.
type SliceCodec struct{}

var _ Codec = &SliceCodec{}

// EncodeValue implements the Codec interface.
func (sc *SliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return fmt.Errorf("%T can only encode arrays and slices", sc)
	}

	length := val.Len()

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	// We do this outside of the loop because an array or a slice can only have
	// one element type. If it's the empty interface, we'll use the empty
	// interface codec.
	codec, err := ec.Lookup(val.Type().Elem())
	if err != nil {
		return err
	}
	for idx := 0; idx < length; idx++ {
		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = codec.EncodeValue(ec, vw, val.Index(idx).Interface())
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

// DecodeValue implements the Codec interface.
func (sc *SliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("%T cannot decode into a nil pointer", sc)
		}
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if !val.CanSet() {
			return fmt.Errorf("cannot set value of %s", val.Type())
		}
	default:
		return fmt.Errorf("%T can only decode arrays and slices", sc)
	}

	elems := make([]reflect.Value, 0)
	eType := val.Type().Elem()
	if eType.Kind() == reflect.Ptr {
		eType = eType.Elem()
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	codec, err := dc.Lookup(eType)
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

		err = codec.DecodeValue(dc, vr, ptr.Interface())
		if err != nil {
			return err
		}
		elems = append(elems, ptr.Elem())
	}

	switch val.Kind() {
	case reflect.Slice:
		slc := reflect.MakeSlice(val.Type(), len(elems), len(elems))

		for idx, elem := range elems {
			slc.Index(idx).Set(elem)
		}

		val.Set(slc)
	case reflect.Array:
		if len(elems) > val.Len() {
			return fmt.Errorf("more elements returned in array than can fit inside %s", val.Type())
		}

		for idx, elem := range elems {
			val.Index(idx).Set(elem)
		}
	}

	return nil
}
