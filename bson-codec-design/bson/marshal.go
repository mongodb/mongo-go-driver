package bson

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
	dst := make([]byte, 0, 1024) // TODO: make the default cap a constant
	return MarshalAppendWithRegistry(r, dst, val)
}

// MarshalAppendWithRegistry will append the BSON encoding of val to dst using
// Registry r. If dst is not large enough to hold the BSON encoding of val, dst
// will be grown.
func MarshalAppendWithRegistry(r *Registry, dst []byte, val interface{}) ([]byte, error) {
	w := writer(dst)
	vw := newValueWriter(&w)

	enc := encPool.Get().(*Encoder)
	defer encPool.Put(enc)

	enc.Reset(vw)
	enc.SetRegistry(r)

	err := enc.Encode(val)
	if err != nil {
		return nil, err
	}

	return []byte(w), nil
}

// MarshalDocument returns val encoded as a *Document.
//
// MarshalDocument will use the default registry created by NewRegistry to recursively
// marshal val into a *Document. MarshalDocument will inspect struct tags and alter the
// marshaling process accordingly.
func MarshalDocument(val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentAppend will append val encoded to dst. If dst is nil, a new *Document will be
// allocated and the encoding of val will be appended to that.
func MarshalDocumentAppend(dst *Document, val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentWithRegistry returns val encoded as a *Document using r.
func MarshalDocumentWithRegistry(r *Registry, val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentAppendWithRegistry will append val encoded to dst using r. If dst is nil, a new
// *Document will be allocated and the encoding of val will be appended to that.
func MarshalDocumentAppendWithRegistry(r *Registry, dst *Document, val interface{}) (*Document, error) {
	return nil, nil
}
