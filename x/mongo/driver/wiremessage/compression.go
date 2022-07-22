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
	"sync"

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

var zstdEncoders = &sync.Map{}

func getZstdEncoder(l zstd.EncoderLevel) (*zstd.Encoder, error) {
	v, ok := zstdEncoders.Load(l)
	if ok {
		return v.(*zstd.Encoder), nil
	}
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(l))
	if err != nil {
		return nil, err
	}
	zstdEncoders.Store(l, encoder)
	return encoder, nil
}

// CompressPayload takes a byte slice and compresses it according to the options passed
func CompressPayload(in []byte, opts CompressionOpts) ([]byte, error) {
	switch opts.Compressor {
	case CompressorNoOp:
		return in, nil
	case CompressorSnappy:
		return snappy.Encode(nil, in), nil
	case CompressorZLib:
		var b bytes.Buffer
		w, err := zlib.NewWriterLevel(&b, opts.ZlibLevel)
		if err != nil {
			return nil, err
		}
		_, err = w.Write(in)
		if err != nil {
			return nil, err
		}
		err = w.Close()
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	case CompressorZstd:
		encoder, err := getZstdEncoder(zstd.EncoderLevelFromZstd(opts.ZstdLevel))
		if err != nil {
			return nil, err
		}
		return encoder.EncodeAll(in, nil), nil
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}

// DecompressPayload takes a byte slice that has been compressed and undoes it according to the options passed
func DecompressPayload(in []byte, opts CompressionOpts) ([]byte, error) {
	switch opts.Compressor {
	case CompressorNoOp:
		return in, nil
	case CompressorSnappy:
		uncompressed := make([]byte, opts.UncompressedSize)
		return snappy.Decode(uncompressed, in)
	case CompressorZLib:
		decompressor, err := zlib.NewReader(bytes.NewReader(in))
		if err != nil {
			return nil, err
		}
		uncompressed := make([]byte, opts.UncompressedSize)
		_, err = io.ReadFull(decompressor, uncompressed)
		if err != nil {
			return nil, err
		}
		return uncompressed, nil
	case CompressorZstd:
		w, err := zstd.NewReader(bytes.NewBuffer(in))
		if err != nil {
			return nil, err
		}
		defer w.Close()
		var b bytes.Buffer
		_, err = io.Copy(&b, w)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}

// NewDecompressedReader ...
func NewDecompressedReader(r io.Reader, opts CompressionOpts) (io.Reader, error) {
	switch opts.Compressor {
	case CompressorNoOp:
		return r, nil
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}
