// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"bytes"
	"compress/zlib"
	"os"
	"testing"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"

	"go.mongodb.org/mongo-driver/internal/assert"
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
		t.Run(compressor.String(), func(t *testing.T) {
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

func TestCompressionLevels(t *testing.T) {
	in := []byte("abc")
	wr := new(bytes.Buffer)

	t.Run("ZLib", func(t *testing.T) {
		opts := CompressionOpts{
			Compressor: wiremessage.CompressorZLib,
		}
		for lvl := zlib.HuffmanOnly - 2; lvl < zlib.BestCompression+2; lvl++ {
			opts.ZlibLevel = lvl
			_, err1 := CompressPayload(in, opts)
			_, err2 := zlib.NewWriterLevel(wr, lvl)
			if err2 != nil {
				assert.Error(t, err1, "expected an error for ZLib level %d", lvl)
			} else {
				assert.NoError(t, err1, "unexpected error for ZLib level %d", lvl)
			}
		}
	})

	t.Run("Zstd", func(t *testing.T) {
		opts := CompressionOpts{
			Compressor: wiremessage.CompressorZstd,
		}
		for lvl := zstd.SpeedFastest - 2; lvl < zstd.SpeedBestCompression+2; lvl++ {
			opts.ZstdLevel = int(lvl)
			_, err1 := CompressPayload(in, opts)
			_, err2 := zstd.NewWriter(wr, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(opts.ZstdLevel)))
			if err2 != nil {
				assert.Error(t, err1, "expected an error for Zstd level %d", lvl)
			} else {
				assert.NoError(t, err1, "unexpected error for Zstd level %d", lvl)
			}
		}
	})
}

func TestDecompressFailures(t *testing.T) {
	t.Parallel()

	t.Run("snappy decompress huge size", func(t *testing.T) {
		t.Parallel()

		opts := CompressionOpts{
			Compressor:       wiremessage.CompressorSnappy,
			UncompressedSize: 100, // reasonable size
		}
		// Compressed data is twice as large as declared above.
		// In test we use actual compression so that the decompress action would pass without fix (thus failing test).
		// When decompression starts it allocates a buffer of the defined size, regardless of a valid compressed body following.
		compressedData, err := CompressPayload(make([]byte, opts.UncompressedSize*2), opts)
		assert.NoError(t, err, "premature error making compressed example")

		_, err = DecompressPayload(compressedData, opts)
		assert.Error(t, err)
	})
}

var (
	compressionPayload      []byte
	compressedSnappyPayload []byte
	compressedZLibPayload   []byte
	compressedZstdPayload   []byte
)

func initCompressionPayload(b *testing.B) {
	if compressionPayload != nil {
		return
	}
	data, err := os.ReadFile("testdata/compression.go")
	if err != nil {
		b.Fatal(err)
	}
	for i := 1; i < 10; i++ {
		data = append(data, data...)
	}
	compressionPayload = data

	compressedSnappyPayload = snappy.Encode(compressedSnappyPayload[:0], data)

	{
		var buf bytes.Buffer
		enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			b.Fatal(err)
		}
		compressedZstdPayload = enc.EncodeAll(data, nil)
	}

	{
		var buf bytes.Buffer
		enc := zlib.NewWriter(&buf)
		if _, err := enc.Write(data); err != nil {
			b.Fatal(err)
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
		compressedZLibPayload = append(compressedZLibPayload[:0], buf.Bytes()...)
	}

	b.ResetTimer()
}

func BenchmarkCompressPayload(b *testing.B) {
	initCompressionPayload(b)

	compressors := []wiremessage.CompressorID{
		wiremessage.CompressorSnappy,
		wiremessage.CompressorZLib,
		wiremessage.CompressorZstd,
	}

	for _, compressor := range compressors {
		b.Run(compressor.String(), func(b *testing.B) {
			opts := CompressionOpts{
				Compressor: compressor,
				ZlibLevel:  wiremessage.DefaultZlibLevel,
				ZstdLevel:  wiremessage.DefaultZstdLevel,
			}
			payload := compressionPayload
			b.SetBytes(int64(len(payload)))
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := CompressPayload(payload, opts)
					if err != nil {
						b.Error(err)
					}
				}
			})
		})
	}
}

func BenchmarkDecompressPayload(b *testing.B) {
	initCompressionPayload(b)

	benchmarks := []struct {
		compressor wiremessage.CompressorID
		payload    []byte
	}{
		{wiremessage.CompressorSnappy, compressedSnappyPayload},
		{wiremessage.CompressorZLib, compressedZLibPayload},
		{wiremessage.CompressorZstd, compressedZstdPayload},
	}

	for _, bench := range benchmarks {
		b.Run(bench.compressor.String(), func(b *testing.B) {
			opts := CompressionOpts{
				Compressor:       bench.compressor,
				ZlibLevel:        wiremessage.DefaultZlibLevel,
				ZstdLevel:        wiremessage.DefaultZstdLevel,
				UncompressedSize: int32(len(compressionPayload)),
			}
			payload := bench.payload
			b.SetBytes(int64(len(compressionPayload)))
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := DecompressPayload(payload, opts)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
