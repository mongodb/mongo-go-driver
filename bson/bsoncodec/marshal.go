package bsoncodec

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Marshaler is an interface implemented by types that can marshal themselves
// into a BSON document represented as bytes. The bytes returned must be a valid
// BSON document if the error is nil.
type Marshaler interface {
	MarshalBSON() ([]byte, error)
}

// ValueMarshaler is an interface implemented by types that can marshal
// themselves into a BSON value as bytes. The type must be the valid type for
// the bytes returned. The bytes and byte type together must be valid if the
// error is nil.
type ValueMarshaler interface {
	MarshalBSONValue() (bson.Type, []byte, error)
}

// Marshal returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshal(val interface{}) ([]byte, error) {
	return MarshalWithRegistry(defaultRegistry, val)
}

// MarshalAppend will append the BSON encoding of val to dst. If dst is not
// large enough to hold the BSON encoding of val, dst will be grown.
func MarshalAppend(dst []byte, val interface{}) ([]byte, error) {
	return MarshalAppendWithRegistry(defaultRegistry, dst, val)
}

// MarshalWithRegistry returns the BSON encoding of val using Registry r.
func MarshalWithRegistry(r *Registry, val interface{}) ([]byte, error) {
	dst := make([]byte, 0, 256) // TODO: make the default cap a constant
	return MarshalAppendWithRegistry(r, dst, val)
}

// MarshalAppendWithRegistry will append the BSON encoding of val to dst using
// Registry r. If dst is not large enough to hold the BSON encoding of val, dst
// will be grown.
func MarshalAppendWithRegistry(r *Registry, dst []byte, val interface{}) ([]byte, error) {
	// w := writer(dst)
	// vw := newValueWriter(&w)
	vw := vwPool.Get().(*valueWriter)
	defer vwPool.Put(vw)

	vw.reset(dst)

	enc := encPool.Get().(*Encoder)
	defer encPool.Put(enc)

	err := enc.Reset(vw)
	if err != nil {
		return nil, err
	}
	err = enc.SetRegistry(r)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(val)
	if err != nil {
		return nil, err
	}

	return vw.buf, nil
}

func marshalElement(r *Registry, dst []byte, key string, val interface{}) ([]byte, error) {
	vw := vwPool.Get().(*valueWriter)
	defer vwPool.Put(vw)

	vw.reset(dst)
	_, err := vw.WriteDocumentElement(key)
	if err != nil {
		return dst, err
	}
	t := reflect.TypeOf(val)
	enc, err := r.LookupEncoder(t)
	if err != nil {
		return dst, err
	}
	err = enc.EncodeValue(EncodeContext{Registry: r}, vw, val)
	return dst, err
}
