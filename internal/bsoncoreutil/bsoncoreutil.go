// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncoreutil

// Truncate truncates a given string for a certain width
func Truncate(str string, width int) string {
	if width == 0 {
		return ""
	}

	if len(str) <= width {
		return str
	}

	// Truncate the byte slice of the string to the given width.
	newStr := str[:width]

	// Check if the last byte is at the beginning of a multi-byte character.
	// If it is, then remove the last byte.
	if newStr[len(newStr)-1]&0xC0 == 0xC0 {
		return newStr[:len(newStr)-1]
	}

	// Check if the last byte is a multi-byte character
	if newStr[len(newStr)-1]&0xC0 == 0x80 {
		// If it is, step back until you we are at the start of a character
		for i := len(newStr) - 1; i >= 0; i-- {
			if newStr[i]&0xC0 == 0xC0 {
				// Truncate at the end of the character before the character we stepped back to
				return newStr[:i]
			}
		}
	}

	return newStr
}
