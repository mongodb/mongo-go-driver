package bson

// Marshal returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshal(val interface{}) ([]byte, error) { return nil, nil }

// MarshalAppend will append the BSON encoding of val to src. If src is not
// large enough to hold the BSON encoding of val, src will be grown.
func MarshalAppend(src []byte, val interface{}) ([]byte, error) { return nil, nil }

// MarshalWithRegistry returns the BSON encoding of val using Registry r.
func MarshalWithRegistry(r Registry, val interface{}) ([]byte, error) { return nil, nil }

// MarshalAppendWithRegistry will append the BSON encoding of val to src using
// Registry r. If src is not large enough to hold the BSON encoding of val, src
// will be grown.
func MarshalAppendWithRegistry(r Registry, src []byte, val interface{}) ([]byte, error) {
	return nil, nil
}
