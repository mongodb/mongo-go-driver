package compressor

import (
	"bytes"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func compress(t *testing.T, c Compressor, src []byte, expected []byte, dest []byte) {
	dest, err := c.CompressBytes(src, dest)
	if err != nil {
		t.Errorf("CompressBytes error: %v", err)
	}

	if !bytes.Equal(dest, expected) {
		t.Errorf("Unexpected compressed bytes. got %q; want %q", dest, expected)
	}
}

func uncompress(t *testing.T, c Compressor, src []byte, expected []byte, dest []byte) {
	dest, err := c.UncompressBytes(src, dest)
	if err != nil {
		t.Errorf("UncompressBytes error: %v", err)
	}

	if !bytes.Equal(dest, expected) {
		t.Errorf("Unexpected uncompressed bytes. got %q; want %q", dest, expected)
	}
}

func TestSnappyCompression(t *testing.T) {

	c := CreateSnappy()

	t.Run("CompressBytes", func(t *testing.T) {
		src := []byte("hello, world\n")
		expected := []byte("\r0hello, world\n")
		dest := make([]byte, len(expected))

		compress(t, c, src, expected, dest)
	})

	t.Run("UnompressBytes", func(t *testing.T) {
		src := []byte("\r0hello, world\n")
		expected := []byte("hello, world\n")
		dest := make([]byte, len(expected))

		uncompress(t, c, src, expected, dest)
	})

	t.Run("UnompressBytes Error", func(t *testing.T) {
		src := []byte("\x08\x0cabcd\x01\x05") // corrupt input (offset = 5, should be 4)
		dest := make([]byte, 5)

		dest, err := c.UncompressBytes(src, dest)
		if err == nil || !strings.Contains(err.Error(), "corrupt input") {
			t.Errorf("Expected corrupt input error, received: %v", err)
		}
	})

	t.Run("CompressorID", func(t *testing.T) {
		compressorID := c.CompressorID()

		if compressorID != 1 {
			t.Errorf("Unexpected compressor ID. got %d; want 1", compressorID)
		}
	})

	t.Run("Name", func(t *testing.T) {
		name := c.Name()

		if name != "snappy" {
			t.Errorf("Unexpected name. got %s; want snappy", name)
		}
	})
}

func TestZlibCompression(t *testing.T) {
	level := wiremessage.DefaultZlibLevel
	c, err := CreateZlib(&level)
	if err != nil {
		t.Errorf("CreateZlib error: %v", err)
	}

	t.Run("CompressBytes", func(t *testing.T) {
		src := []byte("hello, world\n")
		expected := []byte{120, 156, 202, 72, 205, 201, 201, 215, 81, 40, 207,
			47, 202, 73, 225, 2, 4, 0, 0, 255, 255, 33, 231, 4, 147}
		dest := make([]byte, 5) // len(dest) < len(expected) to test writer.Write()

		compress(t, c, src, expected, dest)
	})

	t.Run("UnompressBytes", func(t *testing.T) {
		src := []byte{120, 156, 202, 72, 205, 201, 201, 215, 81, 40, 207,
			47, 202, 73, 225, 2, 4, 0, 0, 255, 255, 33, 231, 4, 147}
		expected := []byte("hello, world\n")
		dest := make([]byte, len(expected)) // len(dest) must equal len(expected) for ZlibCompressor.UncompressBytes

		uncompress(t, c, src, expected, dest)
	})

	t.Run("UnompressBytes Error", func(t *testing.T) {
		src := []byte{77, 85} // corrupt input (invalid header)
		dest := make([]byte, 5)

		dest, err := c.UncompressBytes(src, dest)
		if err == nil || !strings.Contains(err.Error(), "invalid header") {
			t.Errorf("Expected invalid header error, received: %v", err)
		}
	})

	t.Run("CompressorID", func(t *testing.T) {
		compressorID := c.CompressorID()

		if compressorID != 2 {
			t.Errorf("Unexpected compressor ID. got %d; want 2", compressorID)
		}
	})

	t.Run("Name", func(t *testing.T) {
		name := c.Name()

		if name != "zlib" {
			t.Errorf("Unexpected name. got %s; want zlib", name)
		}
	})

	t.Run("CreateZlib", func(t *testing.T) {

		t.Run("with nil level", func(t *testing.T) {
			c, err := CreateZlib(nil)
			if err != nil {
				t.Errorf("CreateZlib error: %v", err)
			}

			// Check compressor is using default compression level
			src := []byte("hello, world\n")
			expected := []byte{120, 156, 202, 72, 205, 201, 201, 215, 81, 40, 207,
				47, 202, 73, 225, 2, 4, 0, 0, 255, 255, 33, 231, 4, 147}
			dest := make([]byte, 5) // len(dest) < len(expected) to test writer.Write()

			compress(t, c, src, expected, dest)
		})

		t.Run("with level lt 0", func(t *testing.T) {
			level := -1
			c, err := CreateZlib(&level)
			if err != nil {
				t.Errorf("CreateZlib error: %v", err)
			}

			// Check compressor is using default compression level
			src := []byte("hello, world\n")
			expected := []byte{120, 156, 202, 72, 205, 201, 201, 215, 81, 40, 207,
				47, 202, 73, 225, 2, 4, 0, 0, 255, 255, 33, 231, 4, 147}
			dest := make([]byte, 5) // len(dest) < len(expected) to test writer.Write()

			compress(t, c, src, expected, dest)
		})

		t.Run("with level gt 9", func(t *testing.T) {
			level := 10
			c, err := CreateZlib(&level)
			if err != nil {
				t.Errorf("CreateZlib error: %v", err)
			}

			// Check compressor is using max compression level
			src := []byte("hello, world\n")
			expected := []byte{120, 218, 202, 72, 205, 201, 201, 215, 81, 40, 207,
				47, 202, 73, 225, 2, 4, 0, 0, 255, 255, 33, 231, 4, 147}
			dest := make([]byte, 5) // len(dest) < len(expected) to test writer.Write()

			compress(t, c, src, expected, dest)
		})

	})
}
