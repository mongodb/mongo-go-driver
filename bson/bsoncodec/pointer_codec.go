package bsoncodec

import (
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

// DecodeValue handles decoding a pointer by looking up a decoder for the type it points to and
// using that to decode. If the BSON value is Null, this method will set the pointer to nil.
func (pc *PointerCodec) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.Ptr {
		return ValueDecoderError{Name: "PointerCodec.DecodeValue", Kinds: []reflect.Kind{reflect.Ptr}, Received: val}
	}

	if vr.Type() == bsontype.Null {
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	}

	if val.IsNil() {
		val.Set(reflect.New(val.Type().Elem()))
	}

	pc.l.RLock()
	dec, ok := pc.dcache[val.Type()]
	pc.l.RUnlock()
	if ok {
		if dec == nil {
			return ErrNoDecoder{Type: val.Type()}
		}
		return dec.DecodeValue(dc, vr, val.Elem())
	}

	dec, err := dc.LookupDecoder(val.Type().Elem())
	pc.l.Lock()
	pc.dcache[val.Type()] = dec
	pc.l.Unlock()
	if err != nil {
		return err
	}

	return dec.DecodeValue(dc, vr, val.Elem())
}
