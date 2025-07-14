// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import "testing"

func TestEnergyStatistics(t *testing.T) {
	v1 := []float64{1.000812854,
		0,
		29128,
		635,
		1271,
		58256,
		500406427,
		1.9981990072491742,
		1.998583360145495,
		1.9983911836973345}

	v2 := []float64{1.194869853,
		17334,
		24551,
		629,
		10904148,
		425573368,
		68932,
		2136.1724489294575,
		16173.901792068316,
		15622.55897516013}

	energyStats, _ := GetEnergyStatisticsAndProbabilities(v1, v2, 1000)
	t.Errorf("Expected h-score: %v, but got: %v", 1, energyStats.H)
}
