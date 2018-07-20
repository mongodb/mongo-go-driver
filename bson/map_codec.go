package bson

import (
	"fmt"
	"reflect"
)

var defaultMapCodec = &MapCodec{}

// MapCodec is the Codec used for map values.
type MapCodec struct{}

var _ Codec = &MapCodec{}

// EncodeValue implements the Codec interface.
func (mc *MapCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Map || val.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%T can only encode maps with string keys", mc)
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	codec, err := ec.Lookup(val.Type().Elem())
	if err != nil {
		return err
	}

	keys := val.MapKeys()
	for _, key := range keys {
		vw, err := dw.WriteDocumentElement(val.String())
		if err != nil {
			return err
		}

		err = codec.EncodeValue(ec, vw, val.MapIndex(key).Interface())
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// DecodeValue implements the Codec interface.
func (mc *MapCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Map || val.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%T can only encode maps with string keys", mc)
	}

	if !val.CanSet() {
		return fmt.Errorf("cannot set value of %v", val)
	}

	eTypeIsPointer := false
	eType := val.Type().Elem()
	if eType.Kind() == reflect.Ptr {
		eTypeIsPointer = true
		eType = eType.Elem()
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	codec, err := dc.Lookup(eType)
	if err != nil {
		return err
	}

	// TODO: Don't make a new map here, use the one provided.
	rmap := reflect.MakeMap(val.Type())

	for {
		key, vr, err := dr.ReadElement()
		if err == EOD {
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

		var mvval reflect.Value
		if eTypeIsPointer {
			mvval = ptr
		} else {
			mvval = ptr.Elem()
		}

		rmap.SetMapIndex(reflect.ValueOf(key), mvval)
	}

	val.Set(rmap)
	return nil
}
