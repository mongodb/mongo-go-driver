// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper function to create a BSON document with a vector binary field (subtype 0x09)
func createBSONWithBinary(data []byte) []byte {
	// Document format: {"v": BinData(subtype, data)}
	buf := make([]byte, 0, 32+len(data))

	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // Length placeholder
	buf = append(buf, 0x05)                   // Binary type
	buf = append(buf, 'v', 0x00)              // Field name "v"

	buf = append(buf,
		byte(len(data)),  // Length of binary data
		0x00, 0x00, 0x00, // 4-byte length (little endian)
		0x09, // Binary subtype for Vector
	)
	buf = append(buf, data...)
	buf = append(buf, 0x00)

	docLen := len(buf)
	buf[0] = byte(docLen)
	buf[1] = byte(docLen >> 8)
	buf[2] = byte(docLen >> 16)
	buf[3] = byte(docLen >> 24)

	return buf
}

func TestVectorBackwardCompatibility(t *testing.T) {
	t.Parallel()

	t.Run("unmarshal to Vector field", func(t *testing.T) {
		t.Parallel()

		vectorData := []byte{
			0x03,                   // int8 vector type (0x03 is Int8Vector)
			0x00,                   // padding
			0x01, 0x02, 0x03, 0x04, // int8 values
		}

		doc := createBSONWithBinary(vectorData)

		var result struct {
			V Vector
		}
		err := Unmarshal(doc, &result)
		require.NoError(t, err)

		require.Equal(t, Int8Vector, result.V.Type())
		int8Data, ok := result.V.Int8OK()
		require.True(t, ok, "expected int8 vector")
		require.Equal(t, []int8{1, 2, 3, 4}, int8Data)
	})
}

func TestUnmarshalVectorToSlices(t *testing.T) {
	t.Parallel()

	t.Run("int8 vector to []int8", func(t *testing.T) {
		t.Parallel()

		doc := D{{"v", NewVector([]int8{-2, 1, 2, 3, 4})}}
		bsonData, err := Marshal(doc)
		require.NoError(t, err)
		var result struct{ V []int8 }
		err = Unmarshal(bsonData, &result)
		require.NoError(t, err)
		require.Equal(t, []int8{-2, 1, 2, 3, 4}, result.V)
	})

	t.Run("float32 vector to []float32", func(t *testing.T) {
		t.Parallel()

		doc := D{{"v", NewVector([]float32{1.1, 2.2, 3.3, 4.4})}}
		bsonData, err := Marshal(doc)
		require.NoError(t, err)
		var result struct{ V []float32 }
		err = Unmarshal(bsonData, &result)
		require.NoError(t, err)
		require.InDeltaSlice(t, []float32{1.1, 2.2, 3.3, 4.4}, result.V, 0.001)
	})

	t.Run("invalid vector type to slice", func(t *testing.T) {
		t.Parallel()

		vectorData := []byte{
			0x10,       // packed bit vector type (unsupported for direct unmarshaling)
			0x00,       // padding
			0x01, 0x02, // some data
		}
		bsonData := createBSONWithBinary(vectorData)

		t.Run("to []int8", func(t *testing.T) {
			t.Parallel()

			vectorData := []byte{0x10, 0x00} // Invalid vector type
			bsonData := createBSONWithBinary(vectorData)

			var result struct{ V []int8 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprintf("invalid vector type: expected int8 vector (0x%02x)", Int8Vector))
		})

		t.Run("to []float32", func(t *testing.T) {
			t.Parallel()
			var result struct{ V []float32 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprintf("invalid vector type: expected float32 vector (0x%02x)", Float32Vector))
		})
	})

	t.Run("invalid binary data", func(t *testing.T) {
		t.Parallel()

		vectorData := []byte{0x01, 0x00, 0x01, 0x02, 0x03, 0x04}
		bsonData := createBSONWithBinary(vectorData)

		t.Run("to []int8", func(t *testing.T) {
			t.Parallel()

			var result struct{ V []int8 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprintf("invalid vector type: expected int8 vector (0x%02x)", Int8Vector))
		})

		t.Run("to []float32", func(t *testing.T) {
			t.Parallel()

			vectorData := []byte{0x01, 0x00}
			bsonData := createBSONWithBinary(vectorData)

			var result struct{ V []float32 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprintf("invalid vector type: expected float32 vector (0x%02x)", Float32Vector))
		})
	})
}
