package mnet

// WireMessageCompressor is an interface used to compress wire messages. If a
// Connection supports compression it should implement this interface as well.
// The CompressWireMessage method will be called during the execution of an
// operation if the wire message is allowed to be compressed.
type WireMessageCompressor interface {
	CompressWireMessage(src, dst []byte) ([]byte, error)
}

type defaultCompressor struct{}

// Assert defaultCompressor implements WireMessageCompressor at compile-time.
var _ WireMessageCompressor = &defaultCompressor{}

func (defaultCompressor) CompressWireMessage([]byte, []byte) ([]byte, error) { return nil, nil }
