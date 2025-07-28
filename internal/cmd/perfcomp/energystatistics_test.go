// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

func createTestVectors(start1 int, stop1 int, step1 int, start2 int, stop2 int, step2 int) (*mat.Dense, *mat.Dense) {
	xData := []float64{}
	yData := []float64{}

	for i := start1; i < stop1; i += step1 {
		xData = append(xData, float64(i))
	}
	for j := start2; j < stop2; j += step2 {
		yData = append(yData, float64(j))
	}

	x := mat.NewDense(len(xData), 1, xData)
	y := mat.NewDense(len(yData), 1, yData)

	return x, y
}

func TestEnergyStatistics(t *testing.T) {

	t.Run("similar distributions should have small e,t,h values ", func(t *testing.T) {
		x, y := createTestVectors(1, 100, 1, 1, 105, 1)
		e, tstat, h, _ := getEnergyStatistics(x, y)

		del := 1e-3
		// Limit precision of comparison to 3 digits after the decimal.
		assert.InDelta(t, 0.160, e, del) // |0.160 - e| < 0.001
		assert.InDelta(t, 8.136, tstat, del)
		assert.InDelta(t, 0.002, h, del)
	})

	t.Run("different distributions should have large e,t,h values", func(t *testing.T) {
		x, y := createTestVectors(1, 100, 1, 10000, 13000, 14)
		e, tstat, h, _ := getEnergyStatistics(x, y)
		del := 1e-3

		assert.InDelta(t, 21859.691, e, del)
		assert.InDelta(t, 1481794.709, tstat, del)
		assert.InDelta(t, 0.954, h, del)
	})

	t.Run("uni-variate distributions", func(t *testing.T) {
		x, y := createTestVectors(1, 300, 1, 1000, 5000, 10)
		e, tstat, h, _ := getEnergyStatistics(x, y)
		del := 1e-3

		assert.InDelta(t, 4257.009, e, del)
		assert.InDelta(t, 728381.015, tstat, del)
		assert.InDelta(t, 0.748, h, del)
	})

	t.Run("equal distributions should have all 0 values", func(t *testing.T) {
		x := mat.NewDense(10, 1, []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
		y := mat.NewDense(1, 1, []float64{1})

		e, tstat, h, _ := getEnergyStatistics(x, y)

		assert.Equal(t, 0.0, e)
		assert.Equal(t, 0.0, tstat)
		assert.Equal(t, 0.0, h)
	})

	t.Run("energy stats returns errors on malformed input", func(t *testing.T) {
		x := mat.NewDense(2, 2, make([]float64, 4))
		y := mat.NewDense(2, 3, make([]float64, 6))

		_, _, _, err := getEnergyStatistics(x, y)
		assert.NotEqual(t, nil, err)
		assert.ErrorContains(t, err, "both inputs must have the same number of columns")

		x.Reset()
		y = &mat.Dense{}

		_, _, _, err = getEnergyStatistics(x, y)
		assert.NotEqual(t, nil, err)
		assert.ErrorContains(t, err, "inputs cannot be empty")

		x = mat.NewDense(2, 2, make([]float64, 4))
		y = mat.NewDense(3, 2, make([]float64, 6))

		_, _, _, err = getEnergyStatistics(x, y)
		assert.NotEqual(t, nil, err)
		assert.ErrorContains(t, err, "both inputs must be column vectors")
	})

}
