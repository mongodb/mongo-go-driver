// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

// Class for representing Energy Statistics.
// E - E-statistic
// T - Test statistic
// H - E-coefficient of inhomogeneity
type EnergyStatistics struct {
	E float64
	T float64
	H float64
}

// Class for representing Energy Statistics and permutation test result.
type EnergyStatisticsWithProbabilities struct {
	EnergyStatistics
	EPValue float64
	TPValue float64
	HPValue float64
}

// _convert converts a series into a 2-dimensional Gonum matrix of float64.
// It accepts []float64 or [][]float64. If a []float64 is provided, it is
// converted into a column vector (N x 1 matrix).
func _convert(series interface{}) (*mat.Dense, error) {
	var data []float64
	var rows, cols int

	switch s := series.(type) {
	case []float64:
		data = s
		rows = len(s)
		cols = 1
	case [][]float64:
		if len(s) == 0 {
			return mat.NewDense(0, 0, nil), nil
		}
		rows = len(s)
		cols = len(s[0])
		for _, row := range s {
			if len(row) != cols {
				return nil, errors.New("input [][]float64 has inconsistent row lengths")
			}
			data = append(data, row...)
		}
	case *mat.Dense:
		// If it's already a mat.Dense, handle potential 1D row vector to column vector conversion
		r, c := s.Dims()
		if r == 1 && c > 1 { // If it's a row vector (1 x N), transpose to column vector (N x 1)
			transposed := mat.NewDense(c, 1, nil)
			transposed.Copy(s.T())
			return transposed, nil
		}
		return s, nil
	default:
		return nil, errors.New("series is not the expected type ([]float64, [][]float64, or *mat.Dense)")
	}

	if len(data) == 0 {
		return mat.NewDense(0, 0, nil), nil
	}

	return mat.NewDense(rows, cols, data), nil
}

// _getValidInput returns a valid form of input as a Gonum matrix.
func _getValidInput(series interface{}) (*mat.Dense, error) {
	m, err := _convert(series)
	if err != nil {
		return nil, err
	}
	r, _ := m.Dims()
	if r == 0 {
		return nil, errors.New("distribution cannot be empty")
	}
	return m, nil
}

// _getDistanceMatrix returns the matrix of pairwise Euclidean distances within the series.
// For an m x n series, it returns an m x m matrix where (i,j)th value is the Euclidean
// distance between the i-th and j-th observations (rows) of the series.
func _getDistanceMatrix(series *mat.Dense) (*mat.Dense, error) {
	r, c := series.Dims()
	if r == 0 {
		return mat.NewDense(0, 0, nil), nil
	}

	distMatrix := mat.NewDense(r, r, nil)

	// Calculate Euclidean distance between each pair of rows
	for i := 0; i < r; i++ {
		vecI := mat.NewVecDense(c, nil)
		for k := 0; k < c; k++ {
			vecI.SetVec(k, series.At(i, k))
		}

		for j := i; j < r; j++ {
			vecJ := mat.NewVecDense(c, nil)
			for k := 0; k < c; k++ {
				vecJ.SetVec(k, series.At(j, k))
			}

			// Calculate Euclidean distance: ||vecI - vecJ||_2
			var diff mat.VecDense
			diff.SubVec(vecI, vecJ)
			dist := floats.Norm(diff.RawVector().Data, 2) // Euclidean norm (L2 norm)

			distMatrix.Set(i, j, dist)
			distMatrix.Set(j, i, dist)
		}
	}
	return distMatrix, nil
}

// _calculateStats calculates the E-statistic, Test statistic, and E-coefficient of inhomogeneity.
// It takes the sums of distances within distributions X (x), within Y (y), and between X and Y (xy),
// along with their respective lengths (n, m).
func _calculateStats(x, y, xy float64, n, m int) (e, t, h float64) {
	xyAvg := 0.0
	if n > 0 && m > 0 {
		xyAvg = xy / float64(n*m)
	}

	xAvg := 0.0
	if n > 0 {
		xAvg = x / float64(n*n)
	}

	yAvg := 0.0
	if m > 0 {
		yAvg = y / float64(m*m)
	}

	// E-statistic
	e = 2*xyAvg - xAvg - yAvg

	// Test statistic
	t = 0.0
	if n+m > 0 {
		t = (float64(n*m) / float64(n+m)) * e
	}

	// E-coefficient of inhomogeneity
	h = 0.0
	if xyAvg > 0 {
		h = e / (2 * xyAvg)
	}
	return e, t, h
}

// _calculateTStats finds t-statistic values given a distance matrix.
// It iteratively calculates the test statistic for all possible partition points (tau).
func _calculateTStats(distanceMatrix *mat.Dense) ([]float64, error) {
	N, _ := distanceMatrix.Dims()
	if N == 0 {
		return []float64{}, nil
	}

	statistics := make([]float64, N)

	initialYSum := 0.0
	for r := 0; r < N; r++ {
		for c := r; c < N; c++ {
			initialYSum += distanceMatrix.At(r, c)
		}
	}

	xy := 0.0
	x := 0.0
	y := initialYSum

	for tau := 0; tau < N; tau++ {
		_, t, _ := _calculateStats(x, y, xy, tau, N-tau)
		statistics[tau] = t

		// columnDelta: sum |Xi - X_tau| for i < tau (distances from elements in X to the new element at tau)
		columnDelta := 0.0
		for rIdx := 0; rIdx < tau; rIdx++ {
			columnDelta += distanceMatrix.At(rIdx, tau)
		}

		// rowDelta: sum |X_tau - Yj| for tau <= j (distances from the new element at tau to elements in Y)
		rowDelta := 0.0
		for cIdx := tau; cIdx < N; cIdx++ {
			rowDelta += distanceMatrix.At(tau, cIdx)
		}

		xy = xy - columnDelta + rowDelta // Distances between X and Y
		x = x + columnDelta              // Distances within X
		y = y - rowDelta                 // Distances within Y
	}

	return statistics, nil
}

// _getNextSignificantChangePoint calculates the next significant change point using a permutation test.
// It searches for change points within windows defined by existing change points.
func _getNextSignificantChangePoint(
	distances *mat.Dense,
	changePoints []int,
	memo map[[2]int]struct {
		idx int
		val float64
	},
	pvalue float64,
	permutations int,
) (int, error) {
	N, _ := distances.Dims()
	if N == 0 {
		return -1, nil
	}

	windows := []int{0}
	windows = append(windows, changePoints...)
	windows = append(windows, N)
	sort.Ints(windows)

	type candidate struct {
		idx int
		val float64
	}
	var candidates []candidate

	for i := 0; i < len(windows)-1; i++ {
		a, b := windows[i], windows[i+1]
		boundsKey := [2]int{a, b}

		if val, ok := memo[boundsKey]; ok {
			candidates = append(candidates, candidate{idx: val.idx, val: val.val})
		} else {
			windowDistances := distances.Slice(a, b, a, b).(*mat.Dense)
			stats, err := _calculateTStats(windowDistances)
			if err != nil {
				return -1, fmt.Errorf("error calculating t-stats for window [%d:%d]: %w", a, b, err)
			}

			if len(stats) == 0 {
				continue
			}

			// Find the index of the maximum T-statistic within the window
			idx := 0
			maxStat := stats[0]
			for k, s := range stats {
				if s > maxStat {
					maxStat = s
					idx = k
				}
			}
			newCandidate := candidate{idx: idx + a, val: maxStat}
			candidates = append(candidates, newCandidate)
			memo[boundsKey] = struct {
				idx int
				val float64
			}{idx: newCandidate.idx, val: newCandidate.val}
		}
	}

	if len(candidates) == 0 {
		return -1, nil
	}

	// Find the overall best candidate among all windows
	bestCandidate := candidates[0]
	for _, c := range candidates {
		if c.val > bestCandidate.val {
			bestCandidate = c
		}
	}

	betterNum := 0
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	for p := 0; p < permutations; p++ {
		permuteT := make([]float64, 0, len(windows)-1)
		for i := 0; i < len(windows)-1; i++ {
			a, b := windows[i], windows[i+1]
			windowSize := b - a
			if windowSize == 0 {
				continue
			}

			rowIndices := make([]int, windowSize)
			for k := 0; k < windowSize; k++ {
				rowIndices[k] = k + a
			}
			r.Shuffle(len(rowIndices), func(i, j int) {
				rowIndices[i], rowIndices[j] = rowIndices[j], rowIndices[i]
			})

			shuffledDistances := mat.NewDense(windowSize, windowSize, nil)
			for row := 0; row < windowSize; row++ {
				for col := 0; col < windowSize; col++ {
					shuffledDistances.Set(row, col, distances.At(rowIndices[row], rowIndices[col]))
				}
			}

			stats, err := _calculateTStats(shuffledDistances)
			if err != nil {
				return -1, fmt.Errorf("error calculating t-stats for shuffled window [%d:%d]: %w", a, b, err)
			}

			if len(stats) == 0 {
				continue
			}

			maxPermuteStat := stats[0]
			for _, s := range stats {
				if s > maxPermuteStat {
					maxPermuteStat = s
				}
			}
			permuteT = append(permuteT, maxPermuteStat)
		}

		if len(permuteT) == 0 {
			continue
		}

		bestPermute := permuteT[0]
		for _, val := range permuteT {
			if val > bestPermute {
				bestPermute = val
			}
		}

		if bestPermute >= bestCandidate.val {
			betterNum++
		}
	}

	probability := float64(betterNum) / float64(permutations+1)
	if probability <= pvalue {
		return bestCandidate.idx, nil
	}
	return -1, nil
}

// _getEnergyStatisticsFromDistanceMatrix returns energy statistics from a combined distance matrix.
// It partitions the combined distance matrix into within-X, within-Y, and between-XY distances
// based on the provided lengths n (for X) and m (for Y).
func _getEnergyStatisticsFromDistanceMatrix(distanceMatrix *mat.Dense, n, m int) (*EnergyStatistics, error) {
	lenDistanceMatrix, _ := distanceMatrix.Dims()

	if lenDistanceMatrix == 0 {
		return &EnergyStatistics{E: 0, T: 0, H: 0}, nil
	}

	// Sum distances within X (top-left sub-matrix)
	xSum := 0.0
	if n > 0 {
		for r := 0; r < n; r++ {
			for c := 0; c < n; c++ {
				xSum += distanceMatrix.At(r, c)
			}
		}
	}

	// Sum distances within Y (bottom-right sub-matrix)
	ySum := 0.0
	if m > 0 {
		for r := n; r < lenDistanceMatrix; r++ {
			for c := n; c < lenDistanceMatrix; c++ {
				ySum += distanceMatrix.At(r, c)
			}
		}
	}

	// Sum distances between X and Y (bottom-left sub-matrix, which is equivalent to top-right due to symmetry)
	xySum := 0.0
	if n > 0 && m > 0 {
		for r := n; r < lenDistanceMatrix; r++ {
			for c := 0; c < n; c++ {
				xySum += distanceMatrix.At(r, c)
			}
		}
	}

	e, t, h := _calculateStats(xSum, ySum, xySum, n, m)
	return &EnergyStatistics{E: e, T: t, H: h}, nil
}

// EDivisive calculates the change points in the series using the e-divisive algorithm.
// It iteratively finds significant change points until no more are found based on the p-value.
func EDivisive(series interface{}, pvalue float64, permutations int) ([]int, error) {
	seriesMat, err := _getValidInput(series)
	if err != nil {
		return nil, err
	}

	distances, err := _getDistanceMatrix(seriesMat)
	if err != nil {
		return nil, err
	}

	changePoints := []int{}
	memo := make(map[[2]int]struct {
		idx int
		val float64
	}) // Cache for _getNextSignificantChangePoint

	for {
		significantChangePoint, err := _getNextSignificantChangePoint(
			distances, changePoints, memo, pvalue, permutations,
		)
		if err != nil {
			return nil, err
		}
		if significantChangePoint == -1 {
			break
		}
		changePoints = append(changePoints, significantChangePoint)
	}

	sort.Ints(changePoints)
	return changePoints, nil
}

// GetEnergyStatistics calculates energy statistics of distributions x and y.
// It combines x and y, calculates the full distance matrix, and then derives
// the E-statistic, Test statistic, and E-coefficient of inhomogeneity.
func GetEnergyStatistics(x, y interface{}) (*EnergyStatistics, error) {
	xMat, err := _getValidInput(x)
	if err != nil {
		return nil, err
	}
	yMat, err := _getValidInput(y)
	if err != nil {
		return nil, err
	}

	n, _ := xMat.Dims()
	m, _ := yMat.Dims()

	_, xCols := xMat.Dims()
	_, yCols := yMat.Dims()
	if xCols != yCols {
		return nil, errors.New("distributions x and y must have the same number of variables (columns)")
	}

	combinedRows := n + m
	combinedData := make([]float64, combinedRows*xCols)

	for r := 0; r < n; r++ {
		for c := 0; c < xCols; c++ {
			combinedData[r*xCols+c] = xMat.At(r, c)
		}
	}
	for r := 0; r < m; r++ {
		for c := 0; c < yCols; c++ {
			combinedData[(n+r)*yCols+c] = yMat.At(r, c)
		}
	}
	combinedMat := mat.NewDense(combinedRows, xCols, combinedData)

	distances, err := _getDistanceMatrix(combinedMat)
	if err != nil {
		return nil, err
	}

	return _getEnergyStatisticsFromDistanceMatrix(distances, n, m)
}

// GetEnergyStatisticsAndProbabilities returns energy statistics and the corresponding
// permutation test results (p-values) for distributions x and y.
func GetEnergyStatisticsAndProbabilities(x, y interface{}, permutations int) (*EnergyStatisticsWithProbabilities, error) {
	xMat, err := _getValidInput(x)
	if err != nil {
		return nil, err
	}
	yMat, err := _getValidInput(y)
	if err != nil {
		return nil, err
	}

	n, _ := xMat.Dims()
	m, _ := yMat.Dims()

	_, xCols := xMat.Dims()
	_, yCols := yMat.Dims()
	if xCols != yCols {
		return nil, errors.New("distributions x and y must have the same number of variables (columns)")
	}

	combinedRows := n + m
	combinedData := make([]float64, combinedRows*xCols)

	for r := 0; r < n; r++ {
		for c := 0; c < xCols; c++ {
			combinedData[r*xCols+c] = xMat.At(r, c)
		}
	}
	for r := 0; r < m; r++ {
		for c := 0; c < yCols; c++ {
			combinedData[(n+r)*yCols+c] = yMat.At(r, c)
		}
	}
	combinedMat := mat.NewDense(combinedRows, xCols, combinedData)

	distancesBetweenAll, err := _getDistanceMatrix(combinedMat)
	if err != nil {
		return nil, err
	}

	lenCombined, _ := distancesBetweenAll.Dims()

	// Counters for permutation test
	countE := 0
	countT := 0
	countH := 0

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	rowIndices := make([]int, lenCombined)
	for i := 0; i < lenCombined; i++ {
		rowIndices[i] = i
	}

	// Calculate initial energy statistics for the original (unshuffled) data
	energyStatistics, err := _getEnergyStatisticsFromDistanceMatrix(distancesBetweenAll, n, m)
	if err != nil {
		return nil, err
	}

	// Perform permutation test
	for p := 0; p < permutations; p++ {
		r.Shuffle(len(rowIndices), func(i, j int) {
			rowIndices[i], rowIndices[j] = rowIndices[j], rowIndices[i]
		})

		// Create a new shuffled distance matrix by reordering rows/columns of the original
		// distance matrix according to the shuffled rowIndices. This simulates shuffling
		// the original combined data and then calculating distances.
		shuffledDistances := mat.NewDense(lenCombined, lenCombined, nil)
		for row := 0; row < lenCombined; row++ {
			for col := 0; col < lenCombined; col++ {
				shuffledDistances.Set(row, col, distancesBetweenAll.At(rowIndices[row], rowIndices[col]))
			}
		}

		shuffledEnergyStatistics, err := _getEnergyStatisticsFromDistanceMatrix(shuffledDistances, n, m)
		if err != nil {
			return nil, err
		}

		// Compare shuffled statistics with original statistics
		if shuffledEnergyStatistics.E >= energyStatistics.E {
			countE++
		}
		if shuffledEnergyStatistics.T >= energyStatistics.T {
			countT++
		}
		if shuffledEnergyStatistics.H >= energyStatistics.H {
			countH++
		}
	}

	// Calculate p-values
	total := float64(permutations + 1)
	return &EnergyStatisticsWithProbabilities{
		EnergyStatistics: *energyStatistics,
		EPValue:          float64(countE) / total,
		TPValue:          float64(countT) / total,
		HPValue:          float64(countH) / total,
	}, nil
}
