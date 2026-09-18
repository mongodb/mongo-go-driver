// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncoreutil

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      string
		width    int
		expected string
	}{
		{
			name:     "empty",
			arg:      "",
			width:    0,
			expected: "",
		},
		{
			name:     "short",
			arg:      "foo",
			width:    1000,
			expected: "foo",
		},
		{
			name:     "long",
			arg:      "foo bar baz",
			width:    9,
			expected: "foo bar b",
		},
		{
			name:     "multi-byte",
			arg:      "你好",
			width:    4,
			expected: "你",
		},
		{
			name:     "multi-byte exact boundary",
			arg:      "a©bc",
			width:    3,
			expected: "a©",
		},
		{
			name:     "multi-byte exact boundary before trailing ascii",
			arg:      "你好!",
			width:    6,
			expected: "你好",
		},
		{
			name:     "negative width",
			arg:      "foo",
			width:    -1,
			expected: "",
		},
		{
			name:     "four-byte rune exact boundary",
			arg:      "ab𤭢cd",
			width:    6,
			expected: "ab𤭢",
		},
		{
			name:     "four-byte rune split",
			arg:      "ab𤭢cd",
			width:    4,
			expected: "ab",
		},
		{
			name:     "invalid utf-8 clean cut",
			arg:      "a\xffbc",
			width:    2,
			expected: "a\xff",
		},
		{
			name:     "invalid utf-8 with continuation-like bytes",
			arg:      "a\xff\x80\x80bc",
			width:    3,
			expected: "a",
		},
		{
			name:     "invalid utf-8 all continuation-like bytes",
			arg:      "\x80\x80\x80\x80",
			width:    2,
			expected: "",
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			actual := Truncate(tcase.arg, tcase.width)
			assert.Equal(t, tcase.expected, actual)
		})
	}
}

// truncateSink prevents the compiler from eliminating the Truncate call in
// BenchmarkTruncate.
var truncateSink string

// benchWidth is the width Truncate is called with on the logging path. It
// matches logger.DefaultMaxDocumentLength, which is not referenced directly to
// avoid an import cycle.
const benchWidth = 1000

// BenchmarkTruncate covers the four distinct paths through Truncate. Widths are
// chosen so that each case provably lands on the intended path; see the byte
// arithmetic on each case below. "界" (U+754C) encodes to 3 bytes and "𤭢"
// (U+24B62) to 4 bytes.
func BenchmarkTruncate(b *testing.B) {
	var (
		ascii     = strings.Repeat("a", 4000) // 4000 bytes
		threeByte = strings.Repeat("界", 500)  // 1500 bytes
		fourByte  = strings.Repeat("𤭢", 400)  // 1600 bytes
		short     = strings.Repeat("a", 500)  // 500 bytes
	)

	for _, bc := range []struct {
		name  string
		str   string
		width int
	}{
		// Width lands mid-string on an ASCII byte, so utf8.RuneStart reports a
		// clean cut immediately and the backward scan never runs. This is the
		// common logging case: truncating extended JSON.
		{"ascii_mid_string", ascii, benchWidth},

		// 999 == 333*3, so width lands exactly on the end of a 3-byte rune and
		// the full prefix is kept. This is the path PR #2564 corrected. Scans
		// back over 2 continuation bytes.
		{"exact_rune_boundary", threeByte, 999},

		// 999 == 249*4+3, so width lands 3 bytes into a 4-byte rune, which is
		// dropped entirely. Scans back over 2 continuation bytes.
		{"inside_rune", fourByte, 999},

		// len(str) <= width, so Truncate returns before doing any UTF-8 work.
		{"no_truncation", short, benchWidth},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				truncateSink = Truncate(bc.str, bc.width)
			}
		})
	}
}
