package bsoncodec

import (
	"errors"
	"reflect"
)

// A Decoder reads and decodes BSON documents from a stream.
type Decoder struct {
	r  *Registry
	vr ValueReader
}

// NewDecoder returns a new decoder that uses Registry reg to read from r.
func NewDecoder(r *Registry, vr ValueReader) (*Decoder, error) {
	if r == nil {
		return nil, errors.New("cannot create a new Decoder with a nil Registry")
	}
	if vr == nil {
		return nil, errors.New("cannot create a new Decoder with a nil ValueReader")
	}

	return &Decoder{
		r:  r,
		vr: vr,
	}, nil
}

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// The documentation for Unmarshal contains details about of BSON into a Go
// value.
func (d *Decoder) Decode(val interface{}) error {
	codec, err := d.r.Lookup(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return codec.DecodeValue(DecodeContext{Registry: d.r}, d.vr, val)
}

// Reset will reset the state of the decoder, using the same *Registry used in
// the original construction but using r for reading.
func (d *Decoder) Reset(vr ValueReader) error {
	d.vr = vr
	return nil
}

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoder) SetRegistry(r *Registry) error {
	d.r = r
	return nil
}
