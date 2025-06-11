// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/binary"
	"math"
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

		vectorData := []byte{
			0x01,                   // int8 vector type
			0x00,                   // padding
			0x01, 0x02, 0x03, 0x04, // int8 values
		}

		bsonData := createBSONWithBinary(vectorData)

		var result struct{ V []int8 }
		err := Unmarshal(bsonData, &result)
		require.NoError(t, err)
		require.Equal(t, []int8{1, 2, 3, 4}, result.V)
	})

	t.Run("float32 vector to []float32", func(t *testing.T) {
		t.Parallel()

		vectorData := make([]byte, 2+4*4) // type + padding + 4 float32s
		vectorData[0] = 0x02              // float32 vector type
		vectorData[1] = 0x00              // padding

		binary.LittleEndian.PutUint32(vectorData[2:], math.Float32bits(1.0))
		binary.LittleEndian.PutUint32(vectorData[6:], math.Float32bits(2.0))
		binary.LittleEndian.PutUint32(vectorData[10:], math.Float32bits(3.0))
		binary.LittleEndian.PutUint32(vectorData[14:], math.Float32bits(4.0))

		bsonData := createBSONWithBinary(vectorData)

		var result struct{ V []float32 }
		err := Unmarshal(bsonData, &result)
		require.NoError(t, err)
		require.InEpsilonSlice(t, []float32{1.0, 2.0, 3.0, 4.0}, result.V, 0.0001)
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
			var result struct{ V []int8 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid vector type: expected int8 vector (0x01)")
		})

		t.Run("to []float32", func(t *testing.T) {
			t.Parallel()
			var result struct{ V []float32 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid vector type: expected float32 vector (0x02)")
		})
	})

	t.Run("invalid binary data", func(t *testing.T) {
		t.Parallel()

		invalidData := []byte{0x01, 0x01, 0x02, 0x03}
		bsonData := createBSONWithBinary(invalidData)

		t.Run("to []int8", func(t *testing.T) {
			t.Parallel()
			var result struct{ V []int8 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid vector: padding byte must be 0")
		})

		t.Run("to []float32", func(t *testing.T) {
			t.Parallel()
			var result struct{ V []float32 }
			err := Unmarshal(bsonData, &result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid vector type: expected float32 vector (0x02)")
		})
	})
}
