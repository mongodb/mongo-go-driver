package bsoncodec

import (
	"errors"
	"reflect"
	"sync"
)

// ErrNilType is returned when nil is passed to either LookupEncoder or LookupDecoder.
var ErrNilType = errors.New("cannot perform an encoder or decoder lookup on <nil>")

// ErrNoEncoder is returned when there wasn't an encoder available for a type.
type ErrNoEncoder struct {
	Type reflect.Type
}

func (ene ErrNoEncoder) Error() string {
	return "no encoder found for " + ene.Type.String()
}

// ErrNoDecoder is returned when there wasn't a decoder available for a type.
type ErrNoDecoder struct {
	Type reflect.Type
}

func (end ErrNoDecoder) Error() string {
	return "no decoder found for " + end.Type.String()
}

// ErrNotInterface is returned when the provided type is not an interface.
var ErrNotInterface = errors.New("The provided type is not an interface")

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistryBuilder().Build()
}

// A RegistryBuilder is used to build a Registry. This type is not goroutine
// safe.
type RegistryBuilder struct {
	typeEncoders      map[reflect.Type]ValueEncoder
	interfaceEncoders []interfaceValueEncoder
	kindEncoders      map[reflect.Kind]ValueEncoder

	typeDecoders      map[reflect.Type]ValueDecoder
	interfaceDecoders []interfaceValueDecoder
	kindDecoders      map[reflect.Kind]ValueDecoder
}

// A Registry is used to store and retrieve codecs for types and interfaces. This type is the main
// typed passed around and Encoders and Decoders are constructed from it.
type Registry struct {
	typeEncoders map[reflect.Type]ValueEncoder
	typeDecoders map[reflect.Type]ValueDecoder

	interfaceEncoders []interfaceValueEncoder
	interfaceDecoders []interfaceValueDecoder

	kindEncoders map[reflect.Kind]ValueEncoder
	kindDecoders map[reflect.Kind]ValueDecoder

	mu sync.RWMutex
}

// NewEmptyRegistryBuilder creates a new RegistryBuilder with no default kind
// Codecs.
func NewEmptyRegistryBuilder() *RegistryBuilder {
	return &RegistryBuilder{
		typeEncoders: make(map[reflect.Type]ValueEncoder),
		typeDecoders: make(map[reflect.Type]ValueDecoder),

		interfaceEncoders: make([]interfaceValueEncoder, 0),
		interfaceDecoders: make([]interfaceValueDecoder, 0),

		kindEncoders: make(map[reflect.Kind]ValueEncoder),
		kindDecoders: make(map[reflect.Kind]ValueDecoder),
	}
}

// NewRegistryBuilder creates a new RegistryBuilder.
func NewRegistryBuilder() *RegistryBuilder {
	var dve DefaultValueEncoders
	var dvd DefaultValueDecoders
	typeEncoders := map[reflect.Type]ValueEncoder{
		tDocument:                     ValueEncoderFunc(dve.DocumentEncodeValue),
		tArray:                        ValueEncoderFunc(dve.ArrayEncodeValue),
		tValue:                        ValueEncoderFunc(dve.ValueEncodeValue),
		reflect.PtrTo(tByteSlice):     ValueEncoderFunc(dve.ByteSliceEncodeValue),
		reflect.PtrTo(tElementSlice):  ValueEncoderFunc(dve.ElementSliceEncodeValue),
		reflect.PtrTo(tTime):          ValueEncoderFunc(dve.TimeEncodeValue),
		reflect.PtrTo(tEmpty):         ValueEncoderFunc(dve.EmptyInterfaceEncodeValue),
		reflect.PtrTo(tBinary):        ValueEncoderFunc(dve.BooleanEncodeValue),
		reflect.PtrTo(tUndefined):     ValueEncoderFunc(dve.UndefinedEncodeValue),
		reflect.PtrTo(tOID):           ValueEncoderFunc(dve.ObjectIDEncodeValue),
		reflect.PtrTo(tDateTime):      ValueEncoderFunc(dve.DateTimeEncodeValue),
		reflect.PtrTo(tNull):          ValueEncoderFunc(dve.NullEncodeValue),
		reflect.PtrTo(tRegex):         ValueEncoderFunc(dve.RegexEncodeValue),
		reflect.PtrTo(tDBPointer):     ValueEncoderFunc(dve.DBPointerEncodeValue),
		reflect.PtrTo(tCodeWithScope): ValueEncoderFunc(dve.CodeWithScopeEncodeValue),
		reflect.PtrTo(tTimestamp):     ValueEncoderFunc(dve.TimestampEncodeValue),
		reflect.PtrTo(tDecimal):       ValueEncoderFunc(dve.Decimal128EncodeValue),
		reflect.PtrTo(tMinKey):        ValueEncoderFunc(dve.MinKeyEncodeValue),
		reflect.PtrTo(tMaxKey):        ValueEncoderFunc(dve.MaxKeyEncodeValue),
		reflect.PtrTo(tJSONNumber):    ValueEncoderFunc(dve.JSONNumberEncodeValue),
		reflect.PtrTo(tURL):           ValueEncoderFunc(dve.URLEncodeValue),
		reflect.PtrTo(tReader):        ValueEncoderFunc(dve.ReaderEncodeValue),
	}

	typeDecoders := map[reflect.Type]ValueDecoder{
		tDocument:                     ValueDecoderFunc(dvd.DocumentDecodeValue),
		tArray:                        ValueDecoderFunc(dvd.ArrayDecodeValue),
		tValue:                        ValueDecoderFunc(dvd.ValueDecodeValue),
		reflect.PtrTo(tByteSlice):     ValueDecoderFunc(dvd.ByteSliceDecodeValue),
		reflect.PtrTo(tElementSlice):  ValueDecoderFunc(dvd.ElementSliceDecodeValue),
		reflect.PtrTo(tTime):          ValueDecoderFunc(dvd.TimeDecodeValue),
		reflect.PtrTo(tEmpty):         ValueDecoderFunc(dvd.EmptyInterfaceDecodeValue),
		reflect.PtrTo(tBinary):        ValueDecoderFunc(dvd.BooleanDecodeValue),
		reflect.PtrTo(tUndefined):     ValueDecoderFunc(dvd.UndefinedDecodeValue),
		reflect.PtrTo(tOID):           ValueDecoderFunc(dvd.ObjectIDDecodeValue),
		reflect.PtrTo(tDateTime):      ValueDecoderFunc(dvd.DateTimeDecodeValue),
		reflect.PtrTo(tNull):          ValueDecoderFunc(dvd.NullDecodeValue),
		reflect.PtrTo(tRegex):         ValueDecoderFunc(dvd.RegexDecodeValue),
		reflect.PtrTo(tDBPointer):     ValueDecoderFunc(dvd.DBPointerDecodeValue),
		reflect.PtrTo(tCodeWithScope): ValueDecoderFunc(dvd.CodeWithScopeDecodeValue),
		reflect.PtrTo(tTimestamp):     ValueDecoderFunc(dvd.TimestampDecodeValue),
		reflect.PtrTo(tDecimal):       ValueDecoderFunc(dvd.Decimal128DecodeValue),
		reflect.PtrTo(tMinKey):        ValueDecoderFunc(dvd.MinKeyDecodeValue),
		reflect.PtrTo(tMaxKey):        ValueDecoderFunc(dvd.MaxKeyDecodeValue),
		reflect.PtrTo(tJSONNumber):    ValueDecoderFunc(dvd.JSONNumberDecodeValue),
		reflect.PtrTo(tURL):           ValueDecoderFunc(dvd.URLDecodeValue),
		reflect.PtrTo(tReader):        ValueDecoderFunc(dvd.ReaderDecodeValue),
	}

	interfaceEncoders := []interfaceValueEncoder{
		{i: tValueMarshaler, ve: ValueEncoderFunc(dve.ValueMarshalerEncodeValue)},
	}

	interfaceDecoders := []interfaceValueDecoder{
		{i: tValueUnmarshaler, vd: ValueDecoderFunc(dvd.ValueUnmarshalerDecodeValue)},
	}

	kindEncoders := map[reflect.Kind]ValueEncoder{
		reflect.Bool:    ValueEncoderFunc(dve.BooleanEncodeValue),
		reflect.Int:     ValueEncoderFunc(dve.IntEncodeValue),
		reflect.Int8:    ValueEncoderFunc(dve.IntEncodeValue),
		reflect.Int16:   ValueEncoderFunc(dve.IntEncodeValue),
		reflect.Int32:   ValueEncoderFunc(dve.IntEncodeValue),
		reflect.Int64:   ValueEncoderFunc(dve.IntEncodeValue),
		reflect.Uint:    ValueEncoderFunc(dve.UintEncodeValue),
		reflect.Uint8:   ValueEncoderFunc(dve.UintEncodeValue),
		reflect.Uint16:  ValueEncoderFunc(dve.UintEncodeValue),
		reflect.Uint32:  ValueEncoderFunc(dve.UintEncodeValue),
		reflect.Uint64:  ValueEncoderFunc(dve.UintEncodeValue),
		reflect.Float32: ValueEncoderFunc(dve.FloatEncodeValue),
		reflect.Float64: ValueEncoderFunc(dve.FloatEncodeValue),
		reflect.Array:   ValueEncoderFunc(dve.SliceEncodeValue),
		reflect.Map:     ValueEncoderFunc(dve.MapEncodeValue),
		reflect.Slice:   ValueEncoderFunc(dve.SliceEncodeValue),
		reflect.String:  ValueEncoderFunc(dve.StringEncodeValue),
		reflect.Struct:  &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser},
	}

	kindDecoders := map[reflect.Kind]ValueDecoder{
		reflect.Bool:    ValueDecoderFunc(dvd.BooleanDecodeValue),
		reflect.Int:     ValueDecoderFunc(dvd.IntDecodeValue),
		reflect.Int8:    ValueDecoderFunc(dvd.IntDecodeValue),
		reflect.Int16:   ValueDecoderFunc(dvd.IntDecodeValue),
		reflect.Int32:   ValueDecoderFunc(dvd.IntDecodeValue),
		reflect.Int64:   ValueDecoderFunc(dvd.IntDecodeValue),
		reflect.Uint:    ValueDecoderFunc(dvd.UintDecodeValue),
		reflect.Uint8:   ValueDecoderFunc(dvd.UintDecodeValue),
		reflect.Uint16:  ValueDecoderFunc(dvd.UintDecodeValue),
		reflect.Uint32:  ValueDecoderFunc(dvd.UintDecodeValue),
		reflect.Uint64:  ValueDecoderFunc(dvd.UintDecodeValue),
		reflect.Float32: ValueDecoderFunc(dvd.FloatDecodeValue),
		reflect.Float64: ValueDecoderFunc(dvd.FloatDecodeValue),
		reflect.Array:   ValueDecoderFunc(dvd.SliceDecodeValue),
		reflect.Map:     ValueDecoderFunc(dvd.MapDecodeValue),
		reflect.Slice:   ValueDecoderFunc(dvd.SliceDecodeValue),
		reflect.String:  ValueDecoderFunc(dvd.StringDecodeValue),
		reflect.Struct:  &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser},
	}

	return &RegistryBuilder{
		typeEncoders:      typeEncoders,
		typeDecoders:      typeDecoders,
		interfaceEncoders: interfaceEncoders,
		interfaceDecoders: interfaceDecoders,
		kindEncoders:      kindEncoders,
		kindDecoders:      kindDecoders,
	}
}

// RegisterCodec will register the provided ValueCodec for the provided type.
func (rb *RegistryBuilder) RegisterCodec(t reflect.Type, codec ValueCodec) *RegistryBuilder {
	rb.RegisterEncoder(t, codec)
	rb.RegisterDecoder(t, codec)
	return rb
}

// RegisterEncoder will register the provided ValueEncoder to the provided type.
func (rb *RegistryBuilder) RegisterEncoder(t reflect.Type, enc ValueEncoder) *RegistryBuilder {
	switch t.Kind() {
	case reflect.Interface:
		for idx, ir := range rb.interfaceEncoders {
			if ir.i == t {
				rb.interfaceEncoders[idx].ve = enc
				return rb
			}
		}

		rb.interfaceEncoders = append(rb.interfaceEncoders, interfaceValueEncoder{i: t, ve: enc})
	default:
		if t.Kind() != reflect.Ptr {
			t = reflect.PtrTo(t)
		}
		rb.typeEncoders[t] = enc
	}
	return rb
}

// RegisterDecoder will register the provided ValueDecoder to the provided type.
func (rb *RegistryBuilder) RegisterDecoder(t reflect.Type, dec ValueDecoder) *RegistryBuilder {
	switch t.Kind() {
	case reflect.Interface:
		for idx, ir := range rb.interfaceDecoders {
			if ir.i == t {
				rb.interfaceDecoders[idx].vd = dec
				return rb
			}
		}

		rb.interfaceDecoders = append(rb.interfaceDecoders, interfaceValueDecoder{i: t, vd: dec})
	default:
		if t.Kind() != reflect.Ptr {
			t = reflect.PtrTo(t)
		}
		rb.typeDecoders[t] = dec
	}
	return rb
}

// RegisterDefaultEncoder will registr the provided ValueEncoder to the provided
// kind.
func (rb *RegistryBuilder) RegisterDefaultEncoder(kind reflect.Kind, enc ValueEncoder) *RegistryBuilder {
	rb.kindEncoders[kind] = enc
	return rb
}

// RegisterDefaultDecoder will register the provided ValueDecoder to the
// provided kind.
func (rb *RegistryBuilder) RegisterDefaultDecoder(kind reflect.Kind, dec ValueDecoder) *RegistryBuilder {
	rb.kindDecoders[kind] = dec
	return rb
}

// Build creates a Registry from the current state of this RegistryBuilder.
func (rb *RegistryBuilder) Build() *Registry {
	registry := new(Registry)

	registry.typeEncoders = make(map[reflect.Type]ValueEncoder)
	for t, enc := range rb.typeEncoders {
		registry.typeEncoders[t] = enc
	}

	registry.typeDecoders = make(map[reflect.Type]ValueDecoder)
	for t, dec := range rb.typeDecoders {
		registry.typeDecoders[t] = dec
	}

	registry.interfaceEncoders = make([]interfaceValueEncoder, len(rb.interfaceEncoders))
	copy(registry.interfaceEncoders, rb.interfaceEncoders)

	registry.interfaceDecoders = make([]interfaceValueDecoder, len(rb.interfaceDecoders))
	copy(registry.interfaceDecoders, rb.interfaceDecoders)

	registry.kindEncoders = make(map[reflect.Kind]ValueEncoder)
	for kind, enc := range rb.kindEncoders {
		registry.kindEncoders[kind] = enc
	}

	registry.kindDecoders = make(map[reflect.Kind]ValueDecoder)
	for kind, dec := range rb.kindDecoders {
		registry.kindDecoders[kind] = dec
	}

	return registry
}

// LookupEncoder will inspect the registry for an encoder that satisfies the
// type provided. An encoder registered for a specific type will take
// precedence over an encoder registered for an interface the type satisfies,
// which takes precedence over an encoder for the reflect.Kind of the value. If
// no encoder can be found, an error is returned.
func (r *Registry) LookupEncoder(t reflect.Type) (ValueEncoder, error) {
	if t == nil {
		return nil, ErrNilType
	}
	encodererr := ErrNoEncoder{Type: t}
	r.mu.RLock()
	enc, found := r.lookupTypeEncoder(t)
	r.mu.RUnlock()
	if found {
		if enc == nil {
			return nil, ErrNoEncoder{Type: t}
		}
		return enc, nil
	}

	enc, found = r.lookupInterfaceEncoder(t)
	if found {
		r.mu.Lock()
		if t.Kind() != reflect.Ptr {
			t = reflect.PtrTo(t)
		}
		r.typeEncoders[t] = enc
		r.mu.Unlock()
		return enc, nil
	}

	if t.Kind() == reflect.Map && t.Key().Kind() != reflect.String {
		r.mu.Lock()
		r.typeEncoders[t] = nil
		r.mu.Unlock()
		return nil, encodererr
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	enc, found = r.kindEncoders[t.Kind()]
	if !found {
		r.mu.Lock()
		r.typeEncoders[t] = nil
		r.mu.Unlock()
		return nil, encodererr
	}

	r.mu.Lock()
	r.typeEncoders[t] = enc
	r.mu.Unlock()
	return enc, nil
}

func (r *Registry) lookupTypeEncoder(t reflect.Type) (ValueEncoder, bool) {
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}

	enc, found := r.typeEncoders[t]
	return enc, found
}

func (r *Registry) lookupInterfaceEncoder(t reflect.Type) (ValueEncoder, bool) {
	for _, ienc := range r.interfaceEncoders {
		if !t.Implements(ienc.i) {
			continue
		}

		return ienc.ve, true
	}
	return nil, false
}

// LookupDecoder will inspect the registry for a decoder that satisfies the
// type provided. A decoder registered for a specific type will take
// precedence over a decoder registered for an interface the type satisfies,
// which takes precedence over a decoder for the reflect.Kind of the value. If
// no decoder can be found, an error is returned.
func (r *Registry) LookupDecoder(t reflect.Type) (ValueDecoder, error) {
	if t == nil {
		return nil, ErrNilType
	}
	decodererr := ErrNoDecoder{Type: t}
	r.mu.RLock()
	dec, found := r.lookupTypeDecoder(t)
	r.mu.RUnlock()
	if found {
		if dec == nil {
			return nil, ErrNoDecoder{Type: t}
		}
		return dec, nil
	}

	dec, found = r.lookupInterfaceDecoder(t)
	if found {
		r.mu.Lock()
		if t.Kind() != reflect.Ptr {
			t = reflect.PtrTo(t)
		}
		r.typeDecoders[t] = dec
		r.mu.Unlock()
		return dec, nil
	}

	if t.Kind() == reflect.Map && t.Key().Kind() != reflect.String {
		r.mu.Lock()
		r.typeDecoders[t] = nil
		r.mu.Unlock()
		return nil, decodererr
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	dec, found = r.kindDecoders[t.Kind()]
	if !found {
		r.mu.Lock()
		r.typeDecoders[t] = nil
		r.mu.Unlock()
		return nil, decodererr
	}

	r.mu.Lock()
	r.typeDecoders[t] = dec
	r.mu.Unlock()
	return dec, nil
}

func (r *Registry) lookupTypeDecoder(t reflect.Type) (ValueDecoder, bool) {
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}

	dec, found := r.typeDecoders[t]
	return dec, found
}

func (r *Registry) lookupInterfaceDecoder(t reflect.Type) (ValueDecoder, bool) {
	for _, idec := range r.interfaceDecoders {
		if !t.Implements(idec.i) {
			continue
		}

		return idec.vd, true
	}
	return nil, false
}

type interfaceValueEncoder struct {
	i  reflect.Type
	ve ValueEncoder
}

type interfaceValueDecoder struct {
	i  reflect.Type
	vd ValueDecoder
}
