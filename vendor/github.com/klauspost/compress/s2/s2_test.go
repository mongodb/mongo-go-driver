// Copyright 2011 The Snappy-Go Authors. All rights reserved.
// Copyright (c) 2019 Klaus Post. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s2

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/klauspost/compress/snappy"
)

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)

var (
	download     = flag.Bool("download", false, "If true, download any missing files before running benchmarks")
	testdataDir  = flag.String("testdataDir", "testdata", "Directory containing the test data")
	benchdataDir = flag.String("benchdataDir", "testdata/bench", "Directory containing the benchmark data")
)

func TestMaxEncodedLen(t *testing.T) {
	testSet := []struct {
		in, out int64
	}{
		{in: 0, out: 1},
		{in: 1 << 24, out: 1<<24 + int64(binary.PutVarint([]byte{binary.MaxVarintLen32: 0}, int64(1<<24))) + literalExtraSize(1<<24)},
		{in: MaxBlockSize, out: math.MaxUint32},
		{in: math.MaxUint32 - binary.MaxVarintLen32 - literalExtraSize(math.MaxUint32), out: math.MaxUint32},
		{in: math.MaxUint32 - 9, out: -1},
		{in: math.MaxUint32 - 8, out: -1},
		{in: math.MaxUint32 - 7, out: -1},
		{in: math.MaxUint32 - 6, out: -1},
		{in: math.MaxUint32 - 5, out: -1},
		{in: math.MaxUint32 - 4, out: -1},
		{in: math.MaxUint32 - 3, out: -1},
		{in: math.MaxUint32 - 2, out: -1},
		{in: math.MaxUint32 - 1, out: -1},
		{in: math.MaxUint32, out: -1},
		{in: -1, out: -1},
		{in: -2, out: -1},
	}
	// 32 bit platforms have a different threshold.
	if maxInt == math.MaxInt32 {
		testSet[2].out = -1
		testSet[3].out = -1
	}
	// Test all sizes up to maxBlockSize.
	for i := int64(0); i < maxBlockSize; i++ {
		testSet = append(testSet, struct{ in, out int64 }{in: i, out: i + int64(binary.PutVarint([]byte{binary.MaxVarintLen32: 0}, i)) + literalExtraSize(i)})
	}
	for i := range testSet {
		tt := testSet[i]
		want := tt.out
		got := int64(MaxEncodedLen(int(tt.in)))
		if got != want {
			t.Fatalf("input: %d, want: %d, got: %d", tt.in, want, got)
		}
	}
}

func cmp(a, b []byte) error {
	if bytes.Equal(a, b) {
		return nil
	}
	if len(a) != len(b) {
		return fmt.Errorf("got %d bytes, want %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("byte #%d: got 0x%02x, want 0x%02x", i, a[i], b[i])
		}
	}
	return nil
}

func roundtrip(b, ebuf, dbuf []byte) error {
	d, err := Decode(dbuf, Encode(ebuf, b))
	if err != nil {
		return fmt.Errorf("decoding error: %v", err)
	}
	if err := cmp(d, b); err != nil {
		return fmt.Errorf("roundtrip mismatch: %v", err)
	}
	d, err = Decode(dbuf, EncodeBetter(ebuf, b))
	if err != nil {
		return fmt.Errorf("decoding better error: %v", err)
	}
	if err := cmp(d, b); err != nil {
		return fmt.Errorf("roundtrip better mismatch: %v", err)
	}

	// Test concat with some existing data.
	dst := []byte("existing")
	// Add 3 different encodes and a 0 length block.
	concat, err := ConcatBlocks(dst, Encode(nil, b), EncodeBetter(nil, b), []byte{0}, snappy.Encode(nil, b))
	if err != nil {
		return fmt.Errorf("concat error: %v", err)
	}
	if err := cmp(concat[:len(dst)], dst); err != nil {
		return fmt.Errorf("concat existing mismatch: %v", err)
	}
	concat = concat[len(dst):]

	d, _ = Decode(nil, concat)
	want := append(make([]byte, 0, len(b)*3), b...)
	want = append(want, b...)
	want = append(want, b...)

	if err := cmp(d, want); err != nil {
		return fmt.Errorf("roundtrip concat mismatch: %v", err)
	}

	return nil
}

func TestEmpty(t *testing.T) {
	if err := roundtrip(nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSmallCopy(t *testing.T) {
	for _, ebuf := range [][]byte{nil, make([]byte, 20), make([]byte, 64)} {
		for _, dbuf := range [][]byte{nil, make([]byte, 20), make([]byte, 64)} {
			for i := 0; i < 32; i++ {
				s := "aaaa" + strings.Repeat("b", i) + "aaaabbbb"
				if err := roundtrip([]byte(s), ebuf, dbuf); err != nil {
					t.Errorf("len(ebuf)=%d, len(dbuf)=%d, i=%d: %v", len(ebuf), len(dbuf), i, err)
				}
			}
		}
	}
}

func TestSmallRand(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for n := 1; n < 20000; n += 23 {
		b := make([]byte, n)
		for i := range b {
			b[i] = uint8(rng.Intn(256))
		}
		if err := roundtrip(b, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSmallRegular(t *testing.T) {
	for n := 1; n < 20000; n += 23 {
		b := make([]byte, n)
		for i := range b {
			b[i] = uint8(i%10 + 'a')
		}
		if err := roundtrip(b, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSmallRepeat(t *testing.T) {
	for n := 1; n < 20000; n += 23 {
		b := make([]byte, n)
		for i := range b[:n/2] {
			b[i] = uint8(i * 255 / n)
		}
		for i := range b[n/2:] {
			b[i+n/2] = uint8(i%10 + 'a')
		}
		if err := roundtrip(b, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidVarint(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
	}{{
		"invalid varint, final byte has continuation bit set",
		"\xff",
	}, {
		"invalid varint, value overflows uint64",
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\x00",
	}, {
		// https://github.com/google/snappy/blob/master/format_description.txt
		// says that "the stream starts with the uncompressed length [as a
		// varint] (up to a maximum of 2^32 - 1)".
		"valid varint (as uint64), but value overflows uint32",
		"\x80\x80\x80\x80\x10",
	}}

	for _, tc := range testCases {
		input := []byte(tc.input)
		if _, err := DecodedLen(input); err != ErrCorrupt {
			t.Errorf("%s: DecodedLen: got %v, want ErrCorrupt", tc.desc, err)
		}
		if _, err := Decode(nil, input); err != ErrCorrupt {
			t.Errorf("%s: Decode: got %v, want ErrCorrupt", tc.desc, err)
		}
	}
}

func TestDecode(t *testing.T) {
	lit40Bytes := make([]byte, 40)
	for i := range lit40Bytes {
		lit40Bytes[i] = byte(i)
	}
	lit40 := string(lit40Bytes)

	testCases := []struct {
		desc    string
		input   string
		want    string
		wantErr error
	}{{
		`decodedLen=0; valid input`,
		"\x00",
		"",
		nil,
	}, {
		`decodedLen=3; tagLiteral, 0-byte length; length=3; valid input`,
		"\x03" + "\x08\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=2; tagLiteral, 0-byte length; length=3; not enough dst bytes`,
		"\x02" + "\x08\xff\xff\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 0-byte length; length=3; not enough src bytes`,
		"\x03" + "\x08\xff\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=40; tagLiteral, 0-byte length; length=40; valid input`,
		"\x28" + "\x9c" + lit40,
		lit40,
		nil,
	}, {
		`decodedLen=1; tagLiteral, 1-byte length; not enough length bytes`,
		"\x01" + "\xf0",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 1-byte length; length=3; valid input`,
		"\x03" + "\xf0\x02\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 2-byte length; not enough length bytes`,
		"\x01" + "\xf4\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 2-byte length; length=3; valid input`,
		"\x03" + "\xf4\x02\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 3-byte length; not enough length bytes`,
		"\x01" + "\xf8\x00\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 3-byte length; length=3; valid input`,
		"\x03" + "\xf8\x02\x00\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 4-byte length; not enough length bytes`,
		"\x01" + "\xfc\x00\x00\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=1; tagLiteral, 4-byte length; length=3; not enough dst bytes`,
		"\x01" + "\xfc\x02\x00\x00\x00\xff\xff\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagLiteral, 4-byte length; length=3; not enough src bytes`,
		"\x04" + "\xfc\x02\x00\x00\x00\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 4-byte length; length=3; valid input`,
		"\x03" + "\xfc\x02\x00\x00\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=4; tagCopy1, 1 extra length|offset byte; not enough extra bytes`,
		"\x04" + "\x01",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagCopy2, 2 extra length|offset bytes; not enough extra bytes`,
		"\x04" + "\x02\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagCopy4, 4 extra length|offset bytes; not enough extra bytes`,
		"\x04" + "\x03\x00\x00\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagLiteral (4 bytes "abcd"); valid input`,
		"\x04" + "\x0cabcd",
		"abcd",
		nil,
	}, {
		`decodedLen=13; tagLiteral (4 bytes "abcd"); tagCopy1; length=9 offset=4; valid input`,
		"\x0d" + "\x0cabcd" + "\x15\x04",
		"abcdabcdabcda",
		nil,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; valid input`,
		"\x08" + "\x0cabcd" + "\x01\x04",
		"abcdabcd",
		nil,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=2; valid input`,
		"\x08" + "\x0cabcd" + "\x01\x02",
		"abcdcdcd",
		nil,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=1; valid input`,
		"\x08" + "\x0cabcd" + "\x01\x01",
		"abcddddd",
		nil,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=0; repeat offset as first match`,
		"\x08" + "\x0cabcd" + "\x01\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=13; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=1; literal: 'z'; tagCopy1; length=4 offset=0; repeat offset as second match`,
		"\x0d" + "\x0cabcd" + "\x01\x01" + "\x00z" + "\x01\x00",
		"abcdddddzzzzz",
		nil,
	}, {
		`decodedLen=9; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; inconsistent dLen`,
		"\x09" + "\x0cabcd" + "\x01\x04",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=5; offset too large`,
		"\x08" + "\x0cabcd" + "\x01\x05",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=7; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; length too large`,
		"\x07" + "\x0cabcd" + "\x01\x04",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=6; tagLiteral (4 bytes "abcd"); tagCopy2; length=2 offset=3; valid input`,
		"\x06" + "\x0cabcd" + "\x06\x03\x00",
		"abcdbc",
		nil,
	}, {
		`decodedLen=6; tagLiteral (4 bytes "abcd"); tagCopy4; length=2 offset=3; valid input`,
		"\x06" + "\x0cabcd" + "\x07\x03\x00\x00\x00",
		"abcdbc",
		nil,
	}}

	const (
		// notPresentXxx defines a range of byte values [0xa0, 0xc5) that are
		// not present in either the input or the output. It is written to dBuf
		// to check that Decode does not write bytes past the end of
		// dBuf[:dLen].
		//
		// The magic number 37 was chosen because it is prime. A more 'natural'
		// number like 32 might lead to a false negative if, for example, a
		// byte was incorrectly copied 4*8 bytes later.
		notPresentBase = 0xa0
		notPresentLen  = 37
	)

	var dBuf [100]byte
loop:
	for i, tc := range testCases {
		input := []byte(tc.input)
		for _, x := range input {
			if notPresentBase <= x && x < notPresentBase+notPresentLen {
				t.Errorf("#%d (%s): input shouldn't contain %#02x\ninput: % x", i, tc.desc, x, input)
				continue loop
			}
		}

		dLen, n := binary.Uvarint(input)
		if n <= 0 {
			t.Errorf("#%d (%s): invalid varint-encoded dLen", i, tc.desc)
			continue
		}
		if dLen > uint64(len(dBuf)) {
			t.Errorf("#%d (%s): dLen %d is too large", i, tc.desc, dLen)
			continue
		}

		for j := range dBuf {
			dBuf[j] = byte(notPresentBase + j%notPresentLen)
		}
		g, gotErr := Decode(dBuf[:], input)
		if got := string(g); got != tc.want || gotErr != tc.wantErr {
			t.Errorf("#%d (%s):\ngot  %q, %v\nwant %q, %v",
				i, tc.desc, got, gotErr, tc.want, tc.wantErr)
			continue
		}
		for j, x := range dBuf {
			if uint64(j) < dLen {
				continue
			}
			if w := byte(notPresentBase + j%notPresentLen); x != w {
				t.Errorf("#%d (%s): Decode overrun: dBuf[%d] was modified: got %#02x, want %#02x\ndBuf: % x",
					i, tc.desc, j, x, w, dBuf)
				continue loop
			}
		}
	}
}

func TestDecodeCopy4(t *testing.T) {
	dots := strings.Repeat(".", 65536)

	input := strings.Join([]string{
		"\x89\x80\x04",         // decodedLen = 65545.
		"\x0cpqrs",             // 4-byte literal "pqrs".
		"\xf4\xff\xff" + dots,  // 65536-byte literal dots.
		"\x13\x04\x00\x01\x00", // tagCopy4; length=5 offset=65540.
	}, "")

	gotBytes, err := Decode(nil, []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	want := "pqrs" + dots + "pqrs."
	if len(got) != len(want) {
		t.Fatalf("got %d bytes, want %d", len(got), len(want))
	}
	if got != want {
		for i := 0; i < len(got); i++ {
			if g, w := got[i], want[i]; g != w {
				t.Fatalf("byte #%d: got %#02x, want %#02x", i, g, w)
			}
		}
	}
}

// TestDecodeLengthOffset tests decoding an encoding of the form literal +
// copy-length-offset + literal. For example: "abcdefghijkl" + "efghij" + "AB".
func TestDecodeLengthOffset(t *testing.T) {
	const (
		prefix = "abcdefghijklmnopqr"
		suffix = "ABCDEFGHIJKLMNOPQR"

		// notPresentXxx defines a range of byte values [0xa0, 0xc5) that are
		// not present in either the input or the output. It is written to
		// gotBuf to check that Decode does not write bytes past the end of
		// gotBuf[:totalLen].
		//
		// The magic number 37 was chosen because it is prime. A more 'natural'
		// number like 32 might lead to a false negative if, for example, a
		// byte was incorrectly copied 4*8 bytes later.
		notPresentBase = 0xa0
		notPresentLen  = 37
	)
	var gotBuf, wantBuf, inputBuf [128]byte
	for length := 1; length <= 18; length++ {
		for offset := 1; offset <= 18; offset++ {
		loop:
			for suffixLen := 0; suffixLen <= 18; suffixLen++ {
				totalLen := len(prefix) + length + suffixLen

				inputLen := binary.PutUvarint(inputBuf[:], uint64(totalLen))
				inputBuf[inputLen] = tagLiteral + 4*byte(len(prefix)-1)
				inputLen++
				inputLen += copy(inputBuf[inputLen:], prefix)
				inputBuf[inputLen+0] = tagCopy2 + 4*byte(length-1)
				inputBuf[inputLen+1] = byte(offset)
				inputBuf[inputLen+2] = 0x00
				inputLen += 3
				if suffixLen > 0 {
					inputBuf[inputLen] = tagLiteral + 4*byte(suffixLen-1)
					inputLen++
					inputLen += copy(inputBuf[inputLen:], suffix[:suffixLen])
				}
				input := inputBuf[:inputLen]

				for i := range gotBuf {
					gotBuf[i] = byte(notPresentBase + i%notPresentLen)
				}
				got, err := Decode(gotBuf[:], input)
				if err != nil {
					t.Errorf("length=%d, offset=%d; suffixLen=%d: %v", length, offset, suffixLen, err)
					continue
				}

				wantLen := 0
				wantLen += copy(wantBuf[wantLen:], prefix)
				for i := 0; i < length; i++ {
					wantBuf[wantLen] = wantBuf[wantLen-offset]
					wantLen++
				}
				wantLen += copy(wantBuf[wantLen:], suffix[:suffixLen])
				want := wantBuf[:wantLen]

				for _, x := range input {
					if notPresentBase <= x && x < notPresentBase+notPresentLen {
						t.Errorf("length=%d, offset=%d; suffixLen=%d: input shouldn't contain %#02x\ninput: % x",
							length, offset, suffixLen, x, input)
						continue loop
					}
				}
				for i, x := range gotBuf {
					if i < totalLen {
						continue
					}
					if w := byte(notPresentBase + i%notPresentLen); x != w {
						t.Errorf("length=%d, offset=%d; suffixLen=%d; totalLen=%d: "+
							"Decode overrun: gotBuf[%d] was modified: got %#02x, want %#02x\ngotBuf: % x",
							length, offset, suffixLen, totalLen, i, x, w, gotBuf)
						continue loop
					}
				}
				for _, x := range want {
					if notPresentBase <= x && x < notPresentBase+notPresentLen {
						t.Errorf("length=%d, offset=%d; suffixLen=%d: want shouldn't contain %#02x\nwant: % x",
							length, offset, suffixLen, x, want)
						continue loop
					}
				}

				if !bytes.Equal(got, want) {
					t.Errorf("length=%d, offset=%d; suffixLen=%d:\ninput % x\ngot   % x\nwant  % x",
						length, offset, suffixLen, input, got, want)
					continue
				}
			}
		}
	}
}

const (
	goldenText       = "Mark.Twain-Tom.Sawyer.txt"
	goldenCompressed = goldenText + ".rawsnappy"
)

func TestDecodeGoldenInput(t *testing.T) {
	tDir := filepath.FromSlash(*testdataDir)
	src, err := ioutil.ReadFile(filepath.Join(tDir, goldenCompressed))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got, err := Decode(nil, src)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want, err := ioutil.ReadFile(filepath.Join(tDir, goldenText))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := cmp(got, want); err != nil {
		t.Fatal(err)
	}
}

// TestSlowForwardCopyOverrun tests the "expand the pattern" algorithm
// described in decode_amd64.s and its claim of a 10 byte overrun worst case.
func TestSlowForwardCopyOverrun(t *testing.T) {
	const base = 100

	for length := 1; length < 18; length++ {
		for offset := 1; offset < 18; offset++ {
			highWaterMark := base
			d := base
			l := length
			o := offset

			// makeOffsetAtLeast8
			for o < 8 {
				if end := d + 8; highWaterMark < end {
					highWaterMark = end
				}
				l -= o
				d += o
				o += o
			}

			// fixUpSlowForwardCopy
			a := d
			d += l

			// finishSlowForwardCopy
			for l > 0 {
				if end := a + 8; highWaterMark < end {
					highWaterMark = end
				}
				a += 8
				l -= 8
			}

			dWant := base + length
			overrun := highWaterMark - dWant
			if d != dWant || overrun < 0 || 10 < overrun {
				t.Errorf("length=%d, offset=%d: d and overrun: got (%d, %d), want (%d, something in [0, 10])",
					length, offset, d, overrun, dWant)
			}
		}
	}
}

// TestEncoderSkip will test skipping various sizes and block types.
func TestEncoderSkip(t *testing.T) {
	for ti, origLen := range []int{10 << 10, 256 << 10, 2 << 20, 8 << 20} {
		if testing.Short() && ti > 1 {
			break
		}
		t.Run(fmt.Sprint(origLen), func(t *testing.T) {
			src := make([]byte, origLen)
			rng := rand.New(rand.NewSource(1))
			firstHalf, secondHalf := src[:origLen/2], src[origLen/2:]
			bonus := secondHalf[len(secondHalf)-origLen/10:]
			for i := range firstHalf {
				// Incompressible.
				firstHalf[i] = uint8(rng.Intn(256))
			}
			for i := range secondHalf {
				// Easy to compress.
				secondHalf[i] = uint8(i & 32)
			}
			for i := range bonus {
				// Incompressible.
				bonus[i] = uint8(rng.Intn(256))
			}
			var dst bytes.Buffer
			enc := NewWriter(&dst, WriterBlockSize(64<<10))
			_, err := io.Copy(enc, bytes.NewBuffer(src))
			if err != nil {
				t.Fatal(err)
			}
			err = enc.Close()
			if err != nil {
				t.Fatal(err)
			}
			compressed := dst.Bytes()
			dec := NewReader(nil)
			for i := 0; i < len(src); i += len(src)/20 - 17 {
				t.Run(fmt.Sprint("skip-", i), func(t *testing.T) {
					want := src[i:]
					dec.Reset(bytes.NewBuffer(compressed))
					// Read some of it first
					read, err := io.CopyN(ioutil.Discard, dec, int64(len(want)/10))
					if err != nil {
						t.Fatal(err)
					}
					// skip what we just read.
					want = want[read:]
					err = dec.Skip(int64(i))
					if err != nil {
						t.Fatal(err)
					}
					got, err := ioutil.ReadAll(dec)
					if err != nil {
						t.Errorf("Skipping %d returned error: %v", i, err)
						return
					}
					if !bytes.Equal(want, got) {
						t.Log("got  len:", len(got))
						t.Log("want len:", len(want))
						t.Errorf("Skipping %d did not return correct data (content mismatch)", i)
						return
					}
				})
				if testing.Short() && i > 0 {
					return
				}
			}
		})
	}
}

// TestEncodeNoiseThenRepeats encodes input for which the first half is very
// incompressible and the second half is very compressible. The encoded form's
// length should be closer to 50% of the original length than 100%.
func TestEncodeNoiseThenRepeats(t *testing.T) {
	for _, origLen := range []int{256 * 1024, 2048 * 1024} {
		src := make([]byte, origLen)
		rng := rand.New(rand.NewSource(1))
		firstHalf, secondHalf := src[:origLen/2], src[origLen/2:]
		for i := range firstHalf {
			firstHalf[i] = uint8(rng.Intn(256))
		}
		for i := range secondHalf {
			secondHalf[i] = uint8(i >> 8)
		}
		dst := Encode(nil, src)
		if got, want := len(dst), origLen*3/4; got >= want {
			t.Fatalf("origLen=%d: got %d encoded bytes, want less than %d", origLen, got, want)
		}
		t.Log(len(dst))
	}
}

func TestFramingFormat(t *testing.T) {
	// src is comprised of alternating 1e5-sized sequences of random
	// (incompressible) bytes and repeated (compressible) bytes. 1e5 was chosen
	// because it is larger than maxBlockSize (64k).
	src := make([]byte, 1e6)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(rng.Intn(256))
			}
		} else {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(i)
			}
		}
	}

	buf := new(bytes.Buffer)
	bw := NewWriter(buf)
	if _, err := bw.Write(src); err != nil {
		t.Fatalf("Write: encoding: %v", err)
	}
	err := bw.Close()
	if err != nil {
		t.Fatal(err)
	}
	dst, err := ioutil.ReadAll(NewReader(buf))
	if err != nil {
		t.Fatalf("ReadAll: decoding: %v", err)
	}
	if err := cmp(dst, src); err != nil {
		t.Fatal(err)
	}
}

func TestFramingFormatBetter(t *testing.T) {
	// src is comprised of alternating 1e5-sized sequences of random
	// (incompressible) bytes and repeated (compressible) bytes. 1e5 was chosen
	// because it is larger than maxBlockSize (64k).
	src := make([]byte, 1e6)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(rng.Intn(256))
			}
		} else {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(i)
			}
		}
	}

	buf := new(bytes.Buffer)
	bw := NewWriter(buf, WriterBetterCompression())
	if _, err := bw.Write(src); err != nil {
		t.Fatalf("Write: encoding: %v", err)
	}
	err := bw.Close()
	if err != nil {
		t.Fatal(err)
	}
	dst, err := ioutil.ReadAll(NewReader(buf))
	if err != nil {
		t.Fatalf("ReadAll: decoding: %v", err)
	}
	if err := cmp(dst, src); err != nil {
		t.Fatal(err)
	}
}

func TestEmitLiteral(t *testing.T) {
	testCases := []struct {
		length int
		want   string
	}{
		{1, "\x00"},
		{2, "\x04"},
		{59, "\xe8"},
		{60, "\xec"},
		{61, "\xf0\x3c"},
		{62, "\xf0\x3d"},
		{254, "\xf0\xfd"},
		{255, "\xf0\xfe"},
		{256, "\xf0\xff"},
		{257, "\xf4\x00\x01"},
		{65534, "\xf4\xfd\xff"},
		{65535, "\xf4\xfe\xff"},
		{65536, "\xf4\xff\xff"},
	}

	dst := make([]byte, 70000)
	nines := bytes.Repeat([]byte{0x99}, 65536)
	for _, tc := range testCases {
		lit := nines[:tc.length]
		n := emitLiteral(dst, lit)
		if !bytes.HasSuffix(dst[:n], lit) {
			t.Errorf("length=%d: did not end with that many literal bytes", tc.length)
			continue
		}
		got := string(dst[:n-tc.length])
		if got != tc.want {
			t.Errorf("length=%d:\ngot  % x\nwant % x", tc.length, got, tc.want)
			continue
		}
	}
}

func TestEmitCopy(t *testing.T) {
	testCases := []struct {
		offset int
		length int
		want   string
	}{
		{8, 04, "\x01\x08"},
		{8, 11, "\x1d\x08"},
		{8, 12, "\x2e\x08\x00"},
		{8, 13, "\x32\x08\x00"},
		{8, 59, "\xea\x08\x00"},
		{8, 60, "\xee\x08\x00"},
		{8, 61, "\xf2\x08\x00"},
		{8, 62, "\xf6\x08\x00"},
		{8, 63, "\xfa\x08\x00"},
		{8, 64, "\xfe\x08\x00"},
		{8, 65, "\xee\x08\x00\x05\x00"},
		{8, 66, "\xee\x08\x00\x09\x00"},
		{8, 67, "\xee\x08\x00\x0d\x00"},
		{8, 68, "\xee\x08\x00\x11\x00"},
		{8, 69, "\xee\x08\x00\x15\x08"},
		{8, 80, "\xee\x08\x00\x15\x00\x0c"},
		{8, 800, "\xee\x08\x00\x19\x00\xe0\x01"},
		{8, 800000, "\xee\x08\x00\x1d\x00\xc0\x34\x0b"},

		{256, 04, "\x21\x00"},
		{256, 11, "\x3d\x00"},
		{256, 12, "\x2e\x00\x01"},
		{256, 13, "\x32\x00\x01"},
		{256, 59, "\xea\x00\x01"},
		{256, 60, "\xee\x00\x01"},
		{256, 61, "\xf2\x00\x01"},
		{256, 62, "\xf6\x00\x01"},
		{256, 63, "\xfa\x00\x01"},
		{256, 64, "\xfe\x00\x01"},
		{256, 65, "\xee\x00\x01\x05\x00"},
		{256, 66, "\xee\x00\x01\x09\x00"},
		{256, 67, "\xee\x00\x01\x0d\x00"},
		{256, 68, "\xee\x00\x01\x11\x00"},
		{256, 69, "\xee\x00\x01\x35\x00"},
		{256, 80, "\xee\x00\x01\x15\x00\x0c"},
		{256, 800, "\xee\x00\x01\x19\x00\xe0\x01"},
		{256, 80000, "\xee\x00\x01\x1d\x00\x40\x38\x00"},

		{2048, 04, "\x0e\x00\x08"},
		{2048, 11, "\x2a\x00\x08"},
		{2048, 12, "\x2e\x00\x08"},
		{2048, 13, "\x32\x00\x08"},
		{2048, 59, "\xea\x00\x08"},
		{2048, 60, "\xee\x00\x08"},
		{2048, 61, "\xf2\x00\x08"},
		{2048, 62, "\xf6\x00\x08"},
		{2048, 63, "\xfa\x00\x08"},
		{2048, 64, "\xfe\x00\x08"},
		{2048, 65, "\xee\x00\x08\x05\x00"},
		{2048, 66, "\xee\x00\x08\x09\x00"},
		{2048, 67, "\xee\x00\x08\x0d\x00"},
		{2048, 68, "\xee\x00\x08\x11\x00"},
		{2048, 69, "\xee\x00\x08\x15\x00\x01"},
		{2048, 80, "\xee\x00\x08\x15\x00\x0c"},
		{2048, 800, "\xee\x00\x08\x19\x00\xe0\x01"},
		{2048, 80000, "\xee\x00\x08\x1d\x00\x40\x38\x00"},

		{204800, 04, "\x0f\x00\x20\x03\x00"},
		{204800, 65, "\xff\x00\x20\x03\x00\x03\x00\x20\x03\x00"},
		{204800, 69, "\xff\x00\x20\x03\x00\x05\x00"},
		{204800, 800, "\xff\x00\x20\x03\x00\x19\x00\xdc\x01"},
		{204800, 80000, "\xff\x00\x20\x03\x00\x1d\x00\x3c\x38\x00"},
	}

	dst := make([]byte, 1024)
	for _, tc := range testCases {
		n := emitCopy(dst, tc.offset, tc.length)
		got := string(dst[:n])
		if got != tc.want {
			t.Errorf("offset=%d, length=%d:\ngot  % x\nwant % x", tc.offset, tc.length, got, tc.want)
		}
	}
}

func TestNewWriter(t *testing.T) {
	// Test all 32 possible sub-sequences of these 5 input slices.
	//
	// Their lengths sum to 400,000, which is over 6 times the Writer ibuf
	// capacity: 6 * maxBlockSize is 393,216.
	inputs := [][]byte{
		bytes.Repeat([]byte{'a'}, 40000),
		bytes.Repeat([]byte{'b'}, 150000),
		bytes.Repeat([]byte{'c'}, 60000),
		bytes.Repeat([]byte{'d'}, 120000),
		bytes.Repeat([]byte{'e'}, 30000),
	}
loop:
	for i := 0; i < 1<<uint(len(inputs)); i++ {
		var want []byte
		buf := new(bytes.Buffer)
		w := NewWriter(buf)
		for j, input := range inputs {
			if i&(1<<uint(j)) == 0 {
				continue
			}
			if _, err := w.Write(input); err != nil {
				t.Errorf("i=%#02x: j=%d: Write: %v", i, j, err)
				continue loop
			}
			want = append(want, input...)
		}
		if err := w.Close(); err != nil {
			t.Errorf("i=%#02x: Close: %v", i, err)
			continue
		}
		got, err := ioutil.ReadAll(NewReader(buf))
		if err != nil {
			t.Errorf("i=%#02x: ReadAll: %v", i, err)
			continue
		}
		if err := cmp(got, want); err != nil {
			t.Errorf("i=%#02x: %v", i, err)
			continue
		}
	}
}

func TestFlush(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewWriter(buf)
	defer w.Close()
	if _, err := w.Write(bytes.Repeat([]byte{'x'}, 20)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n := buf.Len(); n != 0 {
		t.Fatalf("before Flush: %d bytes were written to the underlying io.Writer, want 0", n)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if n := buf.Len(); n == 0 {
		t.Fatalf("after Flush: %d bytes were written to the underlying io.Writer, want non-0", n)
	}
}

func TestReaderUncompressedDataOK(t *testing.T) {
	r := NewReader(strings.NewReader(magicChunk +
		"\x01\x08\x00\x00" + // Uncompressed chunk, 8 bytes long (including 4 byte checksum).
		"\x68\x10\xe6\xb6" + // Checksum.
		"\x61\x62\x63\x64", // Uncompressed payload: "abcd".
	))
	g, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(g), "abcd"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReaderUncompressedDataNoPayload(t *testing.T) {
	r := NewReader(strings.NewReader(magicChunk +
		"\x01\x04\x00\x00" + // Uncompressed chunk, 4 bytes long.
		"", // No payload; corrupt input.
	))
	if _, err := ioutil.ReadAll(r); err != ErrCorrupt {
		t.Fatalf("got %v, want %v", err, ErrCorrupt)
	}
}

func TestReaderUncompressedDataTooLong(t *testing.T) {
	// The maximum legal chunk length... is 4MB + 4 bytes checksum.
	n := maxBlockSize + checksumSize
	n32 := uint32(n)
	r := NewReader(strings.NewReader(magicChunk +
		// Uncompressed chunk, n bytes long.
		string([]byte{chunkTypeUncompressedData, uint8(n32), uint8(n32 >> 8), uint8(n32 >> 16)}) +
		strings.Repeat("\x00", n),
	))
	// CRC is not set, so we should expect that error.
	if _, err := ioutil.ReadAll(r); err != ErrCRC {
		t.Fatalf("got %v, want %v", err, ErrCRC)
	}

	// test first invalid.
	n++
	n32 = uint32(n)
	r = NewReader(strings.NewReader(magicChunk +
		// Uncompressed chunk, n bytes long.
		string([]byte{chunkTypeUncompressedData, uint8(n32), uint8(n32 >> 8), uint8(n32 >> 16)}) +
		strings.Repeat("\x00", n),
	))
	if _, err := ioutil.ReadAll(r); err != ErrCorrupt {
		t.Fatalf("got %v, want %v", err, ErrCorrupt)
	}
}

func TestReaderReset(t *testing.T) {
	gold := bytes.Repeat([]byte("All that is gold does not glitter,\n"), 10000)
	buf := new(bytes.Buffer)
	w := NewWriter(buf)
	_, err := w.Write(gold)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	encoded, invalid, partial := buf.String(), "invalid", "partial"
	r := NewReader(nil)
	for i, s := range []string{encoded, invalid, partial, encoded, partial, invalid, encoded, encoded} {
		if s == partial {
			r.Reset(strings.NewReader(encoded))
			if _, err := r.Read(make([]byte, 101)); err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
			continue
		}
		r.Reset(strings.NewReader(s))
		got, err := ioutil.ReadAll(r)
		switch s {
		case encoded:
			if err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
			if err := cmp(got, gold); err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
		case invalid:
			if err == nil {
				t.Errorf("#%d: got nil error, want non-nil", i)
				continue
			}
		}
	}
}

func TestWriterReset(t *testing.T) {
	gold := bytes.Repeat([]byte("Not all those who wander are lost;\n"), 10000)
	const n = 20
	var w *Writer
	w = NewWriter(nil)
	defer w.Close()

	var gots, wants [][]byte
	failed := false
	for i := 0; i <= n; i++ {
		buf := new(bytes.Buffer)
		w.Reset(buf)
		want := gold[:len(gold)*i/n]
		if _, err := w.Write(want); err != nil {
			t.Errorf("#%d: Write: %v", i, err)
			failed = true
			continue
		}
		if err := w.Flush(); err != nil {
			t.Errorf("#%d: Flush: %v", i, err)
			failed = true
			continue
			got, err := ioutil.ReadAll(NewReader(buf))
			if err != nil {
				t.Errorf("#%d: ReadAll: %v", i, err)
				failed = true
				continue
			}
			gots = append(gots, got)
			wants = append(wants, want)
		}
		if failed {
			continue
		}
		for i := range gots {
			if err := cmp(gots[i], wants[i]); err != nil {
				t.Errorf("#%d: %v", i, err)
			}
		}
	}
}

func TestWriterResetWithoutFlush(t *testing.T) {
	buf0 := new(bytes.Buffer)
	buf1 := new(bytes.Buffer)
	w := NewWriter(buf0)
	if _, err := w.Write([]byte("xxx")); err != nil {
		t.Fatalf("Write #0: %v", err)
	}
	// Note that we don't Flush the Writer before calling Reset.
	w.Reset(buf1)
	if _, err := w.Write([]byte("yyy")); err != nil {
		t.Fatalf("Write #1: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	got, err := ioutil.ReadAll(NewReader(buf1))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if err := cmp(got, []byte("yyy")); err != nil {
		t.Fatal(err)
	}
}

type writeCounter int

func (c *writeCounter) Write(p []byte) (int, error) {
	*c++
	return len(p), nil
}

// TestNumUnderlyingWrites tests that each Writer flush only makes one or two
// Write calls on its underlying io.Writer, depending on whether or not the
// flushed buffer was compressible.
func TestNumUnderlyingWrites(t *testing.T) {
	testCases := []struct {
		input []byte
		want  int
	}{
		// Magic header + block
		{bytes.Repeat([]byte{'x'}, 100), 2},
		// One block each:
		{bytes.Repeat([]byte{'y'}, 100), 1},
		{[]byte("ABCDEFGHIJKLMNOPQRST"), 1},
	}

	// If we are doing sync writes, we write uncompressed as two writes.
	if runtime.GOMAXPROCS(0) == 1 {
		testCases[2].want++
	}
	var c writeCounter
	w := NewWriter(&c)
	defer w.Close()
	for i, tc := range testCases {
		c = 0
		if _, err := w.Write(tc.input); err != nil {
			t.Errorf("#%d: Write: %v", i, err)
			continue
		}
		if err := w.Flush(); err != nil {
			t.Errorf("#%d: Flush: %v", i, err)
			continue
		}
		if int(c) != tc.want {
			t.Errorf("#%d: got %d underlying writes, want %d", i, c, tc.want)
			continue
		}
	}
}

func testWriterRoundtrip(t *testing.T, src []byte, opts ...WriterOption) {
	var buf bytes.Buffer
	enc := NewWriter(&buf, opts...)
	n, err := enc.Write(src)
	if err != nil {
		t.Error(err)
		return
	}
	if n != len(src) {
		t.Error(io.ErrShortWrite)
		return
	}
	err = enc.Flush()
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("encoded to %d -> %d bytes", len(src), buf.Len())
	dec := NewReader(&buf)
	decoded, err := ioutil.ReadAll(dec)
	if err != nil {
		t.Error(err)
		return
	}
	if len(decoded) != len(src) {
		t.Error("decoded len:", len(decoded), "!=", len(src))
		return
	}
	err = cmp(src, decoded)
	if err != nil {
		t.Error(err)
	}
}

func testBlockRoundtrip(t *testing.T, src []byte) {
	dst := Encode(nil, src)
	t.Logf("encoded to %d -> %d bytes", len(src), len(dst))
	decoded, err := Decode(nil, dst)
	if err != nil {
		t.Error(err)
		return
	}
	if len(decoded) != len(src) {
		t.Error("decoded len:", len(decoded), "!=", len(src))
		return
	}
	err = cmp(src, decoded)
	if err != nil {
		t.Error(err)
	}
}

func testBetterBlockRoundtrip(t *testing.T, src []byte) {
	dst := EncodeBetter(nil, src)
	t.Logf("encoded to %d -> %d bytes", len(src), len(dst))
	decoded, err := Decode(nil, dst)
	if err != nil {
		t.Error(err)
		return
	}
	if len(decoded) != len(src) {
		t.Error("decoded len:", len(decoded), "!=", len(src))
		return
	}
	err = cmp(src, decoded)
	if err != nil {
		t.Error(err)
	}
}
func testSnappyDecode(t *testing.T, src []byte) {
	var buf bytes.Buffer
	enc := snappy.NewBufferedWriter(&buf)
	n, err := enc.Write(src)
	if err != nil {
		t.Error(err)
		return
	}
	if n != len(src) {
		t.Error(io.ErrShortWrite)
		return
	}
	t.Logf("encoded to %d -> %d bytes", len(src), buf.Len())
	dec := NewReader(&buf)
	decoded, err := ioutil.ReadAll(dec)
	if err != nil {
		t.Error(err)
		return
	}
	if len(decoded) != len(src) {
		t.Error("decoded len:", len(decoded), "!=", len(src))
		return
	}
	err = cmp(src, decoded)
	if err != nil {
		t.Error(err)
	}
}

func benchDecode(b *testing.B, src []byte) {
	encoded := Encode(nil, src)
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(src, encoded)
	}
}

func benchDecodeBetter(b *testing.B, src []byte) {
	encoded := EncodeBetter(nil, src)
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(src, encoded)
	}
}

func benchEncode(b *testing.B, src []byte) {
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	dst := make([]byte, MaxEncodedLen(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encode(dst, src)
	}
}

func benchEncodeBetter(b *testing.B, src []byte) {
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	dst := make([]byte, MaxEncodedLen(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeBetter(dst, src)
	}
}

func testOrBenchmark(b testing.TB) string {
	if _, ok := b.(*testing.B); ok {
		return "benchmark"
	}
	return "test"
}

func readFile(b testing.TB, filename string) []byte {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		b.Skipf("skipping %s: %v", testOrBenchmark(b), err)
	}
	if len(src) == 0 {
		b.Fatalf("%s has zero length", filename)
	}
	return src
}

// expand returns a slice of length n containing repeated copies of src.
func expand(src []byte, n int) []byte {
	dst := make([]byte, n)
	for x := dst; len(x) > 0; {
		i := copy(x, src)
		x = x[i:]
	}
	return dst
}

func benchWords(b *testing.B, n int, decode bool) {
	// Note: the file is OS-language dependent so the resulting values are not
	// directly comparable for non-US-English OS installations.
	data := expand(readFile(b, "/usr/share/dict/words"), n)
	if decode {
		benchDecode(b, data)
	} else {
		benchEncode(b, data)
	}
}

func BenchmarkWordsDecode1e1(b *testing.B) { benchWords(b, 1e1, true) }
func BenchmarkWordsDecode1e2(b *testing.B) { benchWords(b, 1e2, true) }
func BenchmarkWordsDecode1e3(b *testing.B) { benchWords(b, 1e3, true) }
func BenchmarkWordsDecode1e4(b *testing.B) { benchWords(b, 1e4, true) }
func BenchmarkWordsDecode1e5(b *testing.B) { benchWords(b, 1e5, true) }
func BenchmarkWordsDecode1e6(b *testing.B) { benchWords(b, 1e6, true) }
func BenchmarkWordsEncode1e1(b *testing.B) { benchWords(b, 1e1, false) }
func BenchmarkWordsEncode1e2(b *testing.B) { benchWords(b, 1e2, false) }
func BenchmarkWordsEncode1e3(b *testing.B) { benchWords(b, 1e3, false) }
func BenchmarkWordsEncode1e4(b *testing.B) { benchWords(b, 1e4, false) }
func BenchmarkWordsEncode1e5(b *testing.B) { benchWords(b, 1e5, false) }
func BenchmarkWordsEncode1e6(b *testing.B) { benchWords(b, 1e6, false) }

func BenchmarkRandomEncode(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = uint8(rng.Intn(256))
	}
	benchEncode(b, data)
}

func BenchmarkRandomEncodeBetter(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = uint8(rng.Intn(256))
	}
	benchEncodeBetter(b, data)
}

// testFiles' values are copied directly from
// https://raw.githubusercontent.com/google/snappy/master/snappy_unittest.cc
// The label field is unused in snappy-go.
var testFiles = []struct {
	label     string
	filename  string
	sizeLimit int
}{
	{"html", "html", 0},
	{"urls", "urls.10K", 0},
	{"jpg", "fireworks.jpeg", 0},
	{"jpg_200", "fireworks.jpeg", 0},
	{"pdf", "paper-100k.pdf", 0},
	{"html4", "html_x_4", 0},
	{"txt1", "alice29.txt", 0},
	{"txt2", "asyoulik.txt", 0},
	{"txt3", "lcet10.txt", 0},
	{"txt4", "plrabn12.txt", 0},
	{"pb", "geo.protodata", 0},
	{"gaviota", "kppkn.gtb", 0},
}

const (
	// The benchmark data files are at this canonical URL.
	benchURL = "https://raw.githubusercontent.com/google/snappy/master/testdata/"
)

func downloadBenchmarkFiles(b testing.TB, basename string) (errRet error) {
	bDir := filepath.FromSlash(*benchdataDir)
	filename := filepath.Join(bDir, basename)
	if stat, err := os.Stat(filename); err == nil && stat.Size() != 0 {
		return nil
	}

	if !*download {
		b.Skipf("test data not found; skipping %s without the -download flag", testOrBenchmark(b))
	}
	// Download the official snappy C++ implementation reference test data
	// files for benchmarking.
	if err := os.MkdirAll(bDir, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create %s: %s", bDir, err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %s", filename, err)
	}
	defer f.Close()
	defer func() {
		if errRet != nil {
			os.Remove(filename)
		}
	}()
	url := benchURL + basename
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %s", url, err)
	}
	defer resp.Body.Close()
	if s := resp.StatusCode; s != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP status code %d (%s)", url, s, http.StatusText(s))
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download %s to %s: %s", url, filename, err)
	}
	return nil
}

func benchFile(b *testing.B, i int, decode bool) {
	if err := downloadBenchmarkFiles(b, testFiles[i].filename); err != nil {
		b.Fatalf("failed to download testdata: %s", err)
	}
	bDir := filepath.FromSlash(*benchdataDir)
	data := readFile(b, filepath.Join(bDir, testFiles[i].filename))

	b.Run("block", func(b *testing.B) {
		if n := testFiles[i].sizeLimit; 0 < n && n < len(data) {
			data = data[:n]
		}
		if decode {
			benchDecode(b, data)
		} else {
			benchEncode(b, data)
		}
	})
	b.Run("block-better", func(b *testing.B) {
		if decode {
			benchDecodeBetter(b, data)
		} else {
			benchEncodeBetter(b, data)
		}
	})

}

func TestRoundtrips(t *testing.T) {
	testFile(t, 0, 10)
	testFile(t, 1, 10)
	testFile(t, 2, 10)
	testFile(t, 3, 10)
	testFile(t, 4, 10)
	testFile(t, 5, 10)
	testFile(t, 6, 10)
	testFile(t, 7, 10)
	testFile(t, 8, 10)
	testFile(t, 9, 10)
	testFile(t, 10, 10)
	testFile(t, 11, 10)
}

func testFile(t *testing.T, i, repeat int) {
	if err := downloadBenchmarkFiles(t, testFiles[i].filename); err != nil {
		t.Skipf("failed to download testdata: %s", err)
	}
	if testing.Short() {
		repeat = 0
	}
	t.Run(fmt.Sprint(i, "-", testFiles[i].filename), func(t *testing.T) {
		bDir := filepath.FromSlash(*benchdataDir)
		data := readFile(t, filepath.Join(bDir, testFiles[i].filename))
		oSize := len(data)
		for i := 0; i < repeat; i++ {
			data = append(data, data[:oSize]...)
		}
		t.Run("s2", func(t *testing.T) {
			testWriterRoundtrip(t, data)
		})
		t.Run("s2-better", func(t *testing.T) {
			testWriterRoundtrip(t, data, WriterBetterCompression())
		})
		t.Run("block", func(t *testing.T) {
			d := data
			testBlockRoundtrip(t, d)
		})
		t.Run("block-better", func(t *testing.T) {
			d := data
			testBetterBlockRoundtrip(t, d)
		})
		t.Run("snappy", func(t *testing.T) {
			testSnappyDecode(t, data)
		})
	})
}

func TestDataRoundtrips(t *testing.T) {
	test := func(t *testing.T, data []byte) {
		t.Run("s2", func(t *testing.T) {
			testWriterRoundtrip(t, data)
		})
		t.Run("s2-better", func(t *testing.T) {
			testWriterRoundtrip(t, data, WriterBetterCompression())
		})
		t.Run("block", func(t *testing.T) {
			d := data
			testBlockRoundtrip(t, d)
		})
		t.Run("block-better", func(t *testing.T) {
			d := data
			testBetterBlockRoundtrip(t, d)
		})
		t.Run("snappy", func(t *testing.T) {
			testSnappyDecode(t, data)
		})
	}
	t.Run("longblock", func(t *testing.T) {
		data := make([]byte, 1<<25)
		test(t, data)
	})
}

// Naming convention is kept similar to what snappy's C++ implementation uses.
func Benchmark_UFlat0(b *testing.B)  { benchFile(b, 0, true) }
func Benchmark_UFlat1(b *testing.B)  { benchFile(b, 1, true) }
func Benchmark_UFlat2(b *testing.B)  { benchFile(b, 2, true) }
func Benchmark_UFlat3(b *testing.B)  { benchFile(b, 3, true) }
func Benchmark_UFlat4(b *testing.B)  { benchFile(b, 4, true) }
func Benchmark_UFlat5(b *testing.B)  { benchFile(b, 5, true) }
func Benchmark_UFlat6(b *testing.B)  { benchFile(b, 6, true) }
func Benchmark_UFlat7(b *testing.B)  { benchFile(b, 7, true) }
func Benchmark_UFlat8(b *testing.B)  { benchFile(b, 8, true) }
func Benchmark_UFlat9(b *testing.B)  { benchFile(b, 9, true) }
func Benchmark_UFlat10(b *testing.B) { benchFile(b, 10, true) }
func Benchmark_UFlat11(b *testing.B) { benchFile(b, 11, true) }
func Benchmark_ZFlat0(b *testing.B)  { benchFile(b, 0, false) }
func Benchmark_ZFlat1(b *testing.B)  { benchFile(b, 1, false) }
func Benchmark_ZFlat2(b *testing.B)  { benchFile(b, 2, false) }
func Benchmark_ZFlat3(b *testing.B)  { benchFile(b, 3, false) }
func Benchmark_ZFlat4(b *testing.B)  { benchFile(b, 4, false) }
func Benchmark_ZFlat5(b *testing.B)  { benchFile(b, 5, false) }
func Benchmark_ZFlat6(b *testing.B)  { benchFile(b, 6, false) }
func Benchmark_ZFlat7(b *testing.B)  { benchFile(b, 7, false) }
func Benchmark_ZFlat8(b *testing.B)  { benchFile(b, 8, false) }
func Benchmark_ZFlat9(b *testing.B)  { benchFile(b, 9, false) }
func Benchmark_ZFlat10(b *testing.B) { benchFile(b, 10, false) }
func Benchmark_ZFlat11(b *testing.B) { benchFile(b, 11, false) }
