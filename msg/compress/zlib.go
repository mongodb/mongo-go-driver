package compress

import (
	"compress/zlib"
	"io"

	"github.com/10gen/mongo-go-driver/internal"
)

// NewZLibCompressor creates a new compressor using the zlib format.
func NewZLibCompressor() Compressor {
	return &zlibCompressor{-1}
}

// NewZLibCompressorWithLevel creates a new compressor using the zlib
// format at the specified level.
func NewZLibCompressorWithLevel(level int) Compressor {
	return &zlibCompressor{level}
}

type zlibCompressor struct {
	level int
}

func (c *zlibCompressor) ID() uint8 {
	return 2
}

func (c *zlibCompressor) Name() string {
	return "zlib"
}

func (c *zlibCompressor) Compress(in []byte, w io.Writer) error {
	var zlibWriter io.WriteCloser
	if c.level < 0 {
		zlibWriter = zlib.NewWriter(w)
	} else {
		var err error
		zlibWriter, err = zlib.NewWriterLevel(w, c.level)
		if err != nil {
			return err
		}
	}
	_, err := zlibWriter.Write(in)
	zlibWriter.Close()
	return err
}

func (c *zlibCompressor) Decompress(r io.Reader, bytes []byte) error {
	zlibReader, err := zlib.NewReader(r)
	if err != nil {
		return internal.WrapError(err, "failed creating zlib reader")
	}

	if _, err := io.ReadFull(zlibReader, bytes); err != nil {
		zlibReader.Close()
		return internal.WrapError(err, "failed decompressing using zlib")
	}
	zlibReader.Close()
	return nil
}
