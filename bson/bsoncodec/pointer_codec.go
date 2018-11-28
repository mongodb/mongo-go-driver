package bsoncodec

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

var defaultPointerCodec = &PointerCodec{
	ecache: make(map[reflect.Type]ValueEncoderLegacy),
	dcache: make(map[reflect.Type]ValueDecoder),
}

var _ ValueEncoderLegacy = &PointerCodec{}
var _ ValueDecoder = &PointerCodec{}

// PointerCodec is the Codec used for pointers.
type PointerCodec struct {
	ecache map[reflect.Type]ValueEncoderLegacy
	dcache map[reflect.Type]ValueDecoder
	l      sync.RWMutex
}

// NewPointerCodec returns a PointerCodec that has been initialized.
func NewPointerCodec() *PointerCodec {
	return &PointerCodec{
		ecache: make(map[reflect.Type]ValueEncoderLegacy),
		dcache: make(map[reflect.Type]ValueDecoder),
	}
}

// EncodeValueLegacy handles encoding a pointer by either encoding it to BSON Null if the pointer is nil
// or looking up an encoder for the type of value the pointer points to.
func (pc *PointerCodec) EncodeValueLegacy(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	if i == nil {
		return vw.WriteNull()
	}
	val := reflect.ValueOf(i)

	if val.Type().Kind() != reflect.Ptr {
		return LegacyValueEncoderError{
			Name:     "PointerCodec.EncodeValue",
			Types:    []interface{}{reflect.Ptr},
			Received: i,
		}
	}

	if val.IsNil() {
		return vw.WriteNull()
	}

	pc.l.RLock()
	enc, ok := pc.ecache[val.Type()]
	pc.l.RUnlock()
	if ok {
		if enc == nil {
			return ErrNoEncoder{Type: val.Type()}
		}
		return enc.EncodeValueLegacy(ec, vw, val.Elem().Interface())
	}

	enc, err := ec.LookupEncoder(val.Type().Elem())
	pc.l.Lock()
	pc.ecache[val.Type()] = enc
	pc.l.Unlock()
	if err != nil {
		return err
	}

	return enc.EncodeValueLegacy(ec, vw, val.Elem().Interface())
}

// DecodeValue handles decoding a pointer either by setting the value of the pointer to nil if the
// BSON value is Null or by looking up a decoder for the type of pointer and using that.
//
// This method can only handle double pointers, e.g. **type, since parameters in Go are copies, not
// references.
func (pc *PointerCodec) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() || val.Type().Elem().Kind() != reflect.Ptr {
		return fmt.Errorf("PointerCodec.DecodeValue can only process non-nil pointers to pointers, but got (%T) %v", i, i)
	}

	if vr.Type() == bsontype.Null {
		val.Elem().Set(reflect.Zero(val.Type().Elem()))
		return vr.ReadNull()
	}

	if val.Elem().IsNil() {
		val.Elem().Set(reflect.New(val.Type().Elem().Elem()))
	}

	pc.l.RLock()
	dec, ok := pc.dcache[val.Type()]
	pc.l.RUnlock()
	if ok {
		if dec == nil {
			return ErrNoDecoder{Type: val.Type()}
		}
		return dec.DecodeValue(dc, vr, val.Elem().Interface())
	}

	dec, err := dc.LookupDecoder(val.Type().Elem())
	if _, ok := dec.(*PointerCodec); ok && err == nil {
		// There isn't a decoder registered for *type, so we need to lookup a decoder for type
		dec, err = dc.LookupDecoder(val.Type().Elem().Elem())
	}
	pc.l.Lock()
	pc.dcache[val.Type()] = dec
	pc.l.Unlock()
	if err != nil {
		return err
	}

	return dec.DecodeValue(dc, vr, val.Elem().Interface())
}
