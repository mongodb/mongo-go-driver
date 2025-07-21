// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Given two matrices, this function returns
// (e, t, h) = (E-statistic, test statistic, e-coefficient of inhomogeneity)
func GetEnergyStatistics(x, y *mat.Dense) (float64, float64, float64) {
	n, _ := x.Dims()
	m, _ := y.Dims()
	nf := float64(n)
	mf := float64(m)

	var A float64 // E|X-Y|
	if nf > 0 && mf > 0 {
		A = getDistance(x, y) / (nf * mf)
	} else {
		A = 0
	}
	var B float64 // E|X-X'|
	if nf > 0 {
		B = getDistance(x, x) / (nf * nf)
	} else {
		B = 0
	}
	var C float64 // E|Y-Y'|
	if mf > 0 {
		C = getDistance(y, y) / (mf * mf)
	} else {
		C = 0
	}

	E := 2*A - B - C // D^2(F_x, F_y)
	T := ((nf * mf) / (nf + mf)) * E
	var H float64
	if A > 0 {
		H = E / (2 * A)
	} else {
		H = 0
	}
	return E, T, H
}

// Given two vectors (expected 1 col),
// this function returns the sum of distances between each pair.
func getDistance(x, y *mat.Dense) float64 {
	xrows, _ := x.Dims()
	yrows, _ := y.Dims()

	var sum float64

	for i := 0; i < xrows; i++ {
		for j := 0; j < yrows; j++ {
			sum += math.Sqrt(math.Pow((x.At(i, 0) - y.At(j, 0)), 2))
		}
	}
	return sum
}

func GetZScore(x, u, o float64) float64 {
	if u == 0 || o == 0 {
		return 0
	}
	return (x - u) / o
}
