// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func TestCompression(t *testing.T) {
	compressors := []wiremessage.CompressorID{
		wiremessage.CompressorNoOp,
		wiremessage.CompressorSnappy,
		wiremessage.CompressorZLib,
		wiremessage.CompressorZstd,
	}

	for _, compressor := range compressors {
		t.Run(strconv.Itoa(int(compressor)), func(t *testing.T) {
			payload := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt")
			opts := CompressionOpts{
				Compressor:       compressor,
				ZlibLevel:        wiremessage.DefaultZlibLevel,
				ZstdLevel:        wiremessage.DefaultZstdLevel,
				UncompressedSize: int32(len(payload)),
			}
			compressed, err := CompressPayload(payload, opts)
			assert.NoError(t, err)
			assert.NotEqual(t, 0, len(compressed))
			decompressed, err := DecompressPayload(compressed, opts)
			assert.NoError(t, err)
			assert.Equal(t, payload, decompressed)
		})
	}
}

func TestZstdWindowSize(t *testing.T) {
	tests := []struct {
		inputSize  int
		level      zstd.EncoderLevel
		windowSize int
	}{
		{
			inputSize:  512,
			level:      zstd.EncoderLevelFromZstd(wiremessage.DefaultZstdLevel),
			windowSize: 1024,
		},
		{
			inputSize:  512000,
			level:      zstd.EncoderLevelFromZstd(wiremessage.DefaultZstdLevel),
			windowSize: 524288,
		},
		{
			inputSize:  16000000,
			level:      zstd.EncoderLevelFromZstd(wiremessage.DefaultZstdLevel),
			windowSize: 16777216,
		},
		{
			inputSize:  32000000,
			level:      zstd.EncoderLevelFromZstd(wiremessage.DefaultZstdLevel),
			windowSize: 16777216,
		},
	}

	for _, test := range tests {
		t.Run(strconv.Itoa(test.inputSize), func(t *testing.T) {
			windowSize := calcZstdWindowSize(test.inputSize, test.level)
			assert.Equal(t, test.windowSize, windowSize)
		})
	}
}

func BenchmarkCompression(b *testing.B) {
	payload := func() []byte {
		f, err := os.Open("compression.go")
		if err != nil {
			panic(err)
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		for i := 1; i < 10; i++ {
			buf = append(buf, buf...)
		}
		return buf
	}()

	compressors := []wiremessage.CompressorID{
		wiremessage.CompressorSnappy,
		wiremessage.CompressorZLib,
		wiremessage.CompressorZstd,
	}

	for _, compressor := range compressors {
		b.Run(strconv.Itoa(int(compressor)), func(b *testing.B) {
			opts := CompressionOpts{
				Compressor: compressor,
				ZlibLevel:  wiremessage.DefaultZlibLevel,
				ZstdLevel:  wiremessage.DefaultZstdLevel,
			}
			for i := 0; i < b.N; i++ {
				_, err := CompressPayload(payload, opts)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}
