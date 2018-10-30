// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"strconv"
)

type testArrayPrependAppendGenerator struct{}

var tapag testArrayPrependAppendGenerator

func (testArrayPrependAppendGenerator) oneOne() [][]Valuev2 {
	return [][]Valuev2{
		{Double(3.14159)},
	}
}

func (testArrayPrependAppendGenerator) oneOneBytes(index uint) []byte {
	a := []byte{
		// size
		0x0, 0x0, 0x0, 0x0,
		// type
		0x1,
	}

	// key
	a = append(a, []byte(strconv.FormatUint(uint64(index), 10))...)
	a = append(a, 0)

	a = append(a,
		// value
		0x6e, 0x86, 0x1b, 0xf0, 0xf9, 0x21, 0x9, 0x40,
		// null terminator
		0x0,
	)

	a[0] = byte(len(a))

	return a
}

func (testArrayPrependAppendGenerator) twoOne() [][]Valuev2 {
	return [][]Valuev2{
		{Double(1.234)},
		{Double(5.678)},
	}
}

func (testArrayPrependAppendGenerator) twoOneAppendBytes(index uint) []byte {
	a := []byte{
		// size
		0x0, 0x0, 0x0, 0x0,
		// type
		0x1,
	}

	// key
	a = append(a, []byte(strconv.FormatUint(uint64(index), 10))...)
	a = append(a, 0)

	a = append(a,
		// value
		0x58, 0x39, 0xb4, 0xc8, 0x76, 0xbe, 0xf3, 0x3f,
		// type
		0x1,
	)

	// key
	a = append(a, []byte(strconv.FormatUint(uint64(index+1), 10))...)
	a = append(a, 0)

	a = append(a,
		// value
		0x83, 0xc0, 0xca, 0xa1, 0x45, 0xb6, 0x16, 0x40,
		// null terminator
		0x0,
	)

	a[0] = byte(len(a))

	return a
}

func (testArrayPrependAppendGenerator) twoOnePrependBytes(index uint) []byte {
	a := []byte{
		// size
		0x0, 0x0, 0x0, 0x0,
		// type
		0x1,
	}

	// key
	a = append(a, []byte(strconv.FormatUint(uint64(index), 10))...)
	a = append(a, 0)

	a = append(a,
		// value
		0x83, 0xc0, 0xca, 0xa1, 0x45, 0xb6, 0x16, 0x40,
		// type
		0x1,
	)

	// key
	a = append(a, []byte(strconv.FormatUint(uint64(index+1), 10))...)
	a = append(a, 0)

	a = append(a,
		// value
		0x58, 0x39, 0xb4, 0xc8, 0x76, 0xbe, 0xf3, 0x3f,
		// null terminator
		0x0,
	)

	a[0] = byte(len(a))

	return a
}

func ExampleArray() {
	internalVersion := "1234567"

	f := func(appName string) *Arrayv2 {
		arr := NewArrayv2()
		arr.Append(
			Embed(NewDocumentv2(
				Elementv2{Key: "name", Value: String("mongo-go-driver")},
				Elementv2{Key: "version", Value: String(internalVersion)},
			)),
			Embed(NewDocumentv2(
				Elementv2{Key: "type", Value: String("darwin")},
				Elementv2{Key: "architecture", Value: String("amd64")},
			)),
			String("go1.9.2"),
		)
		if appName != "" {
			arr.Append(Embed(NewDocumentv2(Elementv2{Key: "name", Value: String(appName)})))
		}

		return arr
	}
	_, buf, err := f("hello-world").MarshalBSONValue()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf)

	// Output: [154 0 0 0 3 48 0 52 0 0 0 2 110 97 109 101 0 16 0 0 0 109 111 110 103 111 45 103 111 45 100 114 105 118 101 114 0 2 118 101 114 115 105 111 110 0 8 0 0 0 49 50 51 52 53 54 55 0 0 3 49 0 46 0 0 0 2 116 121 112 101 0 7 0 0 0 100 97 114 119 105 110 0 2 97 114 99 104 105 116 101 99 116 117 114 101 0 6 0 0 0 97 109 100 54 52 0 0 2 50 0 8 0 0 0 103 111 49 46 57 46 50 0 3 51 0 27 0 0 0 2 110 97 109 101 0 12 0 0 0 104 101 108 108 111 45 119 111 114 108 100 0 0 0]
}
