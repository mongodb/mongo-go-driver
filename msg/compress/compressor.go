package compress

import "io"

// Compressor handles compressing and decompressing bytes.
type Compressor interface {
	// ID is the identifier of the compressor
	ID() uint8
	// Name is the name of the compressor
	Name() string
	// Compress compresses the bytes and writes them to the writer.
	Compress([]byte, io.Writer) error
	// Decompress decrompresses the reader and writes them to the the bytes.
	Decompress(io.Reader, []byte) error
}
