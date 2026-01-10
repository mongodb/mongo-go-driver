// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package binaryutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// Test values matching stdlib pattern for comprehensive coverage.
var testValues64 = []uint64{
	0x0000000000000000,
	0x0123456789abcdef,
	0xfedcba9876543210,
	0xffffffffffffffff,
	0xaaaaaaaaaaaaaaaa,
	math.Float64bits(math.Pi),
	math.Float64bits(math.E),
}

var testValues32 = []uint32{
	0x00000000,
	0x01234567,
	0xfedcba98,
	0xffffffff,
	0xdeadbeef,
	0xaaaaaaaa,
}

// -----------------------------------------------------------------------------
// Benchmark Tests - Append Functions
// -----------------------------------------------------------------------------

func BenchmarkAppend32(b *testing.B) {
	b.SetBytes(4)
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 0, 4)
		i := uint32(0)
		for pb.Next() {
			buf = Append32(buf[:0], i)
			i++
		}
	})
}

func BenchmarkAppend64(b *testing.B) {
	b.SetBytes(8)
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 0, 8)
		i := uint64(0)
		for pb.Next() {
			buf = Append64(buf[:0], i)
			i++
		}
	})
}

// Comparison benchmarks against stdlib LittleEndian.
func BenchmarkStdlibAppendUint32(b *testing.B) {
	b.SetBytes(4)
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 0, 4)
		i := uint32(0)
		for pb.Next() {
			buf = binary.LittleEndian.AppendUint32(buf[:0], i)
			i++
		}
	})
}

func BenchmarkStdlibAppendUint64(b *testing.B) {
	b.SetBytes(8)
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 0, 8)
		i := uint64(0)
		for pb.Next() {
			buf = binary.LittleEndian.AppendUint64(buf[:0], i)
			i++
		}
	})
}

// -----------------------------------------------------------------------------
// Benchmark Tests - Read Functions
// -----------------------------------------------------------------------------

func BenchmarkReadU32(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04}
	b.SetBytes(4)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadU32(buf)
		}
	})
}

func BenchmarkReadI32(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04}
	b.SetBytes(4)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadI32(buf)
		}
	})
}

func BenchmarkReadU64(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	b.SetBytes(8)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadU64(buf)
		}
	})
}

func BenchmarkReadI64(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	b.SetBytes(8)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadI64(buf)
		}
	})
}

// Comparison benchmarks against stdlib LittleEndian.
func BenchmarkStdlibUint32(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04}
	b.SetBytes(4)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = binary.LittleEndian.Uint32(buf)
		}
	})
}

func BenchmarkStdlibUint64(b *testing.B) {
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	b.SetBytes(8)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = binary.LittleEndian.Uint64(buf)
		}
	})
}

// -----------------------------------------------------------------------------
// Benchmark Tests - CString Functions
// -----------------------------------------------------------------------------

func BenchmarkReadCString(b *testing.B) {
	buf := []byte("hello world\x00remaining data")
	b.SetBytes(int64(len("hello world") + 1))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadCString(buf)
		}
	})
}

func BenchmarkReadCStringBytes(b *testing.B) {
	buf := []byte("hello world\x00remaining data")
	b.SetBytes(int64(len("hello world") + 1))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadCStringBytes(buf)
		}
	})
}

func BenchmarkReadCStringLong(b *testing.B) {
	// Test with a longer string to see IndexByte performance.
	longStr := make([]byte, 256)
	for i := range longStr {
		longStr[i] = byte('a' + (i % 26))
	}
	longStr[255] = 0x00
	b.SetBytes(256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = ReadCString(longStr)
		}
	})
}

// -----------------------------------------------------------------------------
// Unit Tests - Round-trip Tests
// -----------------------------------------------------------------------------

func TestAppend32RoundTrip(t *testing.T) {
	t.Parallel()
	for _, want := range testValues32 {
		want := want // capture range variable
		t.Run(fmt.Sprintf("%#x", want), func(t *testing.T) {
			t.Parallel()
			buf := Append32(nil, want)
			got, rem, ok := ReadU32(buf)
			if !ok {
				t.Errorf("ReadU32 failed for value %#x", want)
				return
			}
			if len(rem) != 0 {
				t.Errorf("ReadU32(%#x): remaining bytes = %d, want 0", want, len(rem))
			}
			if got != want {
				t.Errorf("Append32/ReadU32 round-trip: got %#x, want %#x", got, want)
			}
		})
	}
}

func TestAppend32RoundTripSigned(t *testing.T) {
	t.Parallel()
	signedValues := []int32{
		0,
		1,
		-1,
		math.MaxInt32,
		math.MinInt32,
		0x01234567,
		-0x01234567,
	}
	for _, want := range signedValues {
		want := want // capture range variable
		t.Run(fmt.Sprintf("%d", want), func(t *testing.T) {
			t.Parallel()
			buf := Append32(nil, want)
			got, rem, ok := ReadI32(buf)
			if !ok {
				t.Errorf("ReadI32 failed for value %d", want)
				return
			}
			if len(rem) != 0 {
				t.Errorf("ReadI32(%d): remaining bytes = %d, want 0", want, len(rem))
			}
			if got != want {
				t.Errorf("Append32/ReadI32 round-trip: got %d, want %d", got, want)
			}
		})
	}
}

func TestAppend64RoundTrip(t *testing.T) {
	t.Parallel()
	for _, want := range testValues64 {
		want := want // capture range variable
		t.Run(fmt.Sprintf("%#x", want), func(t *testing.T) {
			t.Parallel()
			buf := Append64(nil, want)
			got, rem, ok := ReadU64(buf)
			if !ok {
				t.Errorf("ReadU64 failed for value %#x", want)
				return
			}
			if len(rem) != 0 {
				t.Errorf("ReadU64(%#x): remaining bytes = %d, want 0", want, len(rem))
			}
			if got != want {
				t.Errorf("Append64/ReadU64 round-trip: got %#x, want %#x", got, want)
			}
		})
	}
}

func TestAppend64RoundTripSigned(t *testing.T) {
	t.Parallel()
	signedValues := []int64{
		0,
		1,
		-1,
		math.MaxInt64,
		math.MinInt64,
		0x0123456789abcdef,
		-0x0123456789abcdef,
	}
	for _, want := range signedValues {
		want := want // capture range variable
		t.Run(fmt.Sprintf("%d", want), func(t *testing.T) {
			t.Parallel()
			buf := Append64(nil, want)
			got, rem, ok := ReadI64(buf)
			if !ok {
				t.Errorf("ReadI64 failed for value %d", want)
				return
			}
			if len(rem) != 0 {
				t.Errorf("ReadI64(%d): remaining bytes = %d, want 0", want, len(rem))
			}
			if got != want {
				t.Errorf("Append64/ReadI64 round-trip: got %d, want %d", got, want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Stdlib Compatibility
// -----------------------------------------------------------------------------

func TestAppend32MatchesStdlib(t *testing.T) {
	t.Parallel()
	for _, v := range testValues32 {
		v := v // capture range variable
		t.Run(fmt.Sprintf("%#x", v), func(t *testing.T) {
			t.Parallel()
			got := Append32(nil, v)
			want := binary.LittleEndian.AppendUint32(nil, v)
			if !bytes.Equal(got, want) {
				t.Errorf("Append32(%#x): got %v, want %v", v, got, want)
			}
		})
	}
}

func TestAppend64MatchesStdlib(t *testing.T) {
	t.Parallel()
	for _, v := range testValues64 {
		v := v // capture range variable
		t.Run(fmt.Sprintf("%#x", v), func(t *testing.T) {
			t.Parallel()
			got := Append64(nil, v)
			want := binary.LittleEndian.AppendUint64(nil, v)
			if !bytes.Equal(got, want) {
				t.Errorf("Append64(%#x): got %v, want %v", v, got, want)
			}
		})
	}
}

func TestReadU32MatchesStdlib(t *testing.T) {
	t.Parallel()
	for _, v := range testValues32 {
		v := v // capture range variable
		t.Run(fmt.Sprintf("%#x", v), func(t *testing.T) {
			t.Parallel()
			buf := binary.LittleEndian.AppendUint32(nil, v)
			got, _, ok := ReadU32(buf)
			if !ok {
				t.Errorf("ReadU32 failed for value %#x", v)
				return
			}
			want := binary.LittleEndian.Uint32(buf)
			if got != want {
				t.Errorf("ReadU32: got %#x, want %#x", got, want)
			}
		})
	}
}

func TestReadU64MatchesStdlib(t *testing.T) {
	t.Parallel()
	for _, v := range testValues64 {
		v := v // capture range variable
		t.Run(fmt.Sprintf("%#x", v), func(t *testing.T) {
			t.Parallel()
			buf := binary.LittleEndian.AppendUint64(nil, v)
			got, _, ok := ReadU64(buf)
			if !ok {
				t.Errorf("ReadU64 failed for value %#x", v)
				return
			}
			want := binary.LittleEndian.Uint64(buf)
			if got != want {
				t.Errorf("ReadU64: got %#x, want %#x", got, want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Bounds Checking / Short Slices
// -----------------------------------------------------------------------------

func TestReadU32ShortSlice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		buf  []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x01}},
		{"2 bytes", []byte{0x01, 0x02}},
		{"3 bytes", []byte{0x01, 0x02, 0x03}},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, rem, ok := ReadU32(tc.buf)
			if ok {
				t.Errorf("ReadU32(%v): expected ok=false, got ok=true", tc.buf)
			}
			if val != 0 {
				t.Errorf("ReadU32(%v): expected val=0, got val=%d", tc.buf, val)
			}
			if len(rem) != len(tc.buf) {
				t.Errorf("ReadU32(%v): expected rem len=%d, got len=%d", tc.buf, len(tc.buf), len(rem))
			}
		})
	}
}

func TestReadI32ShortSlice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		buf  []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x01}},
		{"2 bytes", []byte{0x01, 0x02}},
		{"3 bytes", []byte{0x01, 0x02, 0x03}},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, rem, ok := ReadI32(tc.buf)
			if ok {
				t.Errorf("ReadI32(%v): expected ok=false, got ok=true", tc.buf)
			}
			if val != 0 {
				t.Errorf("ReadI32(%v): expected val=0, got val=%d", tc.buf, val)
			}
			if len(rem) != len(tc.buf) {
				t.Errorf("ReadI32(%v): expected rem len=%d, got len=%d", tc.buf, len(tc.buf), len(rem))
			}
		})
	}
}

func TestReadU64ShortSlice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		buf  []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x01}},
		{"4 bytes", []byte{0x01, 0x02, 0x03, 0x04}},
		{"7 bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, rem, ok := ReadU64(tc.buf)
			if ok {
				t.Errorf("ReadU64(%v): expected ok=false, got ok=true", tc.buf)
			}
			if val != 0 {
				t.Errorf("ReadU64(%v): expected val=0, got val=%d", tc.buf, val)
			}
			if len(rem) != len(tc.buf) {
				t.Errorf("ReadU64(%v): expected rem len=%d, got len=%d", tc.buf, len(tc.buf), len(rem))
			}
		})
	}
}

func TestReadI64ShortSlice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		buf  []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x01}},
		{"4 bytes", []byte{0x01, 0x02, 0x03, 0x04}},
		{"7 bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, rem, ok := ReadI64(tc.buf)
			if ok {
				t.Errorf("ReadI64(%v): expected ok=false, got ok=true", tc.buf)
			}
			if val != 0 {
				t.Errorf("ReadI64(%v): expected val=0, got val=%d", tc.buf, val)
			}
			if len(rem) != len(tc.buf) {
				t.Errorf("ReadI64(%v): expected rem len=%d, got len=%d", tc.buf, len(tc.buf), len(rem))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Remaining Slice Handling
// -----------------------------------------------------------------------------

func TestReadU32ReturnsRemaining(t *testing.T) {
	t.Parallel()
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	_, rem, ok := ReadU32(buf)
	if !ok {
		t.Fatal("ReadU32 failed")
	}
	if len(rem) != 4 {
		t.Errorf("ReadU32: remaining len = %d, want 4", len(rem))
	}
	// Verify remaining bytes are correct.
	want := []byte{0x05, 0x06, 0x07, 0x08}
	if !bytes.Equal(rem, want) {
		t.Errorf("ReadU32: remaining = %v, want %v", rem, want)
	}
}

func TestReadU64ReturnsRemaining(t *testing.T) {
	t.Parallel()
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	_, rem, ok := ReadU64(buf)
	if !ok {
		t.Fatal("ReadU64 failed")
	}
	if len(rem) != 2 {
		t.Errorf("ReadU64: remaining len = %d, want 2", len(rem))
	}
	want := []byte{0x09, 0x0a}
	if !bytes.Equal(rem, want) {
		t.Errorf("ReadU64: remaining = %v, want %v", rem, want)
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - CString Functions
// -----------------------------------------------------------------------------

func TestReadCString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		input   []byte
		wantStr string
		wantRem []byte
		wantOK  bool
	}{
		{
			name:    "simple string",
			input:   []byte("hello\x00world"),
			wantStr: "hello",
			wantRem: []byte("world"),
			wantOK:  true,
		},
		{
			name:    "empty string",
			input:   []byte("\x00remaining"),
			wantStr: "",
			wantRem: []byte("remaining"),
			wantOK:  true,
		},
		{
			name:    "no null terminator",
			input:   []byte("hello world"),
			wantStr: "",
			wantRem: []byte("hello world"),
			wantOK:  false,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantStr: "",
			wantRem: []byte{},
			wantOK:  false,
		},
		{
			name:    "only null byte",
			input:   []byte{0x00},
			wantStr: "",
			wantRem: []byte{},
			wantOK:  true,
		},
		{
			name:    "null at end",
			input:   []byte("test\x00"),
			wantStr: "test",
			wantRem: []byte{},
			wantOK:  true,
		},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStr, gotRem, gotOK := ReadCString(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("ReadCString(%q): ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotStr != tc.wantStr {
				t.Errorf("ReadCString(%q): str = %q, want %q", tc.input, gotStr, tc.wantStr)
			}
			if string(gotRem) != string(tc.wantRem) {
				t.Errorf("ReadCString(%q): rem = %q, want %q", tc.input, gotRem, tc.wantRem)
			}
		})
	}
}

func TestReadCStringBytes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		input     []byte
		wantBytes []byte
		wantRem   []byte
		wantOK    bool
	}{
		{
			name:      "simple string",
			input:     []byte("hello\x00world"),
			wantBytes: []byte("hello"),
			wantRem:   []byte("world"),
			wantOK:    true,
		},
		{
			name:      "empty string",
			input:     []byte("\x00remaining"),
			wantBytes: []byte{},
			wantRem:   []byte("remaining"),
			wantOK:    true,
		},
		{
			name:      "no null terminator",
			input:     []byte("hello world"),
			wantBytes: nil,
			wantRem:   []byte("hello world"),
			wantOK:    false,
		},
		{
			name:      "empty input",
			input:     []byte{},
			wantBytes: nil,
			wantRem:   []byte{},
			wantOK:    false,
		},
		{
			name:      "binary data with null",
			input:     []byte{0xff, 0xfe, 0x00, 0x01, 0x02},
			wantBytes: []byte{0xff, 0xfe},
			wantRem:   []byte{0x01, 0x02},
			wantOK:    true,
		},
	}
	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotBytes, gotRem, gotOK := ReadCStringBytes(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("ReadCStringBytes(%v): ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if string(gotBytes) != string(tc.wantBytes) {
				t.Errorf("ReadCStringBytes(%v): bytes = %v, want %v", tc.input, gotBytes, tc.wantBytes)
			}
			if string(gotRem) != string(tc.wantRem) {
				t.Errorf("ReadCStringBytes(%v): rem = %v, want %v", tc.input, gotRem, tc.wantRem)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Append to Existing Buffer
// -----------------------------------------------------------------------------

func TestAppend32ToExistingBuffer(t *testing.T) {
	t.Parallel()
	prefix := []byte{0xaa, 0xbb, 0xcc}
	buf := Append32(prefix, uint32(0x12345678))
	if len(buf) != 7 {
		t.Errorf("Append32 to prefix: len = %d, want 7", len(buf))
	}
	// Verify prefix is unchanged.
	if buf[0] != 0xaa || buf[1] != 0xbb || buf[2] != 0xcc {
		t.Errorf("Append32: prefix corrupted: %v", buf[:3])
	}
	// Read back the value.
	val, _, ok := ReadU32(buf[3:])
	if !ok {
		t.Fatal("ReadU32 failed after Append32")
	}
	if val != 0x12345678 {
		t.Errorf("ReadU32 after Append32: got %#x, want 0x12345678", val)
	}
}

func TestAppend64ToExistingBuffer(t *testing.T) {
	t.Parallel()
	prefix := []byte{0xaa, 0xbb, 0xcc}
	buf := Append64(prefix, uint64(0x123456789abcdef0))
	if len(buf) != 11 {
		t.Errorf("Append64 to prefix: len = %d, want 11", len(buf))
	}
	// Verify prefix is unchanged.
	if buf[0] != 0xaa || buf[1] != 0xbb || buf[2] != 0xcc {
		t.Errorf("Append64: prefix corrupted: %v", buf[:3])
	}
	// Read back the value.
	val, _, ok := ReadU64(buf[3:])
	if !ok {
		t.Fatal("ReadU64 failed after Append64")
	}
	if val != 0x123456789abcdef0 {
		t.Errorf("ReadU64 after Append64: got %#x, want 0x123456789abcdef0", val)
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Specific Byte Patterns (Little-Endian Verification)
// -----------------------------------------------------------------------------

func TestAppend32ByteOrder(t *testing.T) {
	t.Parallel()
	// 0x04030201 in little-endian should be [0x01, 0x02, 0x03, 0x04].
	buf := Append32(nil, uint32(0x04030201))
	want := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(buf, want) {
		t.Errorf("Append32: got %v, want %v", buf, want)
	}
}

func TestAppend64ByteOrder(t *testing.T) {
	t.Parallel()
	// 0x0807060504030201 in little-endian should be [0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08].
	buf := Append64(nil, uint64(0x0807060504030201))
	want := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if !bytes.Equal(buf, want) {
		t.Errorf("Append64: got %v, want %v", buf, want)
	}
}

func TestReadU32ByteOrder(t *testing.T) {
	t.Parallel()
	// [0x01, 0x02, 0x03, 0x04] in little-endian should be 0x04030201.
	buf := []byte{0x01, 0x02, 0x03, 0x04}
	val, _, ok := ReadU32(buf)
	if !ok {
		t.Fatal("ReadU32 failed")
	}
	if val != 0x04030201 {
		t.Errorf("ReadU32: got %#x, want 0x04030201", val)
	}
}

func TestReadU64ByteOrder(t *testing.T) {
	t.Parallel()
	// [0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08] in little-endian should be 0x0807060504030201.
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	val, _, ok := ReadU64(buf)
	if !ok {
		t.Fatal("ReadU64 failed")
	}
	if val != 0x0807060504030201 {
		t.Errorf("ReadU64: got %#x, want 0x0807060504030201", val)
	}
}

// -----------------------------------------------------------------------------
// Unit Tests - Multiple Sequential Reads
// -----------------------------------------------------------------------------

func TestMultipleSequentialReads(t *testing.T) {
	t.Parallel()
	// Build a buffer with multiple values.
	var buf []byte
	buf = Append32(buf, uint32(0x11111111))
	buf = Append64(buf, uint64(0x2222222222222222))
	buf = Append32(buf, uint32(0x33333333))
	buf = append(buf, []byte("test\x00")...)

	// Read them back sequentially.
	var ok bool
	var v32 uint32
	var v64 uint64
	var str string

	v32, buf, ok = ReadU32(buf)
	if !ok || v32 != 0x11111111 {
		t.Errorf("First ReadU32: got %#x, ok=%v", v32, ok)
	}

	v64, buf, ok = ReadU64(buf)
	if !ok || v64 != 0x2222222222222222 {
		t.Errorf("ReadU64: got %#x, ok=%v", v64, ok)
	}

	v32, buf, ok = ReadU32(buf)
	if !ok || v32 != 0x33333333 {
		t.Errorf("Second ReadU32: got %#x, ok=%v", v32, ok)
	}

	str, buf, ok = ReadCString(buf)
	if !ok || str != "test" {
		t.Errorf("ReadCString: got %q, ok=%v", str, ok)
	}

	if len(buf) != 0 {
		t.Errorf("Buffer not empty after all reads: %d bytes remaining", len(buf))
	}
}
