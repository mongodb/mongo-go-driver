// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"sync"
)

// DefaultRegistry is the default Registry. It contains the default codecs and the
// primitive codecs.
var DefaultRegistry = NewRegistryBuilder().Build()

// ErrNoEncoder is returned when there wasn't an encoder available for a type.
//
// Deprecated: ErrNoEncoder will not be supported in Go Driver 2.0.
type ErrNoEncoder struct {
	Type reflect.Type
}

func (ene ErrNoEncoder) Error() string {
	if ene.Type == nil {
		return "no encoder found for <nil>"
	}
	return "no encoder found for " + ene.Type.String()
}

// ErrNoDecoder is returned when there wasn't a decoder available for a type.
//
// Deprecated: ErrNoDecoder will not be supported in Go Driver 2.0.
type ErrNoDecoder struct {
	Type reflect.Type
}

func (end ErrNoDecoder) Error() string {
	return "no decoder found for " + end.Type.String()
}

// ErrNoTypeMapEntry is returned when there wasn't a type available for the provided BSON type.
//
// Deprecated: ErrNoTypeMapEntry will not be supported in Go Driver 2.0.
type ErrNoTypeMapEntry struct {
	Type Type
}

func (entme ErrNoTypeMapEntry) Error() string {
	return "no type map entry found for " + entme.Type.String()
}

// EncoderFactory is a factory function that generates a new ValueEncoder.
type EncoderFactory func() ValueEncoder

// DecoderFactory is a factory function that generates a new ValueDecoder.
type DecoderFactory func() ValueDecoder

// A RegistryBuilder is used to build a Registry. This type is not goroutine
// safe.
type RegistryBuilder struct {
	typeEncoders      map[reflect.Type]EncoderFactory
	typeDecoders      *typeDecoderCache
	interfaceEncoders map[reflect.Type]EncoderFactory
	interfaceDecoders []interfaceValueDecoder
	kindEncoders      [reflect.UnsafePointer + 1]EncoderFactory
	kindDecoders      *kindDecoderCache
	typeMap           map[Type]reflect.Type
}

// NewRegistryBuilder creates a new empty RegistryBuilder.
func NewRegistryBuilder() *RegistryBuilder {
	rb := &RegistryBuilder{
		typeEncoders:      make(map[reflect.Type]EncoderFactory),
		typeDecoders:      new(typeDecoderCache),
		interfaceEncoders: make(map[reflect.Type]EncoderFactory),
		kindDecoders:      new(kindDecoderCache),
		typeMap:           make(map[Type]reflect.Type),
	}
	registerDefaultEncoders(rb)
	registerDefaultDecoders(rb)
	registerPrimitiveCodecs(rb)
	return rb
}

// RegisterTypeEncoder registers a ValueEncoder factory for the provided type.
//
// The type will be used as provided, so an encoder factory can be registered for a type and a
// different one can be registered for a pointer to that type.
//
// If the given type is an interface, the encoder will be called when marshaling a type that is
// that interface. It will not be called when marshaling a non-interface type that implements the
// interface. To get the latter behavior, call RegisterInterfaceEncoder instead.
//
// RegisterTypeEncoder should not be called concurrently with any other Registry method.
func (rb *RegistryBuilder) RegisterTypeEncoder(valueType reflect.Type, encFac EncoderFactory) *RegistryBuilder {
	if encFac != nil {
		rb.typeEncoders[valueType] = encFac
	}
	return rb
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
func (rb *RegistryBuilder) RegisterTypeDecoder(valueType reflect.Type, dec ValueDecoder) *RegistryBuilder {
	rb.typeDecoders.Store(valueType, dec)
	return rb
}

// RegisterKindEncoder registers a ValueEncoder factory for the provided kind.
//
// Use RegisterKindEncoder to register an encoder factory for any type with the same underlying kind.
// For example, consider the type MyInt defined as
//
//	type MyInt int32
//
// To define an encoder factory for MyInt and int32, use RegisterKindEncoder like
//
//	reg.RegisterKindEncoder(reflect.Int32, myEncoder)
//
// RegisterKindEncoder should not be called concurrently with any other Registry method.
func (rb *RegistryBuilder) RegisterKindEncoder(kind reflect.Kind, encFac EncoderFactory) *RegistryBuilder {
	if encFac != nil && kind < reflect.Kind(len(rb.kindEncoders)) {
		rb.kindEncoders[kind] = encFac
	}
	return rb
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
func (rb *RegistryBuilder) RegisterKindDecoder(kind reflect.Kind, dec ValueDecoder) *RegistryBuilder {
	rb.kindDecoders.Store(kind, dec)
	return rb
}

// RegisterInterfaceEncoder registers an encoder factory for the provided interface type iface. This
// encoder will be called when marshaling a type if the type implements iface or a pointer to the type
// implements iface. If the provided type is not an interface
// (i.e. iface.Kind() != reflect.Interface), this method will panic.
//
// RegisterInterfaceEncoder should not be called concurrently with any other Registry method.
func (rb *RegistryBuilder) RegisterInterfaceEncoder(iface reflect.Type, encFac EncoderFactory) *RegistryBuilder {
	if iface.Kind() != reflect.Interface {
		panicStr := fmt.Errorf("RegisterInterfaceEncoder expects a type with kind reflect.Interface, "+
			"got type %s with kind %s", iface, iface.Kind())
		panic(panicStr)
	}

	if encFac != nil {
		rb.interfaceEncoders[iface] = encFac
	}

	return rb
}

// RegisterInterfaceDecoder registers an decoder for the provided interface type iface. This decoder will
// be called when unmarshaling into a type if the type implements iface or a pointer to the type
// implements iface. If the provided type is not an interface (i.e. iface.Kind() != reflect.Interface),
// this method will panic.
//
// RegisterInterfaceDecoder should not be called concurrently with any other Registry method.
func (rb *RegistryBuilder) RegisterInterfaceDecoder(iface reflect.Type, dec ValueDecoder) *RegistryBuilder {
	if iface.Kind() != reflect.Interface {
		panicStr := fmt.Errorf("RegisterInterfaceDecoder expects a type with kind reflect.Interface, "+
			"got type %s with kind %s", iface, iface.Kind())
		panic(panicStr)
	}

	for idx, decoder := range rb.interfaceDecoders {
		if decoder.i == iface {
			rb.interfaceDecoders[idx].vd = dec
			return rb
		}
	}

	rb.interfaceDecoders = append(rb.interfaceDecoders, interfaceValueDecoder{i: iface, vd: dec})

	return rb
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
//
// RegisterTypeMapEntry should not be called concurrently with any other Registry method.
func (rb *RegistryBuilder) RegisterTypeMapEntry(bt Type, rt reflect.Type) *RegistryBuilder {
	rb.typeMap[bt] = rt
	return rb
}

// Build creates a Registry from the current state of this RegistryBuilder.
func (rb *RegistryBuilder) Build() *Registry {
	r := &Registry{
		typeEncoders:      new(sync.Map),
		typeDecoders:      rb.typeDecoders.Clone(),
		interfaceEncoders: make([]interfaceValueEncoder, 0, len(rb.interfaceEncoders)),
		interfaceDecoders: append([]interfaceValueDecoder(nil), rb.interfaceDecoders...),
		kindDecoders:      rb.kindDecoders.Clone(),
		encoderTypeMap:    make(map[reflect.Type][]ValueEncoder),
		typeMap:           make(map[Type]reflect.Type),
	}
	encoderCache := make(map[reflect.Value]ValueEncoder)
	for k, v := range rb.typeEncoders {
		var encoder ValueEncoder
		if enc, ok := encoderCache[reflect.ValueOf(v)]; ok {
			encoder = enc
		} else {
			encoder = v()
			encoderCache[reflect.ValueOf(v)] = encoder
			et := reflect.ValueOf(encoder).Type()
			r.encoderTypeMap[et] = append(r.encoderTypeMap[et], encoder)
		}
		r.typeEncoders.Store(k, encoder)
	}
	for k, v := range rb.interfaceEncoders {
		var encoder ValueEncoder
		if enc, ok := encoderCache[reflect.ValueOf(v)]; ok {
			encoder = enc
		} else {
			encoder = v()
			encoderCache[reflect.ValueOf(v)] = encoder
			et := reflect.ValueOf(encoder).Type()
			r.encoderTypeMap[et] = append(r.encoderTypeMap[et], encoder)
		}
		r.interfaceEncoders = append(r.interfaceEncoders, interfaceValueEncoder{k, encoder})
	}
	for i, v := range rb.kindEncoders {
		if v == nil {
			continue
		}
		var encoder ValueEncoder
		if enc, ok := encoderCache[reflect.ValueOf(v)]; ok {
			encoder = enc
		} else {
			encoder = v()
			encoderCache[reflect.ValueOf(v)] = encoder
			et := reflect.ValueOf(encoder).Type()
			r.encoderTypeMap[et] = append(r.encoderTypeMap[et], encoder)
		}
		r.kindEncoders[i] = encoder
	}
	for k, v := range rb.typeMap {
		r.typeMap[k] = v
	}
	return r
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
	typeEncoders      *sync.Map // map[reflect.Type]ValueEncoder
	typeDecoders      *typeDecoderCache
	interfaceEncoders []interfaceValueEncoder
	interfaceDecoders []interfaceValueDecoder
	kindEncoders      [reflect.UnsafePointer + 1]ValueEncoder
	kindDecoders      *kindDecoderCache
	typeMap           map[Type]reflect.Type

	encoderTypeMap map[reflect.Type][]ValueEncoder
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
// concurrent use by multiple goroutines.
func (r *Registry) LookupEncoder(valueType reflect.Type) (ValueEncoder, error) {
	if valueType == nil {
		return nil, ErrNoEncoder{Type: valueType}
	}

	if enc, found := r.typeEncoders.Load(valueType); found {
		if enc == nil {
			return nil, ErrNoEncoder{Type: valueType}
		}
		return enc.(ValueEncoder), nil
	}

	if enc, found := r.lookupInterfaceEncoder(valueType, true); found {
		r.typeEncoders.Store(valueType, enc)
		return enc, nil
	}

	if enc, found := r.lookupKindEncoder(valueType.Kind()); found {
		r.typeEncoders.Store(valueType, enc)
		return enc, nil
	}
	return nil, ErrNoEncoder{Type: valueType}
}

func (r *Registry) lookupKindEncoder(valueKind reflect.Kind) (ValueEncoder, bool) {
	if valueKind < reflect.Kind(len(r.kindEncoders)) {
		if enc := r.kindEncoders[valueKind]; enc != nil {
			return enc, true
		}
	}
	return nil, false
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
				defaultEnc, _ = r.lookupKindEncoder(valueType.Kind())
			}
			return &condAddrEncoder{canAddrEnc: ienc.ve, elseEnc: defaultEnc}, true
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
// concurrent use by multiple goroutines.
func (r *Registry) LookupDecoder(valueType reflect.Type) (ValueDecoder, error) {
	if valueType == nil {
		return nil, ErrNoDecoder{Type: valueType}
	}
	dec, found := r.typeDecoders.Load(valueType)
	if found {
		if dec == nil {
			return nil, ErrNoDecoder{Type: valueType}
		}
		return dec, nil
	}

	dec, found = r.lookupInterfaceDecoder(valueType, true)
	if found {
		return r.typeDecoders.LoadOrStore(valueType, dec), nil
	}

	if v, ok := r.kindDecoders.Load(valueType.Kind()); ok {
		return r.typeDecoders.LoadOrStore(valueType, v), nil
	}
	return nil, ErrNoDecoder{Type: valueType}
}

func (r *Registry) lookupInterfaceDecoder(valueType reflect.Type, allowAddr bool) (ValueDecoder, bool) {
	if valueType == nil {
		return nil, false
	}
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
			return &condAddrDecoder{canAddrDec: idec.vd, elseDec: defaultDec}, true
		}
	}
	return nil, false
}

// LookupTypeMapEntry inspects the registry's type map for a Go type for the corresponding BSON
// type. If no type is found, ErrNoTypeMapEntry is returned.
func (r *Registry) LookupTypeMapEntry(bt Type) (reflect.Type, error) {
	v, ok := r.typeMap[bt]
	if v == nil || !ok {
		return nil, ErrNoTypeMapEntry{Type: bt}
	}
	return v, nil
}

type interfaceValueEncoder struct {
	i  reflect.Type
	ve ValueEncoder
}

type interfaceValueDecoder struct {
	i  reflect.Type
	vd ValueDecoder
}
