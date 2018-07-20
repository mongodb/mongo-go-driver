package bson

import "sync"

// This pool is used to keep the allocations of Decoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Encoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var decPool = sync.Pool{
	New: func() interface{} {
		return new(Decoderv2)
	},
}

// Unmarshal parses the BSON-encoded data and stores the result in the value
// pointed to by val. If val is nil or not a pointer, Unmarshal returns
// InvalidUnmarshalError.
func Unmarshalv2(data []byte, val interface{}) error {
	return UnmarshalWithRegistry(defaultRegistry, data, val)
}

// UnmarshalWithRegistry parses the BSON-encoded data using Registry r and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalWithRegistry(r *Registry, data []byte, val interface{}) error {
	vr := newValueReader(data)

	dec := decPool.Get().(*Decoderv2)
	defer decPool.Put(dec)

	dec.Reset(vr)
	dec.SetRegistry(r)

	return dec.Decode(val)
}

// UnmarshalDocument parses the *Document and stores the result in the value pointed to by val. If
// val is nil or not a pointer, UnmarshalDocument returns InvalidUnmarshalError.
func UnmarshalDocumentv2(d *Document, val interface{}) error {
	return UnmarshalDocumentWithRegistry(defaultRegistry, d, val)
}

// UnmarshalDocumentWithRegistry behaves the same as UnmarshalDocument but uses r as the *Registry.
func UnmarshalDocumentWithRegistry(r *Registry, d *Document, val interface{}) error {
	dvr, err := NewDocumentValueReader(d)
	if err != nil {
		return err
	}

	dec := decPool.Get().(*Decoderv2)
	defer decPool.Put(dec)

	dec.Reset(dvr)
	dec.SetRegistry(r)

	return dec.Decode(val)
}
