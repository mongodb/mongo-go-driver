// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// NB(charlie): the array size is a power of 2 because we use the remainder of
// it (mod) in benchmarks and that is faster when the size is a power of 2.
var codecCacheTestTypes = [16]reflect.Type{
	reflect.TypeOf(uint8(0)),
	reflect.TypeOf(uint16(0)),
	reflect.TypeOf(uint32(0)),
	reflect.TypeOf(uint64(0)),
	reflect.TypeOf(uint(0)),
	reflect.TypeOf(uintptr(0)),
	reflect.TypeOf(int8(0)),
	reflect.TypeOf(int16(0)),
	reflect.TypeOf(int32(0)),
	reflect.TypeOf(int64(0)),
	reflect.TypeOf(int(0)),
	reflect.TypeOf(float32(0)),
	reflect.TypeOf(float64(0)),
	reflect.TypeOf(true),
	reflect.TypeOf(struct{ A int }{}),
	reflect.TypeOf(map[int]int{}),
}

func TestTypeCache(t *testing.T) {
	rt := reflect.TypeOf(int(0))
	ec := new(typeEncoderCache)
	dc := new(typeDecoderCache)

	codec := new(fakeCodec)
	ec.Store(rt, codec)
	dc.Store(rt, codec)
	if v, ok := ec.Load(rt); !ok || !reflect.DeepEqual(v, codec) {
		t.Errorf("Load(%s) = %v, %t; want: %v, %t", rt, v, ok, codec, true)
	}
	if v, ok := dc.Load(rt); !ok || !reflect.DeepEqual(v, codec) {
		t.Errorf("Load(%s) = %v, %t; want: %v, %t", rt, v, ok, codec, true)
	}

	// Make sure we overwrite the stored value with nil
	ec.Store(rt, nil)
	dc.Store(rt, nil)
	if v, ok := ec.Load(rt); ok || v != nil {
		t.Errorf("Load(%s) = %v, %t; want: %v, %t", rt, v, ok, nil, false)
	}
	if v, ok := dc.Load(rt); ok || v != nil {
		t.Errorf("Load(%s) = %v, %t; want: %v, %t", rt, v, ok, nil, false)
	}
}

func TestTypeCacheClone(t *testing.T) {
	codec := new(fakeCodec)
	ec1 := new(typeEncoderCache)
	dc1 := new(typeDecoderCache)
	for _, rt := range codecCacheTestTypes {
		ec1.Store(rt, codec)
		dc1.Store(rt, codec)
	}
	ec2 := ec1.Clone()
	dc2 := dc1.Clone()
	for _, rt := range codecCacheTestTypes {
		if v, _ := ec2.Load(rt); !reflect.DeepEqual(v, codec) {
			t.Errorf("Load(%s) = %#v; want: %#v", rt, v, codec)
		}
		if v, _ := dc2.Load(rt); !reflect.DeepEqual(v, codec) {
			t.Errorf("Load(%s) = %#v; want: %#v", rt, v, codec)
		}
	}
}

func TestKindCacheArray(t *testing.T) {
	// Check array bounds
	var c kindEncoderCache
	codec := new(fakeCodec)
	c.Store(reflect.UnsafePointer, codec)   // valid
	c.Store(reflect.UnsafePointer+1, codec) // ignored
	if v, ok := c.Load(reflect.UnsafePointer); !ok || v != codec {
		t.Errorf("Load(reflect.UnsafePointer) = %v, %t; want: %v, %t", v, ok, codec, true)
	}
	if v, ok := c.Load(reflect.UnsafePointer + 1); ok || v != nil {
		t.Errorf("Load(reflect.UnsafePointer + 1) = %v, %t; want: %v, %t", v, ok, nil, false)
	}

	// Make sure that reflect.UnsafePointer is the last/largest reflect.Type.
	//
	// The String() method of invalid reflect.Type types are of the format
	// "kind{NUMBER}".
	for rt := reflect.UnsafePointer + 1; rt < reflect.UnsafePointer+16; rt++ {
		s := rt.String()
		if !strings.Contains(s, strconv.Itoa(int(rt))) {
			t.Errorf("reflect.Type(%d) appears to be valid: %q", rt, s)
		}
	}
}

func TestKindCacheClone(t *testing.T) {
	e1 := new(kindEncoderCache)
	d1 := new(kindDecoderCache)
	codec := new(fakeCodec)
	for k := reflect.Invalid; k <= reflect.UnsafePointer; k++ {
		e1.Store(k, codec)
		d1.Store(k, codec)
	}
	e2 := e1.Clone()
	for k := reflect.Invalid; k <= reflect.UnsafePointer; k++ {
		v1, ok1 := e1.Load(k)
		v2, ok2 := e2.Load(k)
		if ok1 != ok2 || !reflect.DeepEqual(v1, v2) || v1 == nil || v2 == nil {
			t.Errorf("Encoder(%s): %#v, %t != %#v, %t", k, v1, ok1, v2, ok2)
		}
	}
	d2 := d1.Clone()
	for k := reflect.Invalid; k <= reflect.UnsafePointer; k++ {
		v1, ok1 := d1.Load(k)
		v2, ok2 := d2.Load(k)
		if ok1 != ok2 || !reflect.DeepEqual(v1, v2) || v1 == nil || v2 == nil {
			t.Errorf("Decoder(%s): %#v, %t != %#v, %t", k, v1, ok1, v2, ok2)
		}
	}
}

func TestKindCacheEncoderNilEncoder(t *testing.T) {
	t.Run("Encoder", func(t *testing.T) {
		c := new(kindEncoderCache)
		c.Store(reflect.Invalid, ValueEncoder(nil))
		v, ok := c.Load(reflect.Invalid)
		if v != nil || ok {
			t.Errorf("Load of nil ValueEncoder should return: nil, false; got: %v, %t", v, ok)
		}
	})
	t.Run("Decoder", func(t *testing.T) {
		c := new(kindDecoderCache)
		c.Store(reflect.Invalid, ValueDecoder(nil))
		v, ok := c.Load(reflect.Invalid)
		if v != nil || ok {
			t.Errorf("Load of nil ValueDecoder should return: nil, false; got: %v, %t", v, ok)
		}
	})
}

func BenchmarkEncoderCacheLoad(b *testing.B) {
	c := new(typeEncoderCache)
	codec := new(fakeCodec)
	typs := codecCacheTestTypes
	for _, t := range typs {
		c.Store(t, codec)
	}
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			c.Load(typs[i%len(typs)])
		}
	})
}

func BenchmarkEncoderCacheStore(b *testing.B) {
	c := new(typeEncoderCache)
	codec := new(fakeCodec)
	b.RunParallel(func(pb *testing.PB) {
		typs := codecCacheTestTypes
		for i := 0; pb.Next(); i++ {
			c.Store(typs[i%len(typs)], codec)
		}
	})
}
