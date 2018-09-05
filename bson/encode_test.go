// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

func docToBytes(d *Document) []byte {
	b, err := d.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

func arrToBytes(a *Array) []byte {
	b, err := a.MarshalBSON()
	if err != nil {
		panic(err)
	}
	return b
}

type byteMarshaler []byte

func (bm byteMarshaler) MarshalBSON() ([]byte, error) { return bm, nil }

type _Interface interface {
	method()
}

type _impl struct {
	Foo string
}

func (_impl) method() {}
