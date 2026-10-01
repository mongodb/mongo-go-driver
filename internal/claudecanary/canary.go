// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package claudecanary exercises the claude-review workflow. Do not merge.
package claudecanary

// Max returns the largest element of s, or false if s is empty.
func Max(s []int) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}

	largest := 0
	for _, v := range s {
		if v > largest {
			largest = v
		}
	}

	return largest, true
}
