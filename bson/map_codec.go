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

	return mc.encodeValue(ec, dw, val, nil)
}

// encodeValue handles encoding of the values of a map. The collisionFn returns
// true if the provided key exists, this is mainly used for inline maps in the
// struct codec.
func (mc *MapCodec) encodeValue(ec EncodeContext, dw DocumentWriter, val reflect.Value, collisionFn func(string) bool) error {

	var err error
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
		if collisionFn != nil && collisionFn(key.String()) {
			return fmt.Errorf("Key %s of inlined map conflicts with a struct field name", key)
		}
		vw, err := dw.WriteDocumentElement(key.String())
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
		val.Elem().Set(reflect.MakeMap(val.Elem().Type()))
	}

	mVal := val.Elem()

	dFn, err := mc.decodeFn(dc, mVal)
	if err != nil {
		return err
	}

	for {
		var elem reflect.Value
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}
		key, elem, err = dFn(dc, vr, key)
		if err != nil {
			return err
		}

		mVal.SetMapIndex(reflect.ValueOf(key), elem)
	}
	return err
}

type decodeFn func(dc DecodeContext, vr ValueReader, key string) (updatedKey string, v reflect.Value, err error)

// decodeFn returns a function that can be used to decode the values of a map.
// The mapVal parameter should be a map type, not a pointer to a map type.
//
// If error is nil, decodeFn will return a non-nil decodeFn.
func (mc *MapCodec) decodeFn(dc DecodeContext, mapVal reflect.Value) (decodeFn, error) {
	var dFn decodeFn
	switch mapVal.Type().Elem() {
	case tElement:
		// TODO(skriptble): We have to decide if we want to support this. We have
		// information loss because we can only store either the map key or the element
		// key. We could add a struct tag field that allows the user to make a decision.
		dFn = func(dc DecodeContext, vr ValueReader, key string) (string, reflect.Value, error) {
			var elem *Element
			err := defaultElementCodec.decodeValue(dc, vr, key, &elem)
			if err != nil {
				return key, reflect.Value{}, err
			}
			return key, reflect.ValueOf(elem), nil
		}
	default:
		eType := mapVal.Type().Elem()
		codec, err := dc.Lookup(eType)
		if err != nil {
			return nil, err
		}

		dFn = func(dc DecodeContext, vr ValueReader, key string) (string, reflect.Value, error) {
			ptr := reflect.New(eType)

			err = codec.DecodeValue(dc, vr, ptr.Interface())
			if err != nil {
				return key, reflect.Value{}, err
			}
			return key, ptr.Elem(), nil
		}
	}

	return dFn, nil
}
