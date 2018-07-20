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
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("%T can only be used to decode non-nil pointers to map values, got %T", mc, i)
	}

	if val.Elem().Kind() != reflect.Map || val.Elem().Type().Key().Kind() != reflect.String || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only decode settable maps with string keys", mc)
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	if val.Elem().IsNil() {
		val.Elem().Set(reflect.MakeMap(val.Type()))
	}

	switch val.Type().Elem() {
	case tElement:
		err = mc.decodeElement(dc, dr, val)
	default:
		err = mc.decodeDefault(dc, dr, val)
	}

	return err
}

func (mc *MapCodec) decodeElement(dc DecodeContext, dr DocumentReader, val reflect.Value) error {
	for {
		key, vr, err := dr.ReadElement()
		if err == EOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *Element
		err = defaultElementCodec.decodeValue(dc, vr, key, &elem)
		if err != nil {
			return err
		}

		val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem))
	}

	return nil
}

func (mc *MapCodec) decodeDefault(dc DecodeContext, dr DocumentReader, val reflect.Value) error {
	eType := val.Type().Elem()
	codec, err := dc.Lookup(eType)
	if err != nil {
		return err
	}

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

		val.SetMapIndex(reflect.ValueOf(key), ptr.Elem())
	}

	return nil
}
