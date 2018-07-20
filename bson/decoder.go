package bson

import (
	"errors"
	"reflect"
	"sync"
)

// This pool is used to keep the allocations of Decoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Encoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var decPool = sync.Pool{
	New: func() interface{} {
		return new(Decoderv2)
	},
}

// A Decoderv2 reads and decodes BSON documents from a stream.
type Decoderv2 struct {
	r  *Registry
	vr ValueReader
}

// NewDecoderv2 returns a new decoder that uses Registry reg to read from r.
func NewDecoderv2(r *Registry, vr ValueReader) (*Decoderv2, error) {
	if r == nil {
		return nil, errors.New("cannot create a new Decoder with a nil Registry")
	}
	if vr == nil {
		return nil, errors.New("cannot create a new Decoder with a nil ValueReader")
	}

	return &Decoderv2{
		r:  r,
		vr: vr,
	}, nil
}

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// The documentation for Unmarshal contains details about of BSON into a Go
// value.
func (d *Decoderv2) Decode(val interface{}) error {
	// TODO: Add checking to see if val is an allowable type
	codec, err := d.r.Lookup(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return codec.DecodeValue(d.r, d.vr, val)
}

// Reset will reset the state of the decoder, using the same *Registry used in
// the original construction but using r for reading.
func (d *Decoderv2) Reset(vr ValueReader) error {
	d.vr = vr
	return nil
}

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoderv2) SetRegistry(r *Registry) error {
	d.r = r
	return nil
}
