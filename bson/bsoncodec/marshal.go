package bsoncodec

import "github.com/mongodb/mongo-go-driver/bson"

// Marshalv2 returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshalv2(val interface{}) ([]byte, error) {
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

	enc := encPool.Get().(*Encoderv2)
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

// MarshalDocument returns val encoded as a *Document.
//
// MarshalDocument will use the default registry created by NewRegistry to recursively
// marshal val into a *Document. MarshalDocument will inspect struct tags and alter the
// marshaling process accordingly.
func MarshalDocument(val interface{}) (*bson.Document, error) {
	return MarshalDocumentAppend(bson.NewDocument(), val)
}

// MarshalDocumentAppend will append val encoded to dst. If dst is nil, a new *Document will be
// allocated and the encoding of val will be appended to that.
func MarshalDocumentAppend(dst *bson.Document, val interface{}) (*bson.Document, error) {
	return MarshalDocumentAppendWithRegistry(defaultRegistry, dst, val)
}

// MarshalDocumentWithRegistry returns val encoded as a *Document using r.
func MarshalDocumentWithRegistry(r *Registry, val interface{}) (*bson.Document, error) {
	return MarshalDocumentAppendWithRegistry(r, bson.NewDocument(), val)
}

// MarshalDocumentAppendWithRegistry will append val encoded to dst using r. If dst is nil, a new
// *Document will be allocated and the encoding of val will be appended to that.
func MarshalDocumentAppendWithRegistry(r *Registry, dst *bson.Document, val interface{}) (*bson.Document, error) {
	d := dst
	if d == nil {
		d = bson.NewDocument()
	}
	dvw := newDocumentValueWriter(d)

	enc := encPool.Get().(*Encoderv2)
	defer encPool.Put(enc)

	err := enc.Reset(dvw)
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

	return d, nil
}
