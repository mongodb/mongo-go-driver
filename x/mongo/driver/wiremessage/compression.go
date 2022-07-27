// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
)

// CompressionOpts holds settings for how to compress a payload
type CompressionOpts struct {
	Compressor       CompressorID
	ZlibLevel        int
	ZstdLevel        int
	UncompressedSize int32
}

// NewDecompressedReader ...
func NewDecompressedReader(r io.Reader, opts CompressionOpts) (io.Reader, error) {
	switch opts.Compressor {
	case CompressorNoOp:
		return r, nil
	case CompressorSnappy:
		compressed, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		uncompressed, err := snappy.Decode(nil, compressed)
		return bytes.NewBuffer(uncompressed), err
	case CompressorZLib:
		return zlib.NewReader(r)
	case CompressorZstd:
		return zstd.NewReader(r)
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}
