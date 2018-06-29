package bson

import "io"

// A Decoder reads and decodes BSON documents from a stream.
type Decoder struct{}

// NewDecoder returns a new decoder that uses Registry reg to read serialization type st from r.
func NewDecoder(reg *Registry, r io.Reader, st SerializationType) (*Decoder, error) { return nil, nil }

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// The documentation for Unmarshal contains details about of BSON into a Go
// value.
func (d *Decoder) Decode(val interface{}) error { return nil }

// Reset will reset the state of the decoder, using the same *Registry used in
// the original construction but using r for reading with serialization type st.
func (d *Decoder) Reset(r io.Reader, st SerializationType) error { return nil }

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoder) SetRegistry(r *Registry) error { return nil }
