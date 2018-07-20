package bson

import (
	"errors"
	"reflect"
	"sync"
)

// This pool is used to keep the allocations of Encoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Encoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var encPool = sync.Pool{
	New: func() interface{} {
		return new(Encoderv2)
	},
}

// An Encoderv2 writes a serialization format to an output stream.
type Encoderv2 struct {
	r  *Registry
	vw ValueWriter
}

// NewEncoderv2 returns a new encoder that uses Registry r to write to w.
func NewEncoderv2(r *Registry, vw ValueWriter) (*Encoderv2, error) {
	if r == nil {
		return nil, errors.New("cannot create a new Encoder with a nil Registry")
	}
	if vw == nil {
		return nil, errors.New("cannot create a new Encoder with a nil ValueWriter")
	}

	return &Encoderv2{
		r:  r,
		vw: vw,
	}, nil
}

// Encode writes the BSON encoding of val to the stream.
//
// The documentation for Marshal contains details about the conversion of Go
// values to BSON.
func (e *Encoderv2) Encode(val interface{}) error {
	// TODO: Add checking to see if val is an allowable type
	codec, err := e.r.Lookup(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return codec.EncodeValue(EncodeContext{Registry: e.r}, e.vw, val)
}

// Reset will reset the state of the encoder, using the same *Registry used in
// the original construction but using vw.
func (e *Encoderv2) Reset(vw ValueWriter) error {
	e.vw = vw
	return nil
}

// SetRegistry replaces the current registry of the encoder with r.
func (e *Encoderv2) SetRegistry(r *Registry) error {
	e.r = r
	return nil
}
