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

// BSON binary vector types as described in https://bsonspec.org/spec.html.
const (
	Int8Vector      byte = 0x03
	Float32Vector   byte = 0x27
	PackedBitVector byte = 0x10
)

// These are vector conversion errors.
var (
	errInsufficientVectorData = errors.New("insufficient data")
	errNonZeroVectorPadding   = errors.New("padding must be 0")
	errVectorPaddingTooLarge  = errors.New("padding cannot be larger than 7")
)

type vectorTypeError struct {
	Method string
	Type   byte
}

// Error implements the error interface.
func (vte vectorTypeError) Error() string {
	t := "invalid"
	switch vte.Type {
	case Int8Vector:
		t = "int8"
	case Float32Vector:
		t = "float32"
	case PackedBitVector:
		t = "packed bit"
	}
	return fmt.Sprintf("cannot call %s, on a type %s vector", vte.Method, t)
}

// Vector represents a densely packed array of numbers / bits.
type Vector struct {
	dType       byte
	int8Data    []int8
	float32Data []float32
	bitData     []byte
	bitPadding  uint8
}

// Type returns the vector type.
func (v Vector) Type() byte {
	return v.dType
}

// Int8 returns the int8 slice hold by the vector.
// It panics if v is not an int8 vector.
func (v Vector) Int8() []int8 {
	d, ok := v.Int8OK()
	if !ok {
		panic(vectorTypeError{"bson.Vector.Int8", v.dType})
	}
	return d
}

// Int8OK is the same as Int8, but returns a boolean instead of panicking.
func (v Vector) Int8OK() ([]int8, bool) {
	if v.dType != Int8Vector {
		return nil, false
	}
	return v.int8Data, true
}

// Float32 returns the float32 slice hold by the vector.
// It panics if v is not a float32 vector.
func (v Vector) Float32() []float32 {
	d, ok := v.Float32OK()
	if !ok {
		panic(vectorTypeError{"bson.Vector.Float32", v.dType})
	}
	return d
}

// Float32OK is the same as Float32, but returns a boolean instead of panicking.
func (v Vector) Float32OK() ([]float32, bool) {
	if v.dType != Float32Vector {
		return nil, false
	}
	return v.float32Data, true
}

// PackedBit returns the byte slice representing the binary quantized (packed bit) vector and the byte padding, which
// is the number of bits in the final byte that are to be ignored.
// It panics if v is not a packed bit vector.
func (v Vector) PackedBit() ([]byte, uint8) {
	d, p, ok := v.PackedBitOK()
	if !ok {
		panic(vectorTypeError{"bson.Vector.PackedBit", v.dType})
	}
	return d, p
}

// PackedBitOK is the same as PackedBit, but returns a boolean instead of panicking.
func (v Vector) PackedBitOK() ([]byte, uint8, bool) {
	if v.dType != PackedBitVector {
		return nil, 0, false
	}
	return v.bitData, v.bitPadding, true
}

// Binary returns the BSON Binary representation of the Vector.
func (v Vector) Binary() Binary {
	switch v.Type() {
	case Int8Vector:
		return binaryFromInt8Vector(v.Int8())
	case Float32Vector:
		return binaryFromFloat32Vector(v.Float32())
	case PackedBitVector:
		return binaryFromBitVector(v.PackedBit())
	default:
		panic(fmt.Sprintf("invalid Vector data type: %d", v.dType))
	}
}

func binaryFromInt8Vector(v []int8) Binary {
	data := make([]byte, len(v)+2)
	data[0] = Int8Vector
	data[1] = 0

	for i, e := range v {
		data[i+2] = byte(e)
	}

	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}
}

func binaryFromFloat32Vector(v []float32) Binary {
	data := make([]byte, 2, len(v)*4+2)
	data[0] = Float32Vector
	data[1] = 0
	var a [4]byte
	for _, e := range v {
		binary.LittleEndian.PutUint32(a[:], math.Float32bits(e))
		data = append(data, a[:]...)
	}

	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}
}

func binaryFromBitVector(bits []byte, padding uint8) Binary {
	data := make([]byte, len(bits)+2)
	data[0] = PackedBitVector
	data[1] = padding
	copy(data[2:], bits)
	return Binary{
		Subtype: TypeBinaryVector,
		Data:    data,
	}
}

// NewVector constructs a Vector from a slice of int8 or float32.
func NewVector[T int8 | float32](data []T) Vector {
	var v Vector
	switch a := any(data).(type) {
	case []int8:
		v.dType = Int8Vector
		v.int8Data = make([]int8, len(data))
		copy(v.int8Data, a)
	case []float32:
		v.dType = Float32Vector
		v.float32Data = make([]float32, len(data))
		copy(v.float32Data, a)
	default:
		panic(fmt.Errorf("unsupported type %T", data))
	}
	return v
}

// NewPackedBitVector constructs a Vector from a byte slice and a value of byte padding.
func NewPackedBitVector(bits []byte, padding uint8) (Vector, error) {
	var v Vector
	if padding > 7 {
		return v, errVectorPaddingTooLarge
	}
	if padding > 0 && len(bits) == 0 {
		return v, errNonZeroVectorPadding
	}
	v.dType = PackedBitVector
	v.bitData = make([]byte, len(bits))
	copy(v.bitData, bits)
	v.bitPadding = padding
	return v, nil
}

// NewVectorFromBinary unpacks a BSON Binary into a Vector.
func NewVectorFromBinary(b Binary) (Vector, error) {
	var v Vector
	if b.Subtype != TypeBinaryVector {
		return v, errors.New("not a vector")
	}
	if len(b.Data) < 2 {
		return v, errInsufficientVectorData
	}
	switch t := b.Data[0]; t {
	case Int8Vector:
		return newInt8Vector(b.Data[1:])
	case Float32Vector:
		return newFloat32Vector(b.Data[1:])
	case PackedBitVector:
		return newBitVector(b.Data[1:])
	default:
		return v, fmt.Errorf("invalid Vector data type: %d", t)
	}
}

func newInt8Vector(b []byte) (Vector, error) {
	var v Vector
	if len(b) == 0 {
		return v, errInsufficientVectorData
	}
	if padding := b[0]; padding > 0 {
		return v, errNonZeroVectorPadding
	}
	s := make([]int8, 0, len(b)-1)
	for i := 1; i < len(b); i++ {
		s = append(s, int8(b[i]))
	}
	return NewVector(s), nil
}

func newFloat32Vector(b []byte) (Vector, error) {
	var v Vector
	if len(b) == 0 {
		return v, errInsufficientVectorData
	}
	if padding := b[0]; padding > 0 {
		return v, errNonZeroVectorPadding
	}
	l := (len(b) - 1) / 4
	if l*4 != len(b)-1 {
		return v, errInsufficientVectorData
	}
	s := make([]float32, 0, l)
	for i := 1; i < len(b); i += 4 {
		s = append(s, math.Float32frombits(binary.LittleEndian.Uint32(b[i:i+4])))
	}
	return NewVector(s), nil
}

func newBitVector(b []byte) (Vector, error) {
	if len(b) == 0 {
		return Vector{}, errInsufficientVectorData
	}
	return NewPackedBitVector(b[1:], b[0])
}
