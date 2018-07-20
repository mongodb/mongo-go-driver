package bson

import (
	"errors"
	"reflect"
	"sync"
)

// ErrNoCodec is returned when there is no codec available for a type or interface in the registry.
type ErrNoCodec struct {
	Type reflect.Type
}

func (enc ErrNoCodec) Error() string {
	return "no codec found for " + enc.Type.String()
}

// ErrNotInterface is returned when the provided type is not an interface.
var ErrNotInterface = errors.New("The provided typeis not an interface")

var defaultRegistry = NewRegistryBuilder().Build()

// ErrFrozenRegistry is returned when an attempt to mutate a frozen Registry is
// made. A Registry is considered frozen when a call to Lookup has been made.
var ErrFrozenRegistry = errors.New("the Registry has been frozen and can no longer be modified")

// A RegistryBuilder is used to build a Registry. This type is not goroutine
// safe.
type RegistryBuilder struct {
	tr map[reflect.Type]Codec
	ir []interfacePair

	m   Codec // map codec
	s   Codec // struct codec
	slc Codec // slice & array codec
}

// A Registry is used to store and retrieve codecs for types and interfaces. This type is the main
// typed passed around and Encoders and Decoders are constructed from it.
//
// TODO: Create a RegistryBuilder type and make the Registry type immutable.
type Registry struct {
	tr       typeRegistry
	ir       interfaceRegistry
	ircache  map[reflect.Type]Codec
	ircacheL sync.RWMutex

	dm   Codec // default map codec
	ds   Codec // default struct codec
	dslc Codec // default slice & array codec
}

// NewRegistryBuilder creates a new RegistryBuilder.
func NewRegistryBuilder() *RegistryBuilder {
	// TODO: Register codecs.
	tr := map[reflect.Type]Codec{reflect.PtrTo(reflect.TypeOf(false)): new(BooleanCodec)}

	return &RegistryBuilder{
		tr:  tr,
		ir:  make([]interfacePair, 0),
		s:   defaultStructCodec,
		slc: defaultSliceCodec,
	}
}

// Register will register the provided Codec to the provided type. If the type is
// an interface, it will be registered in the interface registry. If the type is
// a pointer to or a type that is not an interface, it will be registered in the type
// registry.
func (r *RegistryBuilder) Register(t reflect.Type, codec Codec) *RegistryBuilder {
	switch t.Kind() {
	case reflect.Interface:
		for idx, ip := range r.ir {
			if ip.i == t {
				r.ir[idx].c = codec
				return r
			}
		}

		r.ir = append(r.ir, interfacePair{i: t, c: codec})
	default:
		if t.Kind() != reflect.Ptr {
			t = reflect.PtrTo(t)
		}

		r.tr[t] = codec
	}
	return r
}

// Build creates a Registry from the current state of this RegistryBuilder.
func (rb *RegistryBuilder) Build() *Registry {
	tr := make(typeRegistry)
	for t, c := range rb.tr {
		tr[t] = c
	}
	ir := make(interfaceRegistry, len(rb.ir))
	copy(ir, rb.ir)

	return &Registry{
		tr:      tr,
		ir:      ir,
		ircache: make(map[reflect.Type]Codec),
		dm:      rb.m,
		ds:      rb.s,
		dslc:    rb.slc,
	}
}

// SetDefaultMapCodec will set the Codec used when encoding or decoding a map that does
// not have another codec registered for it.
func (r *RegistryBuilder) SetDefaultMapCodec(codec Codec) *RegistryBuilder {
	r.m = codec
	return r
}

// SetDefaultStructCodec will set the Codec used when encoding or decoding a struct that
// does not have another codec registered for it.
func (r *RegistryBuilder) SetDefaultStructCodec(codec Codec) *RegistryBuilder {
	r.s = codec
	return r
}

// SetDefaultSliceCodec will set the Codec used when encoding or decoding a
// slice or array that does not have another codec registered for it.
func (r *RegistryBuilder) SetDefaultSliceCodec(codec Codec) *RegistryBuilder {
	r.slc = codec
	return r
}

// Lookup will inspect the type registry for either the type or a pointer to the type,
// if it doesn't find a codec it will inspect the interface registry for an interface
// that the type satisfies, if it doesn't find a codec there it will attempt to
// return either the default map codec or the default struct codec. If none of those
// apply, an error will be returned.
func (r *Registry) Lookup(t reflect.Type) (Codec, error) {
	codec, found := r.tr.lookup(t)
	if found {
		return codec, nil
	}

	r.ircacheL.RLock()
	codec, found = r.ircache[t]
	r.ircacheL.RUnlock()
	if found {
		return codec, nil
	}

	codec, found = r.ir.lookup(t)
	if found {
		r.ircacheL.Lock()
		r.ircache[t] = codec
		r.ircacheL.Unlock()
		return codec, nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		return r.ds, nil
	}

	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		return r.dslc, nil
	}

	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		return r.dm, nil
	}

	return nil, ErrNoCodec{Type: t}
}

// The type registry handles codecs that are for specifics types that are not interfaces.
// This registry will handle both the types themselves and pointers to those types.
type typeRegistry map[reflect.Type]Codec

// lookup handles finding a codec for the registered type. Will return an error if no codec
// could be found.
func (tr typeRegistry) lookup(t reflect.Type) (Codec, bool) {
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}

	codec, found := tr[t]
	return codec, found
}

type interfacePair struct {
	i reflect.Type
	c Codec
}

// The interface registry handles codecs that are for interface types.
type interfaceRegistry []interfacePair

// lookup handles finding a codec for the registered interface. Will return an error if no codec
// could be found.
func (ir interfaceRegistry) lookup(t reflect.Type) (Codec, bool) {
	for _, ip := range ir {
		if !t.Implements(ip.i) {
			continue
		}

		return ip.c, true
	}
	return nil, false
}
