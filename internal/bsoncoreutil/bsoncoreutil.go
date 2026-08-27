// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncoreutil

import "unicode/utf8"

// Truncate truncates a given string to at most width bytes without splitting
// a multi-byte UTF-8 character, even when width falls exactly on a character
// boundary.
func Truncate(str string, width int) string {
	if width <= 0 {
		return ""
	}

	if len(str) <= width {
		return str
	}

	// Step back over any trailing continuation bytes (10xxxxxx) to find the
	// start of the last rune within the width-byte prefix.
	start := width
	for start > 0 && str[start-1]&0xC0 == 0x80 {
		start--
	}
	if start == 0 {
		return ""
	}
	start--

	// Decode the rune starting at start from the original (untruncated)
	// string to determine its true byte length. If that rune extends past
	// width, it was cut off and must be dropped entirely.
	_, size := utf8.DecodeRuneInString(str[start:])
	if start+size > width {
		return str[:start]
	}

	return str[:width]
}
