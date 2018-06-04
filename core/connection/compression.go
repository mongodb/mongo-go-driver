package connection

import (
	"bytes"
	"compress/zlib"

	"io"

	"github.com/golang/snappy"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Compressor is the interface implemented by types that can compress and decompress wire messages. This is used
// when sending and receiving messages to and from the server.
type Compressor interface {
	compressBytes(src, dest []byte) ([]byte, error)
	uncompressBytes(src, dest []byte) ([]byte, error)
	compressorID() wiremessage.CompressorID
	name() string
}

type writer struct {
	buf []byte
}

func (w *writer) Write(p []byte) (n int, err error) {
	index := len(w.buf)
	if len(p) > cap(w.buf)-index {
		buf := make([]byte, 2*cap(w.buf)+len(p))
		copy(buf, w.buf)
		w.buf = buf
	}

	w.buf = w.buf[:index+len(p)]
	copy(w.buf[index:], p)
	return len(p), nil
}

// SnappyCompressor uses the snappy method to compress data
type SnappyCompressor struct {
}

// ZlibCompressor uses the zlib method to compress data
type ZlibCompressor struct {
	level      int
	zlibWriter *zlib.Writer
}

func (s *SnappyCompressor) compressBytes(src, dest []byte) ([]byte, error) {
	dest = dest[:0]
	dest = snappy.Encode(dest, src)
	return dest, nil
}

func (s *SnappyCompressor) uncompressBytes(src, dest []byte) ([]byte, error) {
	var err error
	dest, err = snappy.Decode(dest, src)
	if err != nil {
		return dest, err
	}

	return dest, nil
}

func (s *SnappyCompressor) compressorID() wiremessage.CompressorID {
	return wiremessage.CompressorSnappy
}

func (s *SnappyCompressor) name() string {
	return "snappy"
}

func (z *ZlibCompressor) compressBytes(src, dest []byte) ([]byte, error) {
	dest = dest[:0]
	z.zlibWriter.Reset(&writer{
		buf: dest,
	})

	_, err := z.zlibWriter.Write(src)
	if err != nil {
		_ = z.zlibWriter.Close()
		return dest, err
	}

	err = z.zlibWriter.Close()
	if err != nil {
		return dest, err
	}
	return dest, nil
}

// assume dest is empty and is exact size that needs to be read
func (z *ZlibCompressor) uncompressBytes(src, dest []byte) ([]byte, error) {
	reader := bytes.NewReader(src)
	zlibReader, err := zlib.NewReader(reader)

	if err != nil {
		return dest, err
	}
	defer func() {
		_ = zlibReader.Close()
	}()

	_, err = io.ReadFull(zlibReader, dest)
	if err != nil {
		return dest, err
	}

	return dest, nil
}

func (z *ZlibCompressor) compressorID() wiremessage.CompressorID {
	return wiremessage.CompressorZLib
}

func (z *ZlibCompressor) name() string {
	return "zlib"
}

// CreateSnappy creates a snappy compressor
func CreateSnappy() Compressor {
	return &SnappyCompressor{}
}

// CreateZlib creates a zlib compressor
func CreateZlib(level int) (Compressor, error) {
	if level < 0 {
		level = wiremessage.DefaultZlibLevel
	}

	var compressBuf bytes.Buffer
	zlibWriter, err := zlib.NewWriterLevel(&compressBuf, level)

	if err != nil {
		return &ZlibCompressor{}, err
	}

	return &ZlibCompressor{
		level:      level,
		zlibWriter: zlibWriter,
	}, nil
}
