package bsoncodec

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
		return new(Encoder)
	},
}

// An Encoder writes a serialization format to an output stream.
type Encoder struct {
	r  *Registry
	vw ValueWriter
}

// NewEncoder returns a new encoder that uses Registry r to write to w.
func NewEncoder(r *Registry, vw ValueWriter) (*Encoder, error) {
	if r == nil {
		return nil, errors.New("cannot create a new Encoder with a nil Registry")
	}
	if vw == nil {
		return nil, errors.New("cannot create a new Encoder with a nil ValueWriter")
	}

	return &Encoder{
		r:  r,
		vw: vw,
	}, nil
}

// Encode writes the BSON encoding of val to the stream.
//
// The documentation for Marshal contains details about the conversion of Go
// values to BSON.
func (e *Encoder) Encode(val interface{}) error {
	if marshaler, ok := val.(Marshaler); ok {
		// TODO(skriptble): Should we have a MarshalAppender interface so that we can have []byte reuse?
		buf, err := marshaler.MarshalBSON()
		if err != nil {
			return err
		}
		return Copier{r: e.r}.CopyDocumentFromBytes(e.vw, buf)
	}

	encoder, err := e.r.LookupEncoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return encoder.EncodeValue(EncodeContext{Registry: e.r}, e.vw, val)
}

// Reset will reset the state of the encoder, using the same *Registry used in
// the original construction but using vw.
func (e *Encoder) Reset(vw ValueWriter) error {
	e.vw = vw
	return nil
}

// SetRegistry replaces the current registry of the encoder with r.
func (e *Encoder) SetRegistry(r *Registry) error {
	e.r = r
	return nil
}
