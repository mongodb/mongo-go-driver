package bson

// An Encoder writes a serialization format to an output stream.
type Encoder struct{}

// NewEncoder returns a new encoder that uses Registry r to write to w.
func NewEncoder(r *Registry, vw ValueWriter) (*Encoder, error) { return nil, nil }

// Encode writes the BSON encoding of val to the stream.
//
// The documentation for Marshal contains details about the conversion of Go
// values to BSON.
func (e *Encoder) Encode(val interface{}) error { return nil }

// Reset will reset the state of the encoder, using the same *Registry used in
// the original construction but using vw.
func (e *Encoder) Reset(vw ValueWriter) error { return nil }

// SetRegistry replaces the current registry of the encoder with r.
func (e *Encoder) SetRegistry(r *Registry) error { return nil }
