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
	case reflect.Array:
	case reflect.Slice:
		if val.IsNil() { // When nil, special case to null
			return vw.WriteNull()
		}
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
	var codec Codec
	switch val.Type().Elem() {
	case tElement:
		codec = defaultElementCodec
	default:
		codec, err = ec.Lookup(val.Type().Elem())
		if err != nil {
			return err
		}
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
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("%T can only be used to decode non-nil pointers to slice or array values, got %T", sc, i)
	}

	switch val.Elem().Kind() {
	case reflect.Slice, reflect.Array:
		if !val.Elem().CanSet() {
			return fmt.Errorf("%T can only decode settable slice and array values", sc)
		}
	default:
		return fmt.Errorf("%T can only decode settable slice and array values, got %T", sc, i)
	}

	switch vr.Type() {
	case TypeArray:
	case TypeNull:
		if val.Elem().Kind() != reflect.Slice {
			return fmt.Errorf("cannot decode %v into an array", vr.Type())
		}
		null := reflect.Zero(val.Elem().Type())
		val.Elem().Set(null)
		return vr.ReadNull()
	default:
		return fmt.Errorf("cannot decode %v into a slice", vr.Type())
	}

	eType := val.Type().Elem().Elem()

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	var elems []reflect.Value
	switch eType {
	case tElement:
		elems, err = sc.decodeElement(dc, ar)
	default:
		elems, err = sc.decodeDefault(dc, ar, eType)
	}

	if err != nil {
		return err
	}

	switch val.Elem().Kind() {
	case reflect.Slice:
		slc := reflect.MakeSlice(val.Elem().Type(), len(elems), len(elems))

		for idx, elem := range elems {
			slc.Index(idx).Set(elem)
		}

		val.Elem().Set(slc)
	case reflect.Array:
		if len(elems) > val.Elem().Len() {
			return fmt.Errorf("more elements returned in array than can fit inside %s", val.Elem().Type())
		}

		for idx, elem := range elems {
			val.Elem().Index(idx).Set(elem)
		}
	}

	return nil
}

func (sc *SliceCodec) decodeElement(dc DecodeContext, ar ArrayReader) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)
	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		var elem *Element
		err = defaultElementCodec.decodeValue(dc, vr, "", &elem)
		if err != nil {
			return nil, err
		}
		elems = append(elems, reflect.ValueOf(elem))
	}

	return elems, nil
}

func (sc *SliceCodec) decodeDefault(dc DecodeContext, ar ArrayReader, eType reflect.Type) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	codec, err := dc.Lookup(eType)
	if err != nil {
		return nil, err
	}

	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		ptr := reflect.New(eType)

		err = codec.DecodeValue(dc, vr, ptr.Interface())
		if err != nil {
			return nil, err
		}
		elems = append(elems, ptr.Elem())
	}

	return elems, nil
}
