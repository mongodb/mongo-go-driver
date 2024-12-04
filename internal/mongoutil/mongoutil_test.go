// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func BenchmarkNewOptions(b *testing.B) {
	b.Run("reflect.ValueOf is always called", func(b *testing.B) {
		opts := make([]options.Lister[options.FindOptions], b.N)

		// Create a huge string to see if we can force reflect.ValueOf to use heap
		// over stack.
		size := 16 * 1024 * 1024
		str := strings.Repeat("a", size)

		for i := 0; i < b.N; i++ {
			opts[i] = options.Find().SetComment(str).SetHint("y").SetMin(1).SetMax(2)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = NewOptions[options.FindOptions](opts...)
		}
	})
}
