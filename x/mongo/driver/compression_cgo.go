// +build cgo

package driver

import (
	"bytes"
	"io"

	"github.com/DataDog/zstd"
)

func zstdCompress(in []byte, level int) ([]byte, error) {
	var b bytes.Buffer
	w := zstd.NewWriterLevel(&b, level)
	_, err := w.Write(in)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func zstdDecompress(in []byte, size int32) ([]byte, error) {
	out := make([]byte, size)
	decompressor := zstd.NewReader(bytes.NewReader(in))
	_, err := io.ReadFull(decompressor, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
