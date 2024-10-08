// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"container/list"
	"math"
	"time"
)

func standardDeviationList(l *list.List) float64 {
	if l.Len() == 0 {
		return 0
	}

	var mean, variance float64
	count := 0.0

	for el := l.Front(); el != nil; el = el.Next() {
		count++
		sample := float64(el.Value.(time.Duration))

		delta := sample - mean
		mean += delta / count
		variance += delta * (sample - mean)
	}

	return math.Sqrt(variance / count)
}
