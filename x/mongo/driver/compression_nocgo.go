// +build !cgo

package driver

import "errors"

func zstdCompress(_ []byte, _ int) ([]byte, error) {
	return nil, errors.New("zstd support requires cgo")
}

func zstdDecompress(in []byte, size int32) ([]byte, error) {
	return nil, errors.New("zstd support requires cgo")
}
