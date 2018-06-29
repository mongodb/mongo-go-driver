package bson

// A DocumentDecoder transforms a Document into a Go type.
//
// A DocumentDecoder is goroutine safe and the DecodeDocument method can be called
// from different goroutines simultaneously.
type DocumentDecoder struct{}

// NewDocumentDecoder returns a new document decoder that uses Registry r.
func NewDocumentDecoder(r *Registry) (*DocumentDecoder, error) { return nil, nil }

// DecodeDocument decodes d into val.
//
// The documentation for Unmarshal contains details about the conversion of a *Document into
// a Go value.
func (dd *DocumentDecoder) DecodeDocument(d *Document, val interface{}) error { return nil }
