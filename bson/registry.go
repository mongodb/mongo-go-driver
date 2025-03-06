// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// defaultRegistry is the default Registry. It contains the default codecs and the
// primitive codecs.
var defaultRegistry = NewRegistry()

// errNoEncoder is returned when there wasn't an encoder available for a type.
type errNoEncoder struct {
	Type reflect.Type
}

func (ene errNoEncoder) Error() string {
	if ene.Type == nil {
		return "no encoder found for <nil>"
	}
	return "no encoder found for " + ene.Type.String()
}

// errNoDecoder is returned when there wasn't a decoder available for a type.
type errNoDecoder struct {
	Type reflect.Type
}

func (end errNoDecoder) Error() string {
	return "no decoder found for " + end.Type.String()
}

// errNoTypeMapEntry is returned when there wasn't a type available for the provided BSON type.
type errNoTypeMapEntry struct {
	Type Type
}

func (entme errNoTypeMapEntry) Error() string {
	return "no type map entry found for " + entme.Type.String()
}

// A Registry is a store for ValueEncoders, ValueDecoders, and a type map. See the Registry type
// documentation for examples of registering various custom encoders and decoders. A Registry can
// have four main types of codecs:
//
// 1. Type encoders/decoders - These can be registered using the RegisterTypeEncoder and
// RegisterTypeDecoder methods. The registered codec will be invoked when encoding/decoding a value
// whose type matches the registered type exactly.
// If the registered type is an interface, the codec will be invoked when encoding or decoding
// values whose type is the interface, but not for values with concrete types that implement the
// interface.
//
// 2. Interface encoders/decoders - These can be registered using the RegisterInterfaceEncoder and
// RegisterInterfaceDecoder methods. These methods only accept interface types and the registered codecs
// will be invoked when encoding or decoding values whose types implement the interface. An example
// of an interface defined by the driver is bson.Marshaler. The driver will call the MarshalBSON method
// for any value whose type implements bson.Marshaler, regardless of the value's concrete type.
//
// 3. Type map entries - This can be used to associate a BSON type with a Go type. These type
// associations are used when decoding into a bson.D/bson.M or a struct field of type interface{}.
// For example, by default, BSON int32 and int64 values decode as Go int32 and int64 instances,
// respectively, when decoding into a bson.D. The following code would change the behavior so these
// values decode as Go int instances instead:
//
//	intType := reflect.TypeOf(int(0))
//	registry.RegisterTypeMapEntry(bson.TypeInt32, intType).RegisterTypeMapEntry(bson.TypeInt64, intType)
//
// 4. Kind encoder/decoders - These can be registered using the RegisterDefaultEncoder and
// RegisterDefaultDecoder methods. The registered codec will be invoked when encoding or decoding
// values whose reflect.Kind matches the registered reflect.Kind as long as the value's type doesn't
// match a registered type or interface encoder/decoder first. These methods should be used to change the
// behavior for all values for a specific kind.
//
// Read [Registry.LookupDecoder] and [Registry.LookupEncoder] for Registry lookup procedure.
type Registry struct {
	interfaceEncoders []interfaceValueEncoder
	interfaceDecoders []interfaceValueDecoder
	typeEncoders      *typeEncoderCache
	typeDecoders      *typeDecoderCache
	kindEncoders      *kindEncoderCache
	kindDecoders      *kindDecoderCache
	typeMap           sync.Map // map[Type]reflect.Type
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	reg := &Registry{
		typeEncoders: new(typeEncoderCache),
		typeDecoders: new(typeDecoderCache),
		kindEncoders: new(kindEncoderCache),
		kindDecoders: new(kindDecoderCache),
	}
	registerDefaultEncoders(reg)
	registerDefaultDecoders(reg)
	registerPrimitiveCodecs(reg)
	return reg
}

// RegisterTypeEncoder registers the provided ValueEncoder for the provided type.
//
// The type will be used as provided, so an encoder can be registered for a type and a different
// encoder can be registered for a pointer to that type.
//
// If the given type is an interface, the encoder will be called when marshaling a type that is
// that interface. It will not be called when marshaling a non-interface type that implements the
// interface. To get the latter behavior, call RegisterHookEncoder instead.
//
// RegisterTypeEncoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterTypeEncoder(valueType reflect.Type, enc ValueEncoder) {
	r.typeEncoders.Store(valueType, enc)
}

// RegisterTypeDecoder registers the provided ValueDecoder for the provided type.
//
// The type will be used as provided, so a decoder can be registered for a type and a different
// decoder can be registered for a pointer to that type.
//
// If the given type is an interface, the decoder will be called when unmarshaling into a type that
// is that interface. It will not be called when unmarshaling into a non-interface type that
// implements the interface. To get the latter behavior, call RegisterHookDecoder instead.
//
// RegisterTypeDecoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterTypeDecoder(valueType reflect.Type, dec ValueDecoder) {
	r.typeDecoders.Store(valueType, dec)
}

// RegisterKindEncoder registers the provided ValueEncoder for the provided kind.
//
// Use RegisterKindEncoder to register an encoder for any type with the same underlying kind. For
// example, consider the type MyInt defined as
//
//	type MyInt int32
//
// To define an encoder for MyInt and int32, use RegisterKindEncoder like
//
//	reg.RegisterKindEncoder(reflect.Int32, myEncoder)
//
// RegisterKindEncoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterKindEncoder(kind reflect.Kind, enc ValueEncoder) {
	r.kindEncoders.Store(kind, enc)
}

// RegisterKindDecoder registers the provided ValueDecoder for the provided kind.
//
// Use RegisterKindDecoder to register a decoder for any type with the same underlying kind. For
// example, consider the type MyInt defined as
//
//	type MyInt int32
//
// To define an decoder for MyInt and int32, use RegisterKindDecoder like
//
//	reg.RegisterKindDecoder(reflect.Int32, myDecoder)
//
// RegisterKindDecoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterKindDecoder(kind reflect.Kind, dec ValueDecoder) {
	r.kindDecoders.Store(kind, dec)
}

// RegisterInterfaceEncoder registers an encoder for the provided interface type iface. This encoder will
// be called when marshaling a type if the type implements iface or a pointer to the type
// implements iface. If the provided type is not an interface
// (i.e. iface.Kind() != reflect.Interface), this method will panic.
//
// RegisterInterfaceEncoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterInterfaceEncoder(iface reflect.Type, enc ValueEncoder) {
	if iface.Kind() != reflect.Interface {
		panicStr := fmt.Errorf("RegisterInterfaceEncoder expects a type with kind reflect.Interface, "+
			"got type %s with kind %s", iface, iface.Kind())
		panic(panicStr)
	}

	for idx, encoder := range r.interfaceEncoders {
		if encoder.i == iface {
			r.interfaceEncoders[idx].ve = enc
			return
		}
	}

	r.interfaceEncoders = append(r.interfaceEncoders, interfaceValueEncoder{i: iface, ve: enc})
}

// RegisterInterfaceDecoder registers an decoder for the provided interface type iface. This decoder will
// be called when unmarshaling into a type if the type implements iface or a pointer to the type
// implements iface. If the provided type is not an interface (i.e. iface.Kind() != reflect.Interface),
// this method will panic.
//
// RegisterInterfaceDecoder should not be called concurrently with any other Registry method.
func (r *Registry) RegisterInterfaceDecoder(iface reflect.Type, dec ValueDecoder) {
	if iface.Kind() != reflect.Interface {
		panicStr := fmt.Errorf("RegisterInterfaceDecoder expects a type with kind reflect.Interface, "+
			"got type %s with kind %s", iface, iface.Kind())
		panic(panicStr)
	}

	for idx, decoder := range r.interfaceDecoders {
		if decoder.i == iface {
			r.interfaceDecoders[idx].vd = dec
			return
		}
	}

	r.interfaceDecoders = append(r.interfaceDecoders, interfaceValueDecoder{i: iface, vd: dec})
}

// RegisterTypeMapEntry will register the provided type to the BSON type. The primary usage for this
// mapping is decoding situations where an empty interface is used and a default type needs to be
// created and decoded into.
//
// By default, BSON documents will decode into interface{} values as bson.D. To change the default type for BSON
// documents, a type map entry for TypeEmbeddedDocument should be registered. For example, to force BSON documents
// to decode to bson.Raw, use the following code:
//
//	reg.RegisterTypeMapEntry(TypeEmbeddedDocument, reflect.TypeOf(bson.Raw{}))
func (r *Registry) RegisterTypeMapEntry(bt Type, rt reflect.Type) {
	r.typeMap.Store(bt, rt)
}

// LookupEncoder returns the first matching encoder in the Registry. It uses the following lookup
// order:
//
// 1. An encoder registered for the exact type. If the given type is an interface, an encoder
// registered using RegisterTypeEncoder for that interface will be selected.
//
// 2. An encoder registered using RegisterInterfaceEncoder for an interface implemented by the type
// or by a pointer to the type. If the value matches multiple interfaces (e.g. the type implements
// bson.Marshaler and bson.ValueMarshaler), the first one registered will be selected.
// Note that registries constructed using bson.NewRegistry have driver-defined interfaces registered
// for the bson.Marshaler, bson.ValueMarshaler, and bson.Proxy interfaces, so those will take
// precedence over any new interfaces.
//
// 3. An encoder registered using RegisterKindEncoder for the kind of value.
//
// If no encoder is found, an error of type ErrNoEncoder is returned. LookupEncoder is safe for
// concurrent use by multiple goroutines after all codecs and encoders are registered.
func (r *Registry) LookupEncoder(valueType reflect.Type) (ValueEncoder, error) {
	if valueType == nil {
		return nil, errNoEncoder{Type: valueType}
	}
	enc, found := r.lookupTypeEncoder(valueType)
	if found {
		if enc == nil {
			return nil, errNoEncoder{Type: valueType}
		}
		return enc, nil
	}

	enc, found = r.lookupInterfaceEncoder(valueType, true)
	if found {
		return r.typeEncoders.LoadOrStore(valueType, enc), nil
	}

	if v, ok := r.kindEncoders.Load(valueType.Kind()); ok {
		return r.storeTypeEncoder(valueType, v), nil
	}
	return nil, errNoEncoder{Type: valueType}
}

func (r *Registry) storeTypeEncoder(rt reflect.Type, enc ValueEncoder) ValueEncoder {
	return r.typeEncoders.LoadOrStore(rt, enc)
}

func (r *Registry) lookupTypeEncoder(rt reflect.Type) (ValueEncoder, bool) {
	return r.typeEncoders.Load(rt)
}

func (r *Registry) lookupInterfaceEncoder(valueType reflect.Type, allowAddr bool) (ValueEncoder, bool) {
	if valueType == nil {
		return nil, false
	}
	for _, ienc := range r.interfaceEncoders {
		if valueType.Implements(ienc.i) {
			return ienc.ve, true
		}
		if allowAddr && valueType.Kind() != reflect.Ptr && reflect.PtrTo(valueType).Implements(ienc.i) {
			// if *t implements an interface, this will catch if t implements an interface further
			// ahead in interfaceEncoders
			defaultEnc, found := r.lookupInterfaceEncoder(valueType, false)
			if !found {
				defaultEnc, _ = r.kindEncoders.Load(valueType.Kind())
			}
			return newCondAddrEncoder(ienc.ve, defaultEnc), true
		}
	}
	return nil, false
}

// LookupDecoder returns the first matching decoder in the Registry. It uses the following lookup
// order:
//
// 1. A decoder registered for the exact type. If the given type is an interface, a decoder
// registered using RegisterTypeDecoder for that interface will be selected.
//
// 2. A decoder registered using RegisterInterfaceDecoder for an interface implemented by the type or by
// a pointer to the type. If the value matches multiple interfaces (e.g. the type implements
// bson.Unmarshaler and bson.ValueUnmarshaler), the first one registered will be selected.
// Note that registries constructed using bson.NewRegistry have driver-defined interfaces registered
// for the bson.Unmarshaler and bson.ValueUnmarshaler interfaces, so those will take
// precedence over any new interfaces.
//
// 3. A decoder registered using RegisterKindDecoder for the kind of value.
//
// If no decoder is found, an error of type ErrNoDecoder is returned. LookupDecoder is safe for
// concurrent use by multiple goroutines after all codecs and decoders are registered.
func (r *Registry) LookupDecoder(valueType reflect.Type) (ValueDecoder, error) {
	if valueType == nil {
		return nil, errors.New("cannot perform a decoder lookup on <nil>")
	}
	dec, found := r.lookupTypeDecoder(valueType)
	if found {
		if dec == nil {
			return nil, errNoDecoder{Type: valueType}
		}
		return dec, nil
	}

	dec, found = r.lookupInterfaceDecoder(valueType, true)
	if found {
		return r.storeTypeDecoder(valueType, dec), nil
	}

	if v, ok := r.kindDecoders.Load(valueType.Kind()); ok {
		return r.storeTypeDecoder(valueType, v), nil
	}
	return nil, errNoDecoder{Type: valueType}
}

func (r *Registry) lookupTypeDecoder(valueType reflect.Type) (ValueDecoder, bool) {
	return r.typeDecoders.Load(valueType)
}

func (r *Registry) storeTypeDecoder(typ reflect.Type, dec ValueDecoder) ValueDecoder {
	return r.typeDecoders.LoadOrStore(typ, dec)
}

func (r *Registry) lookupInterfaceDecoder(valueType reflect.Type, allowAddr bool) (ValueDecoder, bool) {
	for _, idec := range r.interfaceDecoders {
		if valueType.Implements(idec.i) {
			return idec.vd, true
		}
		if allowAddr && valueType.Kind() != reflect.Ptr && reflect.PtrTo(valueType).Implements(idec.i) {
			// if *t implements an interface, this will catch if t implements an interface further
			// ahead in interfaceDecoders
			defaultDec, found := r.lookupInterfaceDecoder(valueType, false)
			if !found {
				defaultDec, _ = r.kindDecoders.Load(valueType.Kind())
			}
			return newCondAddrDecoder(idec.vd, defaultDec), true
		}
	}
	return nil, false
}

// LookupTypeMapEntry inspects the registry's type map for a Go type for the corresponding BSON
// type. If no type is found, ErrNoTypeMapEntry is returned.
//
// LookupTypeMapEntry should not be called concurrently with any other Registry method.
func (r *Registry) LookupTypeMapEntry(bt Type) (reflect.Type, error) {
	v, ok := r.typeMap.Load(bt)
	if v == nil || !ok {
		return nil, errNoTypeMapEntry{Type: bt}
	}
	return v.(reflect.Type), nil
}

type interfaceValueEncoder struct {
	i  reflect.Type
	ve ValueEncoder
}

type interfaceValueDecoder struct {
	i  reflect.Type
	vd ValueDecoder
}
