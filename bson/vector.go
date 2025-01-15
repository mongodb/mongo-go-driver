// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// These constants are vector data types.
const (
	Int8Vector      = 0x03
	Float32Vector   = 0x27
	PackedBitVector = 0x10
)

// These are vector conversion errors.
var (
	ErrNotVector              = errors.New("not a vector")
	ErrInsufficientVectorData = errors.New("insufficient data")
	ErrNonZeroVectorPadding   = errors.New("padding must be 0")
	ErrVectorPaddingTooLarge  = errors.New("padding larger than 7")
)

// Vector represents a densely packed array of numbers.
type Vector[T int8 | float32] struct {
	Data []T
}

// BitVector represents a binary quantized (PACKED_BIT) vector of 0s and 1s
// (bits). The Padding prescribes the number of bits to ignore in the final byte
// of the Data. It should be 0 for an empty Data and always less than 8.
type BitVector struct {
	Padding uint8
	Data    []byte
}

func newInt8Vector(b []byte) (Vector[int8], error) {
	var v Vector[int8]
	if len(b) == 0 {
		return v, ErrInsufficientVectorData
	}
	if padding := b[0]; padding > 0 {
		return v, ErrNonZeroVectorPadding
	}
	s := make([]int8, 0, len(b)-1)
	for i := 1; i < len(b); i++ {
		s = append(s, int8(b[i]))
	}
	v.Data = s
	return v, nil
}

func newFloat32Vector(b []byte) (Vector[float32], error) {
	var v Vector[float32]
	if len(b) == 0 {
		return v, ErrInsufficientVectorData
	}
	if padding := b[0]; padding > 0 {
		return v, ErrNonZeroVectorPadding
	}
	l := (len(b) - 1) / 4
	if l*4 != len(b)-1 {
		return v, ErrInsufficientVectorData
	}
	s := make([]float32, 0, l)
	for i := 1; i < len(b); i += 4 {
		s = append(s, math.Float32frombits(binary.LittleEndian.Uint32(b[i:i+4])))
	}
	v.Data = s
	return v, nil
}

func newBitVector(b []byte) (BitVector, error) {
	var v BitVector
	if len(b) == 0 {
		return v, ErrInsufficientVectorData
	}
	padding := b[0]
	if padding > 7 {
		return v, ErrVectorPaddingTooLarge
	}
	if padding > 0 && len(b) == 1 {
		return v, ErrNonZeroVectorPadding
	}
	v.Padding = padding
	v.Data = b[1:]
	return v, nil
}

// NewVectorFromBinary unpacks a BSON Binary into a Vector.
func NewVectorFromBinary(b Binary) (interface{}, error) {
	if b.Subtype != TypeBinaryVector {
		return nil, ErrNotVector
	}
	if len(b.Data) < 2 {
		return nil, ErrInsufficientVectorData
	}
	switch t := b.Data[0]; t {
	case Int8Vector:
		return newInt8Vector(b.Data[1:])
	case Float32Vector:
		return newFloat32Vector(b.Data[1:])
	case PackedBitVector:
		return newBitVector(b.Data[1:])
	default:
		return nil, fmt.Errorf("invalid Vector data type: %d", t)
	}
}

func binaryFromFloat32Vector(v Vector[float32]) (Binary, error) {
	data := make([]byte, 2, len(v.Data)*4+2)
	copy(data, []byte{Float32Vector, 0})
	var a [4]byte
	for _, e := range v.Data {
		binary.LittleEndian.PutUint32(a[:], math.Float32bits(e))
		data = append(data, a[:]...)
	}

	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}, nil
}

func binaryFromInt8Vector(v Vector[int8]) (Binary, error) {
	data := make([]byte, 2, len(v.Data)+2)
	copy(data, []byte{Int8Vector, 0})
	for _, e := range v.Data {
		data = append(data, byte(e))
	}

	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}, nil
}

func binaryFromBitVector(v BitVector) (Binary, error) {
	var b Binary
	if v.Padding > 7 {
		return b, ErrVectorPaddingTooLarge
	}
	if v.Padding > 0 && len(v.Data) == 0 {
		return b, ErrNonZeroVectorPadding
	}
	data := []byte{PackedBitVector, v.Padding}
	data = append(data, v.Data...)
	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}, nil
}

// NewBinaryFromVector converts a Vector into a BSON Binary.
func NewBinaryFromVector[T BitVector | Vector[int8] | Vector[float32]](v T) (Binary, error) {
	switch a := any(v).(type) {
	case Vector[int8]:
		return binaryFromInt8Vector(a)
	case Vector[float32]:
		return binaryFromFloat32Vector(a)
	case BitVector:
		return binaryFromBitVector(a)
	default:
		return Binary{}, fmt.Errorf("unsupported type %T", v)
	}
}
