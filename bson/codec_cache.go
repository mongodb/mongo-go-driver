// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// Runtime check that the kind encoder and decoder caches can store any valid
// reflect.Kind constant.
func init() {
	if s := reflect.Kind(len(kindEncoderCache{}.entries)).String(); s != "kind27" {
		panic("The capacity of kindEncoderCache is too small.\n" +
			"This is due to a new type being added to reflect.Kind.")
	}
}

// statically assert array size
var _ = (kindEncoderCache{}).entries[reflect.UnsafePointer]
var _ = (kindDecoderCache{}).entries[reflect.UnsafePointer]

type typeEncoderCache struct {
	cache sync.Map // map[reflect.Type]ValueEncoder
}

func (c *typeEncoderCache) Store(rt reflect.Type, enc valueEncoder) {
	c.cache.Store(rt, enc)
}

func (c *typeEncoderCache) Load(rt reflect.Type) (valueEncoder, bool) {
	if v, _ := c.cache.Load(rt); v != nil {
		return v.(valueEncoder), true
	}
	return nil, false
}

func (c *typeEncoderCache) LoadOrStore(rt reflect.Type, enc valueEncoder) valueEncoder {
	if v, loaded := c.cache.LoadOrStore(rt, enc); loaded {
		enc = v.(valueEncoder)
	}
	return enc
}

func (c *typeEncoderCache) Clone() *typeEncoderCache {
	cc := new(typeEncoderCache)
	c.cache.Range(func(k, v interface{}) bool {
		if k != nil && v != nil {
			cc.cache.Store(k, v)
		}
		return true
	})
	return cc
}

type typeDecoderCache struct {
	cache sync.Map // map[reflect.Type]ValueDecoder
}

func (c *typeDecoderCache) Store(rt reflect.Type, dec valueDecoder) {
	c.cache.Store(rt, dec)
}

func (c *typeDecoderCache) Load(rt reflect.Type) (valueDecoder, bool) {
	if v, _ := c.cache.Load(rt); v != nil {
		return v.(valueDecoder), true
	}
	return nil, false
}

func (c *typeDecoderCache) LoadOrStore(rt reflect.Type, dec valueDecoder) valueDecoder {
	if v, loaded := c.cache.LoadOrStore(rt, dec); loaded {
		dec = v.(valueDecoder)
	}
	return dec
}

func (c *typeDecoderCache) Clone() *typeDecoderCache {
	cc := new(typeDecoderCache)
	c.cache.Range(func(k, v interface{}) bool {
		if k != nil && v != nil {
			cc.cache.Store(k, v)
		}
		return true
	})
	return cc
}

// atomic.Value requires that all calls to Store() have the same concrete type
// so we wrap the ValueEncoder with a kindEncoderCacheEntry to ensure the type
// is always the same (since different concrete types may implement the
// ValueEncoder interface).
type kindEncoderCacheEntry struct {
	enc valueEncoder
}

type kindEncoderCache struct {
	entries [reflect.UnsafePointer + 1]atomic.Value // *kindEncoderCacheEntry
}

func (c *kindEncoderCache) Store(rt reflect.Kind, enc valueEncoder) {
	if enc != nil && rt < reflect.Kind(len(c.entries)) {
		c.entries[rt].Store(&kindEncoderCacheEntry{enc: enc})
	}
}

func (c *kindEncoderCache) Load(rt reflect.Kind) (valueEncoder, bool) {
	if rt < reflect.Kind(len(c.entries)) {
		if ent, ok := c.entries[rt].Load().(*kindEncoderCacheEntry); ok {
			return ent.enc, ent.enc != nil
		}
	}
	return nil, false
}

func (c *kindEncoderCache) Clone() *kindEncoderCache {
	cc := new(kindEncoderCache)
	for i, v := range c.entries {
		if val := v.Load(); val != nil {
			cc.entries[i].Store(val)
		}
	}
	return cc
}

// atomic.Value requires that all calls to Store() have the same concrete type
// so we wrap the ValueDecoder with a kindDecoderCacheEntry to ensure the type
// is always the same (since different concrete types may implement the
// ValueDecoder interface).
type kindDecoderCacheEntry struct {
	dec valueDecoder
}

type kindDecoderCache struct {
	entries [reflect.UnsafePointer + 1]atomic.Value // *kindDecoderCacheEntry
}

func (c *kindDecoderCache) Store(rt reflect.Kind, dec valueDecoder) {
	if rt < reflect.Kind(len(c.entries)) {
		c.entries[rt].Store(&kindDecoderCacheEntry{dec: dec})
	}
}

func (c *kindDecoderCache) Load(rt reflect.Kind) (valueDecoder, bool) {
	if rt < reflect.Kind(len(c.entries)) {
		if ent, ok := c.entries[rt].Load().(*kindDecoderCacheEntry); ok {
			return ent.dec, ent.dec != nil
		}
	}
	return nil, false
}

func (c *kindDecoderCache) Clone() *kindDecoderCache {
	cc := new(kindDecoderCache)
	for i, v := range c.entries {
		if val := v.Load(); val != nil {
			cc.entries[i].Store(val)
		}
	}
	return cc
}
