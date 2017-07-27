package compress

import "io"

// NewNoopCompressor creates a new Noop compressor.
func NewNoopCompressor() Compressor {
	return &noopCompressor{}
}

type noopCompressor struct{}

func (c *noopCompressor) ID() uint8 {
	return 0
}

func (c *noopCompressor) Name() string {
	return "noop"
}

func (c *noopCompressor) Compress(in []byte, w io.Writer) error {
	target := len(in)
	for {
		n, err := w.Write(in)
		if err != nil {
			return err
		}
		target -= n
		if n <= 0 {
			break
		}
	}

	return nil
}

func (c *noopCompressor) Decompress(r io.Reader, out []byte) error {
	_, err := io.ReadFull(r, out)
	return err
}
