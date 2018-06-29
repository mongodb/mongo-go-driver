package bson

// A DocumentEncoder transforms Go types into a Document.
//
// A DocumentEncoder is goroutine safe and the EncodeDocument method can be called
// from different goroutines simultaneously.
type DocumentEncoder struct{}

// NewDocumentEncoder returns a new document encoder that uses Registry r.
func NewDocumentEncoder(r *Registry) (*DocumentEncoder, error) { return nil, nil }

// EncodeDocument encodes val into a *Document.
//
// The documentation for Marshal contains details about the conversion of Go values to
// *Document.
func (de *DocumentEncoder) EncodeDocument(val interface{}) (*Document, error) { return nil, nil }
