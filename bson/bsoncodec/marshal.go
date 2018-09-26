package bsoncodec

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
)

const defaultDstCap = 256

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
	dst := make([]byte, 0, defaultDstCap)
	return MarshalAppendWithRegistry(r, dst, val)
}

// MarshalAppendWithRegistry will append the BSON encoding of val to dst using
// Registry r. If dst is not large enough to hold the BSON encoding of val, dst
// will be grown.
func MarshalAppendWithRegistry(r *Registry, dst []byte, val interface{}) ([]byte, error) {
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

// MarshalExtJSON returns the extended JSON encoding of val.
func MarshalExtJSON(val interface{}, canonical, escapeHTML bool) ([]byte, error) {
	return MarshalExtJSONWithRegistry(defaultRegistry, val, canonical, escapeHTML)
}

// MarshalExtJSONAppend will append the extended JSON encoding of val to dst.
// If dst is not large enough to hold the extended JSON encoding of val, dst
// will be grown.
func MarshalExtJSONAppend(dst []byte, val interface{}, canonical, escapeHTML bool) ([]byte, error) {
	return MarshalExtJSONAppendWithRegistry(defaultRegistry, dst, val, canonical, escapeHTML)
}

// MarshalExtJSONWithRegistry returns the extended JSON encoding of val using Registry r.
func MarshalExtJSONWithRegistry(r *Registry, val interface{}, canonical, escapeHTML bool) ([]byte, error) {
	dst := make([]byte, 0, defaultDstCap)
	return MarshalExtJSONAppendWithRegistry(r, dst, val, canonical, escapeHTML)
}

// MarshalExtJSONAppendWithRegistry will append the extended JSON encoding of
// val to dst using Registry r. If dst is not large enough to hold the BSON
// encoding of val, dst will be grown.
func MarshalExtJSONAppendWithRegistry(r *Registry, dst []byte, val interface{}, canonical, escapeHTML bool) ([]byte, error) {
	ejvw := ejvwPool.Get().(*extJSONValueWriter)
	defer ejvwPool.Put(ejvw)

	ejvw.reset(dst, canonical, escapeHTML)

	enc := encPool.Get().(*Encoder)
	defer encPool.Put(enc)

	err := enc.Reset(ejvw)
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

	return ejvw.buf, nil
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
