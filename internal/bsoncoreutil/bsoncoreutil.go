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

	// If the byte immediately after the cut point starts a new rune (or is
	// ASCII), the cut point does not split a multi-byte character.
	if utf8.RuneStart(str[width]) {
		return str[:width]
	}

	// Otherwise, the rune that the cut point falls inside of was split. Back
	// up over its continuation bytes (10xxxxxx) to find where it starts and
	// drop it entirely, since only part of it fits within width.
	i := width
	for i > 0 && !utf8.RuneStart(str[i]) {
		i--
	}

	return str[:i]
}
